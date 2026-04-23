package pkg

import "testing"

func TestClassifyUsesRef(t *testing.T) {
	tests := []struct {
		name          string
		uses          string
		expectedClass UsesRefClass
		expectedPath  string
		expectedRepo  string
		expectedRef   string
	}{
		{
			name:          "local action",
			uses:          "./.github/actions/setup",
			expectedClass: UsesRefLocal,
			expectedPath:  "./.github/actions/setup",
		},
		{
			name:          "sha ref",
			uses:          "actions/checkout@8f4b7f84864484a7bf31766abe9204da3cbe65b3",
			expectedClass: UsesRefSHA,
			expectedPath:  "actions/checkout",
			expectedRepo:  "actions/checkout",
			expectedRef:   "8f4b7f84864484a7bf31766abe9204da3cbe65b3",
		},
		{
			name:          "sha ref with subpath",
			uses:          "owner/repo/path/to/action@0123456789abcdef0123456789abcdef01234567",
			expectedClass: UsesRefSHA,
			expectedPath:  "owner/repo/path/to/action",
			expectedRepo:  "owner/repo",
			expectedRef:   "0123456789abcdef0123456789abcdef01234567",
		},
		{
			name:          "version ref with v prefix",
			uses:          "actions/setup-go@v5.0.1",
			expectedClass: UsesRefVersion,
			expectedPath:  "actions/setup-go",
			expectedRepo:  "actions/setup-go",
			expectedRef:   "v5.0.1",
		},
		{
			name:          "version ref without v prefix",
			uses:          "actions/setup-go@3.2",
			expectedClass: UsesRefVersion,
			expectedPath:  "actions/setup-go",
			expectedRepo:  "actions/setup-go",
			expectedRef:   "3.2",
		},
		{
			name:          "branch ref",
			uses:          "actions/checkout@main",
			expectedClass: UsesRefBranch,
			expectedPath:  "actions/checkout",
			expectedRepo:  "actions/checkout",
			expectedRef:   "main",
		},
		{
			name:          "branch ref with subpath",
			uses:          "owner/repo/path/to/action@feature/new-branch",
			expectedClass: UsesRefBranch,
			expectedPath:  "owner/repo/path/to/action",
			expectedRepo:  "owner/repo",
			expectedRef:   "feature/new-branch",
		},
		{
			name:          "branch ref with dot slash and subpath",
			uses:          "owner/repo/path/to/action@release/v1.2",
			expectedClass: UsesRefBranch,
			expectedPath:  "owner/repo/path/to/action",
			expectedRepo:  "owner/repo",
			expectedRef:   "release/v1.2",
		},
		{
			name:          "branch ref with slash underscore hyphen",
			uses:          "owner/repo/path/to/action@feature/foo_bar-baz",
			expectedClass: UsesRefBranch,
			expectedPath:  "owner/repo/path/to/action",
			expectedRepo:  "owner/repo",
			expectedRef:   "feature/foo_bar-baz",
		},
		{
			name:          "unsupported docker ref",
			uses:          "docker://alpine:3.20",
			expectedClass: UsesRefUnsupported,
			expectedPath:  "docker://alpine:3.20",
		},
		{
			name:          "unsupported external without at",
			uses:          "actions/checkout",
			expectedClass: UsesRefUnsupported,
			expectedPath:  "actions/checkout",
			expectedRepo:  "actions/checkout",
		},
		{
			name:          "unsupported external trailing at",
			uses:          "actions/checkout@",
			expectedClass: UsesRefUnsupported,
			expectedPath:  "actions/checkout",
			expectedRepo:  "actions/checkout",
		},
		{
			name:          "unsupported malformed external",
			uses:          "actions@v1",
			expectedClass: UsesRefUnsupported,
			expectedPath:  "actions",
			expectedRef:   "v1",
		},
		{
			name:          "version ref with subpath",
			uses:          "owner/repo/sub-action@v3",
			expectedClass: UsesRefVersion,
			expectedPath:  "owner/repo/sub-action",
			expectedRepo:  "owner/repo",
			expectedRef:   "v3",
		},
		{
			name:          "version ref major only with v",
			uses:          "actions/checkout@v4",
			expectedClass: UsesRefVersion,
			expectedPath:  "actions/checkout",
			expectedRepo:  "actions/checkout",
			expectedRef:   "v4",
		},
		{
			name:          "version ref major only without v",
			uses:          "actions/checkout@1",
			expectedClass: UsesRefVersion,
			expectedPath:  "actions/checkout",
			expectedRepo:  "actions/checkout",
			expectedRef:   "1",
		},
		{
			name:          "version ref with subpath and full semver",
			uses:          "owner/repo/sub-action@v1.2.3",
			expectedClass: UsesRefVersion,
			expectedPath:  "owner/repo/sub-action",
			expectedRepo:  "owner/repo",
			expectedRef:   "v1.2.3",
		},
		{
			name:          "branch ref with subpath",
			uses:          "owner/repo/sub-action@main",
			expectedClass: UsesRefBranch,
			expectedPath:  "owner/repo/sub-action",
			expectedRepo:  "owner/repo",
			expectedRef:   "main",
		},
		{
			name:          "sha ref with subpath deep nested",
			uses:          "owner/repo/a/b/c@0123456789abcdef0123456789abcdef01234567",
			expectedClass: UsesRefSHA,
			expectedPath:  "owner/repo/a/b/c",
			expectedRepo:  "owner/repo",
			expectedRef:   "0123456789abcdef0123456789abcdef01234567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyUsesRef(tt.uses)

			if result.Class != tt.expectedClass {
				t.Errorf("ClassifyUsesRef(%q).Class = %q, expected %q", tt.uses, result.Class, tt.expectedClass)
			}
			if result.ActionPath != tt.expectedPath {
				t.Errorf("ClassifyUsesRef(%q).ActionPath = %q, expected %q", tt.uses, result.ActionPath, tt.expectedPath)
			}
			if result.BaseRepo != tt.expectedRepo {
				t.Errorf("ClassifyUsesRef(%q).BaseRepo = %q, expected %q", tt.uses, result.BaseRepo, tt.expectedRepo)
			}
			if result.Ref != tt.expectedRef {
				t.Errorf("ClassifyUsesRef(%q).Ref = %q, expected %q", tt.uses, result.Ref, tt.expectedRef)
			}
		})
	}
}
