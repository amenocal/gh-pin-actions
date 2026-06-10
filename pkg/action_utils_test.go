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

func TestRepoFromPinnedRef(t *testing.T) {
	const sha = "1234567890abcdef1234567890abcdef12345678"
	tests := []struct {
		name    string
		action  string
		want    string
		wantErr bool
	}{
		{name: "simple action", action: "actions/checkout@" + sha, want: "actions/checkout"},
		{name: "sub-path action", action: "docker/build-push-action/sub@" + sha, want: "docker/build-push-action/sub"},
		{name: "malformed no at-sign", action: "noatsign", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RepoFromPinnedRef(tt.action)
			if tt.wantErr {
				if err == nil {
					t.Errorf("RepoFromPinnedRef(%q) expected error, got nil", tt.action)
				}
				return
			}
			if err != nil {
				t.Fatalf("RepoFromPinnedRef(%q) unexpected error: %v", tt.action, err)
			}
			if got != tt.want {
				t.Errorf("RepoFromPinnedRef(%q) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

func TestReplaceActionRef(t *testing.T) {
	const (
		shaA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		shaB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		shaX = "cccccccccccccccccccccccccccccccccccccccc"
	)
	tests := []struct {
		name        string
		content     string
		action      string
		replacement string
		want        string
		wantMatched bool
	}{
		{
			name:        "ref with trailing comment",
			content:     "      - uses: actions/checkout@" + shaA + " # v4.1.1\n",
			action:      "actions/checkout@" + shaA,
			replacement: "actions/checkout@" + shaB + " #v4.2.2",
			want:        "      - uses: actions/checkout@" + shaB + " #v4.2.2\n",
			wantMatched: true,
		},
		{
			name:        "ref without comment",
			content:     "      - uses: actions/checkout@" + shaA + "\n",
			action:      "actions/checkout@" + shaA,
			replacement: "actions/checkout@" + shaB + " #v4.2.2",
			want:        "      - uses: actions/checkout@" + shaB + " #v4.2.2\n",
			wantMatched: true,
		},
		{
			name:        "sub-path action",
			content:     "      - uses: docker/build-push-action/sub@" + shaA + " # v5.0.0\n",
			action:      "docker/build-push-action/sub@" + shaA,
			replacement: "docker/build-push-action/sub@" + shaB + " #v5.1.0",
			want:        "      - uses: docker/build-push-action/sub@" + shaB + " #v5.1.0\n",
			wantMatched: true,
		},
		{
			name:        "CRLF line preserves carriage return",
			content:     "      - uses: actions/checkout@" + shaA + " # v4.1.1\r\n",
			action:      "actions/checkout@" + shaA,
			replacement: "actions/checkout@" + shaB + " #v4.2.2",
			want:        "      - uses: actions/checkout@" + shaB + " #v4.2.2\r\n",
			wantMatched: true,
		},
		{
			name: "duplicate refs only first rewritten",
			content: "      - uses: actions/checkout@" + shaA + " # v4.1.1\n" +
				"      - uses: actions/checkout@" + shaA + " # v4.1.1\n",
			action:      "actions/checkout@" + shaA,
			replacement: "actions/checkout@" + shaB + " #v4.2.2",
			want: "      - uses: actions/checkout@" + shaB + " #v4.2.2\n" +
				"      - uses: actions/checkout@" + shaA + " # v4.1.1\n",
			wantMatched: true,
		},
		{
			name:        "comment with extra hash fully replaced",
			content:     "      - uses: actions/checkout@" + shaA + " # v4.1.1 # keep\n",
			action:      "actions/checkout@" + shaA,
			replacement: "actions/checkout@" + shaB + " #v4.2.2",
			want:        "      - uses: actions/checkout@" + shaB + " #v4.2.2\n",
			wantMatched: true,
		},
		{
			name:        "no match leaves content unchanged",
			content:     "      - uses: \"actions/checkout@" + shaA + "\"\n",
			action:      "actions/missing@" + shaX,
			replacement: "actions/missing@" + shaB + " #v1.0.0",
			want:        "      - uses: \"actions/checkout@" + shaA + "\"\n",
			wantMatched: false,
		},
		{
			name:        "idempotent identical replacement",
			content:     "      - uses: actions/checkout@" + shaB + " #v4.2.2\n",
			action:      "actions/checkout@" + shaB,
			replacement: "actions/checkout@" + shaB + " #v4.2.2",
			want:        "      - uses: actions/checkout@" + shaB + " #v4.2.2\n",
			wantMatched: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, matched := ReplaceActionRef(tt.content, tt.action, tt.replacement)
			if matched != tt.wantMatched {
				t.Errorf("ReplaceActionRef matched = %v, want %v", matched, tt.wantMatched)
			}
			if got != tt.want {
				t.Errorf("ReplaceActionRef result = %q, want %q", got, tt.want)
			}
		})
	}
}
