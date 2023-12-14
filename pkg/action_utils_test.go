package pkg

import (
	"fmt"
	"testing"
)

func TestSplitActionString(t *testing.T) {
	tests := []struct {
		action      string
		delimiter   string
		expected1   string
		expected2   string
		expectedErr error
	}{
		{"owner/repo@v3", "@v", "owner/repo", "3", nil},
		{"owner/repo@main", "@", "owner/repo", "main", nil},
		{"owner/repo@v3.3.3", "@v", "owner/repo", "3.3.3", nil},
		{"repo", "/", "", "", fmt.Errorf("invalid action format: repo")},
		{"repo", "|", "", "", fmt.Errorf("invalid action format: repo")},
	}

	for _, test := range tests {
		result1, result2, err := SplitActionString(test.action, test.delimiter)
		if err != nil && test.expectedErr == nil {
			t.Errorf("Unexpected error for action %s: %v", test.action, err)
		} else if err == nil && test.expectedErr != nil {
			t.Errorf("Expected error for action %s: %v", test.action, test.expectedErr)
		} else if result1 != test.expected1 || result2 != test.expected2 {
			t.Errorf("Unexpected result for action %s: got (%s, %s), want (%s, %s)", test.action, result1, result2, test.expected1, test.expected2)
		}
	}
}
func TestExtractOwnerRepo(t *testing.T) {
	tests := []struct {
		repository string
		expected   string
	}{
		{"owner/repo", "owner/repo"},
		{"owner/repo/sub", "owner/repo"},
		{"owner/repo/sub/sub", "owner/repo"},
		{"repo", "repo"},
		{"", ""},
	}

	for _, test := range tests {
		result := ExtractOwnerRepo(test.repository)
		if result != test.expected {
			t.Errorf("Unexpected result for repository %s: got %s, want %s", test.repository, result, test.expected)
		}
	}
}
