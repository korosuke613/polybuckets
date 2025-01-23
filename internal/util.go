package internal

import (
	"strings"
)

func joinPath(parts []string) string {
	return strings.Join(parts, "/")
}

func ParsePath(path string) (string, string, string) {
	// Remove leading slash and split path
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "", "", ""
	}
	bucket := parts[0]
	parentPrefix := ""
	if len(parts) > 1 {
		parentPrefix = joinPath(parts[1 : len(parts)-1])
	}
	prefix := joinPath(parts[1:])

	return bucket, parentPrefix, prefix
}
