package pkg

import (
	"fmt"
	"regexp"
	"strings"
)

func SplitActionString(action string, delimiter string) (string, string, error) {
	actionSplit := strings.Split(action, delimiter)
	if len(actionSplit) < 2 {
		return "", "", fmt.Errorf("invalid action format: %s", action)
	}
	repoWithOwner := actionSplit[0]
	branchOrVersion := actionSplit[1]
	return repoWithOwner, branchOrVersion, nil
}

func ExtractOwnerRepo(repository string) string {
	if strings.Count(repository, "/") > 1 {
		parts := strings.Split(repository, "/")
		if len(parts) > 2 {
			repository = parts[0] + "/" + parts[1]
		}
	}
	return repository
}

// RepoFromPinnedRef returns the owner/repo(/sub) prefix of a SHA-pinned action ref
// (e.g. "owner/repo/sub@<sha>" -> "owner/repo/sub").
func RepoFromPinnedRef(action string) (string, error) {
	repoWithOwner, _, err := SplitActionString(action, "@")
	if err != nil {
		return "", err
	}
	return repoWithOwner, nil
}

// ReplaceActionRef replaces the first occurrence of a SHA-pinned action ref (and its optional
// trailing comment) in content with replacement, returning the updated content and whether a match
// was found. Splice replacement avoids regexp `$` expansion and guarantees exactly one occurrence is
// replaced; the optional `[ \t]+#[^\r\n]*` group is CRLF/tab-safe so it consumes a stale comment
// without disturbing line endings.
func ReplaceActionRef(content, action, replacement string) (string, bool) {
	re := regexp.MustCompile(regexp.QuoteMeta(action) + `([ \t]+#[^\r\n]*)?`)
	loc := re.FindStringIndex(content)
	if loc == nil {
		return content, false
	}
	return content[:loc[0]] + replacement + content[loc[1]:], true
}
