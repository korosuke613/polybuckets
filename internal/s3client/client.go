package s3client

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/korosuke613/polybuckets/internal/env"
)

// S3Client defines the operations required for interacting with S3.
type S3Client interface {
	ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// Client wraps the S3 client and provides additional functionality.
type Client struct {
	s3Client              S3Client
	CacheDuration         time.Duration
	listObjectsCacheEntry map[string]ListObjectsCacheEntry
}

// ClientOption defines a function type for configuring the Client.
type ClientOption func(*Client) error

// NewClient creates a new S3 client with the provided options.
func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	pbConfig := env.PBConfig
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(pbConfig.AWSRegion),
		config.WithSharedConfigProfile(pbConfig.AWSProfile),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := &Client{
		s3Client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			// Suppress warnings about checksum validation skipped in log output
			// e.g. SDK 2025/01/26 02:05:17 WARN Response has no supported checksum. Not validating response payload.
			o.DisableLogOutputChecksumValidationSkipped = true

			// Use the specified endpoint if set, and enforce path style
			if pbConfig.AWSEndpoint != "" {
				// if endpoint without http/https is specified, add https
				endpoint := pbConfig.AWSEndpoint 
				if !strings.HasPrefix(pbConfig.AWSEndpoint, "http") {
					endpoint = "http://" + endpoint
				}

				o.BaseEndpoint = aws.String(endpoint)
				o.UsePathStyle = true
			}
		}),
		listObjectsCacheEntry: make(map[string]ListObjectsCacheEntry),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("failed to apply client option: %w", err)
		}
	}

	return client, nil
}

// WithCustomClient injects a custom S3 client.
func WithCustomClient(cli S3Client) ClientOption {
	return func(c *Client) error {
		c.s3Client = cli
		return nil
	}
}

// WithConfig customizes the AWS configuration.
func WithConfig(cfg aws.Config) ClientOption {
	return func(c *Client) error {
		c.s3Client = s3.NewFromConfig(cfg)
		return nil
	}
}

// BucketInfo contains information about an S3 bucket.
type BucketInfo struct {
	Name         string
	CreationDate time.Time
}

// ListObjectsCacheEntry contains the data and expiry time for a listObjects cache entry.
type ListObjectsCacheEntry struct {
	data   *s3.ListObjectsV2Output
	Expiry time.Time
}

// ClearListObjectsCache clears the listObjects cache for the specified bucket and prefix.
func (c *Client) ClearListObjectsCache(ctx context.Context, bucket, prefix string) {
	cacheKey := fmt.Sprintf("%s/%s", bucket, prefix)
	delete(c.listObjectsCacheEntry, cacheKey)
}

// GetListObjectsCacheEntry retrieves the listObjects cache entry for the specified bucket and prefix.
func (c *Client) GetListObjectsCacheEntry(ctx context.Context, bucket, prefix string) *ListObjectsCacheEntry {
	cacheKey := fmt.Sprintf("%s/%s", bucket, prefix)
	entry, found := c.listObjectsCacheEntry[cacheKey]
	if !found {
		return nil
	}
	return &entry
}

// ClearOldListObjectsCache clears the listObjects cache for entries that have expired.
func (c *Client) ClearOldListObjectsCache(ctx context.Context) {
	now := time.Now()
	for key, entry := range c.listObjectsCacheEntry {
		if entry.Expiry.Before(now) {
			delete(c.listObjectsCacheEntry, key)
		}
	}
}

// ListBuckets lists all S3 buckets.
func (c *Client) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	result, err := c.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("ListBuckets operation failed: %w", err)
	}

	buckets := make([]BucketInfo, len(result.Buckets))
	for i, b := range result.Buckets {
		buckets[i] = BucketInfo{
			Name:         *b.Name,
			CreationDate: *b.CreationDate,
		}
	}
	return buckets, nil
}

// ObjectInfo contains information about an S3 object.
type ObjectInfo struct {
	Name         string
	ShortName    string
	IsDirectory  bool
	Size         string
	LastModified time.Time
}

// ListObjects lists objects in the specified S3 bucket and prefix.
func (c *Client) ListObjects(ctx context.Context, bucket, prefix string) (objectInfo []ObjectInfo, hitCache bool, err error) {
	cacheKey := fmt.Sprintf("%s/%s", bucket, prefix)
	now := time.Now()

	// Add a trailing slash to the prefix if it doesn't already have one
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// if the cache exists and is within the expiration date, return the cache
	if entry, found := c.listObjectsCacheEntry[cacheKey]; found && entry.Expiry.After(now) {
		return convertToObjectInfo(entry.data, prefix), true, nil
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, false, fmt.Errorf("ListObjectsV2 operation failed for bucket %q: %w", bucket, err)
	}

	// Save to cache
	c.listObjectsCacheEntry[cacheKey] = ListObjectsCacheEntry{
		data:   result,
		Expiry: now.Add(c.CacheDuration),
	}

	return convertToObjectInfo(result, prefix), false, nil
}

func convertToObjectInfo(result *s3.ListObjectsV2Output, prefix string) []ObjectInfo {
	var objects []ObjectInfo
	for _, commonPrefix := range result.CommonPrefixes {
		objects = append(objects, ObjectInfo{
			Name:        *commonPrefix.Prefix,
			ShortName:   strings.TrimPrefix(*commonPrefix.Prefix, prefix),
			IsDirectory: true,
		})
	}

	for _, obj := range result.Contents {
		// Skip if the current prefix is the same
		if *obj.Key == prefix {
			continue
		}

		// Convert size to a string with SI prefixes
		size := formatSize(*obj.Size)

		objects = append(objects, ObjectInfo{
			Name:         *obj.Key,
			ShortName:    strings.TrimPrefix(*obj.Key, prefix),
			IsDirectory:  false,
			Size:         size,
			LastModified: *obj.LastModified,
		})
	}
	return objects
}

// GetObject retrieves an object from the specified S3 bucket and key.
func (c *Client) GetObject(ctx context.Context, bucket, key string) (*s3.GetObjectOutput, error) {
	output, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("GetObject failed for bucket %q key %q: %w", bucket, key, err)
	}
	return output, nil
}

// formatSize converts a size in bytes to a human-readable string with SI prefixes.
func formatSize(size int64) string {
	var unit string
	var value float64
	switch {
	case size >= 1<<40:
		value = float64(size) / (1 << 40)
		unit = "TB"
	case size >= 1<<30:
		value = float64(size) / (1 << 30)
		unit = "GB"
	case size >= 1<<20:
		value = float64(size) / (1 << 20)
		unit = "MB"
	case size >= 1<<10:
		value = float64(size) / (1 << 10)
		unit = "KB"
	default:
		value = float64(size)
		unit = "B"
	}
	return fmt.Sprintf("%s %s", strconv.FormatFloat(value, 'f', 1, 64), unit)
}
