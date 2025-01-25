package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParsePath tests the ParsePath function with various path inputs.
func TestParsePath(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectedBucket string
		expectedParent string
		expectedPrefix string
	}{
		{
			name:          "空のパス",
			path:          "",
			expectedBucket: "",
			expectedParent: "",
			expectedPrefix: "",
		},
		{
			name:          "バケットのみ",
			path:          "my-bucket",
			expectedBucket: "my-bucket",
			expectedParent: "",
			expectedPrefix: "",
		},
		{
			name:          "バケットとディレクトリ",
			path:          "my-bucket/dir1/dir2",
			expectedBucket: "my-bucket",
			expectedParent: "dir1",
			expectedPrefix: "dir1/dir2",
		},
		{
			name:          "先頭と末尾のスラッシュをトリム",
			path:          "/my-bucket/dir1/",
			expectedBucket: "my-bucket",
			expectedParent: "",
			expectedPrefix: "dir1",
		},
		{
			name:          "複数階層のディレクトリ",
			path:          "my-bucket/parent/current/child",
			expectedBucket: "my-bucket",
			expectedParent: "parent/current",
			expectedPrefix: "parent/current/child",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, parent, prefix := ParsePath(tt.path)
			assert.Equal(t, tt.expectedBucket, bucket)
			assert.Equal(t, tt.expectedParent, parent)
			assert.Equal(t, tt.expectedPrefix, prefix)
		})
	}
}

// TestJoinPath tests the joinPath function with various parts inputs.
func TestJoinPath(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{
			name:     "空のスライス",
			parts:    []string{},
			expected: "",
		},
		{
			name:     "単一要素",
			parts:    []string{"dir"},
			expected: "dir",
		},
		{
			name:     "複数要素",
			parts:    []string{"parent", "child", "file.txt"},
			expected: "parent/child/file.txt",
		},
		{
			name:     "空文字列を含む要素",
			parts:    []string{"", "dir", ""},
			expected: "/dir/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinPath(tt.parts)
			assert.Equal(t, tt.expected, result)
		})
	}
}
