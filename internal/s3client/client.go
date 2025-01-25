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

// S3Clientインターフェースで必要な操作を定義
type S3Client interface {
	ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type Client struct {
	s3Client S3Client
}

// Client設定オプション用の関数型
type ClientOption func(*Client) error

// NewClient コンストラクタでオプションを受け取る
func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	pbConfig := env.LoadPBConfig()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(pbConfig.AWSRegion),
		config.WithSharedConfigProfile(pbConfig.AWSProfile),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := &Client{
		s3Client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			// ログ出力に checksum validation skipped の警告が出るのを抑制
			// e.g. SDK 2025/01/26 02:05:17 WARN Response has no supported checksum. Not validating response payload.
			o.DisableLogOutputChecksumValidationSkipped = true

			// エンドポイントが設定されている場合は、そのエンドポイントを使用する。パススタイルを強制
			if pbConfig.AWSEndpoint != "" {
				o.BaseEndpoint = aws.String(pbConfig.AWSEndpoint)
				o.UsePathStyle = true
			}
		}),
	}

	// オプション適用
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("failed to apply client option: %w", err)
		}
	}

	return client, nil
}

// WithCustomClient カスタムS3クライアントを注入するオプション
func WithCustomClient(cli S3Client) ClientOption {
	return func(c *Client) error {
		c.s3Client = cli
		return nil
	}
}

// WithConfig AWS設定をカスタマイズするオプション
func WithConfig(cfg aws.Config) ClientOption {
	return func(c *Client) error {
		c.s3Client = s3.NewFromConfig(cfg)
		return nil
	}
}

type BucketInfo struct {
	Name         string
	CreationDate time.Time
}

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

type ObjectInfo struct {
	Name         string
	ShortName    string
	IsDirectory  bool
	Size         string
	LastModified time.Time
}

func (c *Client) ListObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	// もし prefix の末尾に / がない場合は付与する
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("ListObjectsV2 operation failed for bucket %q: %w", bucket, err)
	}

	var objects []ObjectInfo
	for _, commonPrefix := range result.CommonPrefixes {
		objects = append(objects, ObjectInfo{
			Name:        *commonPrefix.Prefix,
			ShortName:   strings.TrimPrefix(*commonPrefix.Prefix, prefix),
			IsDirectory: true,
		})
	}

	for _, obj := range result.Contents {
		// もし現在の prefix と同じ場合はスキップ
		if *obj.Key == prefix {
			continue
		}

		// size を SI 接頭辞付きの文字列に変換
		size := formatSize(*obj.Size)

		objects = append(objects, ObjectInfo{
			Name:         *obj.Key,
			ShortName:    strings.TrimPrefix(*obj.Key, prefix),
			IsDirectory:  false,
			Size:         size,
			LastModified: *obj.LastModified,
		})
	}

	return objects, nil
}

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

// 数値を SI 接頭辞付きの文字列に変換
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
