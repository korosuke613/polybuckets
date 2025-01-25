package s3client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
)

// MockS3Client implements S3Client interface for testing
type MockS3Client struct {
	listBucketsOutput *s3.ListBucketsOutput
	listBucketsError  error
	listObjectsOutput *s3.ListObjectsV2Output
	listObjectsError  error
	getObjectOutput   *s3.GetObjectOutput
	getObjectError    error
}

// ListBuckets mocks the ListBuckets method of S3Client
func (m *MockS3Client) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	return m.listBucketsOutput, m.listBucketsError
}

// ListObjectsV2 mocks the ListObjectsV2 method of S3Client
func (m *MockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return m.listObjectsOutput, m.listObjectsError
}

// GetObject mocks the GetObject method of S3Client
func (m *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.getObjectOutput, m.getObjectError
}

// TestClient_ListBuckets tests the ListBuckets method of Client
func TestClient_ListBuckets(t *testing.T) {
	mockTime := time.Now()

	tests := []struct {
		name        string
		mock        *MockS3Client
		expected    []BucketInfo
		expectedErr string
	}{
		{
			name: "正常系: バケットリスト取得",
			mock: &MockS3Client{
				listBucketsOutput: &s3.ListBucketsOutput{
					Buckets: []types.Bucket{
						{
							Name:         aws.String("bucket1"),
							CreationDate: &mockTime,
						},
						{
							Name:         aws.String("bucket2"),
							CreationDate: &mockTime,
						},
					},
				},
			},
			expected: []BucketInfo{
				{Name: "bucket1", CreationDate: mockTime},
				{Name: "bucket2", CreationDate: mockTime},
			},
		},
		{
			name: "異常系: バケットリスト取得失敗",
			mock: &MockS3Client{
				listBucketsError: errors.New("connection error"),
			},
			expectedErr: "ListBuckets operation failed: connection error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{s3Client: tt.mock}
			result, err := client.ListBuckets(context.Background())

			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestClient_ListObjects tests the ListObjects method of Client
func TestClient_ListObjects(t *testing.T) {
	mockTime := time.Now()

	tests := []struct {
		name        string
		bucket      string
		prefix      string
		mock        *MockS3Client
		expected    []ObjectInfo
		expectedErr string
	}{
		{
			name:   "正常系: オブジェクトリスト取得",
			bucket: "test-bucket",
			prefix: "test/prefix/",
			mock: &MockS3Client{
				listObjectsOutput: &s3.ListObjectsV2Output{
					CommonPrefixes: []types.CommonPrefix{
						{Prefix: aws.String("test/prefix/dir1/")},
						{Prefix: aws.String("test/prefix/dir2/")},
					},
					Contents: []types.Object{
						{
							Key:          aws.String("test/prefix/file1.txt"),
							Size:         aws.Int64(1024),
							LastModified: &mockTime,
						},
					},
				},
			},
			expected: []ObjectInfo{
				{
					Name:        "test/prefix/dir1/",
					ShortName:   "dir1/",
					IsDirectory: true,
				},
				{
					Name:        "test/prefix/dir2/",
					ShortName:   "dir2/",
					IsDirectory: true,
				},
				{
					Name:         "test/prefix/file1.txt",
					ShortName:    "file1.txt",
					IsDirectory:  false,
					Size:         "1.0 KB",
					LastModified: mockTime,
				},
			},
		},
		{
			name:   "異常系: オブジェクトリスト取得失敗",
			bucket: "invalid-bucket",
			prefix: "invalid/prefix/",
			mock: &MockS3Client{
				listObjectsError: errors.New("access denied"),
			},
			expectedErr: "ListObjectsV2 operation failed for bucket \"invalid-bucket\": access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{s3Client: tt.mock}
			result, err := client.ListObjects(context.Background(), tt.bucket, tt.prefix)

			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestClient_GetObject tests the GetObject method of Client
func TestClient_GetObject(t *testing.T) {
	tests := []struct {
		name        string
		bucket      string
		key         string
		mock        *MockS3Client
		expectedErr string
	}{
		{
			name:   "正常系: オブジェクト取得",
			bucket: "test-bucket",
			key:    "test-key",
			mock: &MockS3Client{
				getObjectOutput: &s3.GetObjectOutput{},
			},
		},
		{
			name:   "異常系: オブジェクト取得失敗",
			bucket: "invalid-bucket",
			key:    "invalid-key",
			mock: &MockS3Client{
				getObjectError: errors.New("not found"),
			},
			expectedErr: "GetObject failed for bucket \"invalid-bucket\" key \"invalid-key\": not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{s3Client: tt.mock}
			_, err := client.GetObject(context.Background(), tt.bucket, tt.key)

			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}

			assert.NoError(t, err)
		})
	}
}

// TestFormatSize tests the formatSize function with various size inputs.
func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0.0 B"},
		{500, "500.0 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
