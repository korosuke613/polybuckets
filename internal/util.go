package internal

import (
	"strings"
)

// joinPath joins the given parts into a single path string.
func joinPath(parts []string) string {
	return strings.Join(parts, "/")
}

// ParsePath parses the given path into bucket, parent prefix, and prefix.
func ParsePath(path string) (string, string, string) {
	// Remove leading slash and split path
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	// Split the path by slashes and trim any leading/trailing slashes
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "", "", ""
	}
	// The first part is the bucket name
	bucket := parts[0]
	parentPrefix := ""
	// If there are more parts, join them to form the parent prefix
	if len(parts) > 1 {
		parentPrefix = joinPath(parts[1 : len(parts)-1])
	}
	// Join all parts except the first to form the prefix
	prefix := joinPath(parts[1:])

	return bucket, parentPrefix, prefix
}
