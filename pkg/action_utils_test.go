package pkg

import (
	"testing"
)

func TestExtractOwnerRepo(t *testing.T) {
	tests := []struct {
		name         string
		actionPath   string
		expectedRepo string
		expectedOK   bool
	}{
		{
			name:         "owner repo only",
			actionPath:   "owner/repo",
			expectedRepo: "owner/repo",
			expectedOK:   true,
		},
		{
			name:         "owner repo with subpath",
			actionPath:   "owner/repo/path/to/action",
			expectedRepo: "owner/repo",
			expectedOK:   true,
		},
		{
			name:         "owner repo with single subpath",
			actionPath:   "owner/repo/sub",
			expectedRepo: "owner/repo",
			expectedOK:   true,
		},
		{
			name:       "missing repo",
			actionPath: "owner/",
			expectedOK: false,
		},
		{
			name:       "missing owner",
			actionPath: "/repo",
			expectedOK: false,
		},
		{
			name:       "single path segment",
			actionPath: "repo",
			expectedOK: false,
		},
		{
			name:       "empty string",
			actionPath: "",
			expectedOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, ok := ExtractOwnerRepo(test.actionPath)
			if repo != test.expectedRepo || ok != test.expectedOK {
				t.Errorf("ExtractOwnerRepo(%q) = (%q, %t), want (%q, %t)", test.actionPath, repo, ok, test.expectedRepo, test.expectedOK)
			}
		})
	}
}

func TestSplitOnLastAt(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		expectedPath string
		expectedRef  string
		expectedHas  bool
	}{
		{
			name:         "splits once",
			value:        "actions/checkout@v4",
			expectedPath: "actions/checkout",
			expectedRef:  "v4",
			expectedHas:  true,
		},
		{
			name:         "splits on final at",
			value:        "owner/repo@release@candidate",
			expectedPath: "owner/repo@release",
			expectedRef:  "candidate",
			expectedHas:  true,
		},
		{
			name:         "no at",
			value:        "actions/checkout",
			expectedPath: "actions/checkout",
			expectedRef:  "",
			expectedHas:  false,
		},
		{
			name:         "trailing at",
			value:        "owner/repo@",
			expectedPath: "owner/repo",
			expectedRef:  "",
			expectedHas:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, ref, hasRef := splitOnLastAt(test.value)
			if path != test.expectedPath || ref != test.expectedRef || hasRef != test.expectedHas {
				t.Errorf("splitOnLastAt(%q) = (%q, %q, %t), want (%q, %q, %t)", test.value, path, ref, hasRef, test.expectedPath, test.expectedRef, test.expectedHas)
			}
		})
	}
}


