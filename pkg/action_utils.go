package pkg

import (
	"strings"
)

func splitOnLastAt(value string) (string, string, bool) {
	lastAt := strings.LastIndex(value, "@")
	if lastAt == -1 {
		return value, "", false
	}

	return value[:lastAt], value[lastAt+1:], true
}

// ExtractOwnerRepo extracts the "owner/repo" base from an action path
// that may include subpaths (e.g., "owner/repo/sub/path" → "owner/repo").
// Returns false if the path doesn't contain a valid owner/repo pair.
func ExtractOwnerRepo(actionPath string) (string, bool) {
	parts := strings.Split(actionPath, "/")
	if len(parts) < 2 {
		return "", false
	}

	owner := parts[0]
	repo := parts[1]
	if owner == "" || repo == "" {
		return "", false
	}

	return owner + "/" + repo, true
}
