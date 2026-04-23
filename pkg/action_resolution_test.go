package pkg

import (
	"errors"
	"testing"
)

func TestProcessActionWithCacheVersionRefResolvesCorrectly(t *testing.T) {
	var called bool
	resolver := func(repository, version string) (string, string, error) {
		called = true
		if repository != "actions/checkout" {
			t.Errorf("Expected repository actions/checkout, got %s", repository)
		}
		if version != "3" {
			t.Errorf("Expected version 3, got %s", version)
		}
		return "abc123", "v3.5.1", nil
	}
	branchResolver := func(repository, branch string) (string, error) {
		t.Fatal("Branch resolver should not be called for version refs")
		return "", nil
	}

	cache := NewActionResolutionCache()
	got, updated, err := ProcessActionWithCache("actions/checkout@v3", false, cache, resolver, branchResolver, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !called {
		t.Fatal("Expected version resolver to be called")
	}
	if !updated {
		t.Fatal("Expected action to be updated")
	}
	if got != "actions/checkout@abc123 #v3.5.1" {
		t.Errorf("Unexpected pinned action: %s", got)
	}
}

func TestProcessActionWithCacheVersionRefLatestMode(t *testing.T) {
	resolver := func(repository, version string) (string, string, error) {
		if repository != "actions/checkout" {
			t.Errorf("Expected repository actions/checkout, got %s", repository)
		}
		if version != "latest" {
			t.Errorf("Expected version latest, got %s", version)
		}
		return "cafebabe", "v5.0.0", nil
	}
	branchResolver := func(repository, branch string) (string, error) {
		t.Fatal("Branch resolver should not be called for version refs")
		return "", nil
	}

	cache := NewActionResolutionCache()
	got, updated, err := ProcessActionWithCache("actions/checkout@v3", true, cache, resolver, branchResolver, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("Expected action to be updated")
	}
	if got != "actions/checkout@cafebabe #v5.0.0" {
		t.Errorf("Unexpected pinned action: %s", got)
	}
}

func TestProcessActionWithCacheSHARefDefaultVsLatestMode(t *testing.T) {
	var calls int
	versionResolver := func(repository, version string) (string, string, error) {
		calls++
		if repository != "owner/repo" {
			t.Errorf("Expected repository owner/repo, got %s", repository)
		}
		if version != "latest" {
			t.Errorf("Expected version latest, got %s", version)
		}
		return "deadc0de", "v10.2.0", nil
	}
	branchResolver := func(repository, branch string) (string, error) {
		return "", nil
	}

	action := "owner/repo/path/to/action@0123456789abcdef0123456789abcdef01234567"

	// Default mode: SHA ref should be skipped
	cache := NewActionResolutionCache()
	defaultPinned, defaultUpdated, defaultErr := ProcessActionWithCache(action, false, cache, versionResolver, branchResolver, nil)
	if defaultErr != nil {
		t.Fatalf("Unexpected error in default mode: %v", defaultErr)
	}
	if defaultUpdated {
		t.Fatal("Expected sha ref to remain unchanged in default mode")
	}
	if defaultPinned != "" {
		t.Errorf("Expected no replacement action in default mode, got %q", defaultPinned)
	}
	if calls != 0 {
		t.Fatalf("Expected resolver not to be called in default mode, got %d", calls)
	}

	// Latest mode: SHA ref should be re-pinned
	cache = NewActionResolutionCache()
	latestPinned, latestUpdated, latestErr := ProcessActionWithCache(action, true, cache, versionResolver, branchResolver, nil)
	if latestErr != nil {
		t.Fatalf("Unexpected error in latest mode: %v", latestErr)
	}
	if !latestUpdated {
		t.Fatal("Expected sha ref to be re-pinned in latest mode")
	}
	if calls != 1 {
		t.Fatalf("Expected resolver to be called once in latest mode, got %d", calls)
	}
	if latestPinned != "owner/repo/path/to/action@deadc0de #v10.2.0" {
		t.Errorf("Unexpected pinned action in latest mode: %s", latestPinned)
	}
}

func TestProcessActionWithCacheBranchRefBehavesSameInDefaultAndLatestModes(t *testing.T) {
	var branchCalls int
	var versionCalls int
	branchResolver := func(repository, branch string) (string, error) {
		branchCalls++
		if repository != "owner/repo" {
			t.Errorf("Expected repository owner/repo, got %s", repository)
		}
		if branch != "release/v1.2" {
			t.Errorf("Expected branch release/v1.2, got %s", branch)
		}
		return "1234abcd", nil
	}
	versionResolver := func(repository, version string) (string, string, error) {
		versionCalls++
		return "", "", nil
	}

	action := "owner/repo/path/to/action@release/v1.2"

	cache := NewActionResolutionCache()
	defaultPinned, defaultUpdated, defaultErr := ProcessActionWithCache(action, false, cache, versionResolver, branchResolver, nil)
	if defaultErr != nil {
		t.Fatalf("Unexpected error in default mode: %v", defaultErr)
	}
	if !defaultUpdated {
		t.Fatal("Expected branch ref to be updated in default mode")
	}

	cache = NewActionResolutionCache()
	latestPinned, latestUpdated, latestErr := ProcessActionWithCache(action, true, cache, versionResolver, branchResolver, nil)
	if latestErr != nil {
		t.Fatalf("Unexpected error in latest mode: %v", latestErr)
	}
	if !latestUpdated {
		t.Fatal("Expected branch ref to be updated in latest mode")
	}

	expectedPinned := "owner/repo/path/to/action@1234abcd #release/v1.2"
	if defaultPinned != expectedPinned {
		t.Errorf("Unexpected pinned action in default mode: %s", defaultPinned)
	}
	if latestPinned != expectedPinned {
		t.Errorf("Unexpected pinned action in latest mode: %s", latestPinned)
	}
	if branchCalls != 2 {
		t.Fatalf("Expected branch resolver to be called twice, got %d", branchCalls)
	}
	if versionCalls != 0 {
		t.Fatalf("Expected version resolver not to be called for branch refs, got %d", versionCalls)
	}
}

func TestProcessActionWithCacheReusesLatestLookupBetweenVersionAndSHA(t *testing.T) {
	var calls int
	versionResolver := func(repository, version string) (string, string, error) {
		calls++
		if repository != "owner/repo" {
			t.Errorf("Expected repository owner/repo, got %s", repository)
		}
		if version != "latest" {
			t.Errorf("Expected version latest, got %s", version)
		}
		return "beadbead", "v7.1.0", nil
	}
	branchResolver := func(repository, branch string) (string, error) {
		return "", nil
	}

	cache := NewActionResolutionCache()
	versionPinned, versionUpdated, versionErr := ProcessActionWithCache("owner/repo/path/to/action@v1", true, cache, versionResolver, branchResolver, nil)
	if versionErr != nil {
		t.Fatalf("Unexpected error for version action: %v", versionErr)
	}
	if !versionUpdated {
		t.Fatal("Expected version action to be updated")
	}

	shaPinned, shaUpdated, shaErr := ProcessActionWithCache("owner/repo/another/path/action@0123456789abcdef0123456789abcdef01234567", true, cache, versionResolver, branchResolver, nil)
	if shaErr != nil {
		t.Fatalf("Unexpected error for sha action: %v", shaErr)
	}
	if !shaUpdated {
		t.Fatal("Expected sha action to be updated")
	}

	if calls != 1 {
		t.Fatalf("Expected resolver call to be reused from cache, got %d calls", calls)
	}
	if versionPinned != "owner/repo/path/to/action@beadbead #v7.1.0" {
		t.Errorf("Unexpected pinned version action: %s", versionPinned)
	}
	if shaPinned != "owner/repo/another/path/action@beadbead #v7.1.0" {
		t.Errorf("Unexpected pinned sha action: %s", shaPinned)
	}
}

func TestProcessActionWithCacheCachesResolutionFailures(t *testing.T) {
	var calls int
	var warnings int
	versionResolver := func(repository, version string) (string, string, error) {
		calls++
		return "", "", errors.New("boom")
	}
	branchResolver := func(repository, branch string) (string, error) {
		return "", nil
	}
	warn := func(action string, err error) {
		warnings++
	}

	cache := NewActionResolutionCache()
	firstPinned, firstUpdated, firstErr := ProcessActionWithCache("actions/checkout@v4", true, cache, versionResolver, branchResolver, warn)
	secondPinned, secondUpdated, secondErr := ProcessActionWithCache("actions/checkout@v2", true, cache, versionResolver, branchResolver, warn)

	if firstErr != nil || secondErr != nil {
		t.Fatalf("Expected failures to be handled without bubbling errors, got first=%v second=%v", firstErr, secondErr)
	}
	if firstUpdated || secondUpdated {
		t.Fatal("Expected unresolved actions to remain unchanged")
	}
	if firstPinned != "" || secondPinned != "" {
		t.Fatalf("Expected no pinned output on failure, got first=%q second=%q", firstPinned, secondPinned)
	}
	if calls != 1 {
		t.Fatalf("Expected failure result to be cached, got %d calls", calls)
	}
	if warnings != 2 {
		t.Fatalf("Expected warnings for each unresolved action, got %d", warnings)
	}
}

func TestProcessActionWithCacheSkipsLocalAndUnsupportedRefsWithoutResolverCalls(t *testing.T) {
	var versionCalls int
	var branchCalls int
	versionResolver := func(repository, version string) (string, string, error) {
		versionCalls++
		return "", "", nil
	}
	branchResolver := func(repository, branch string) (string, error) {
		branchCalls++
		return "", nil
	}

	tests := []struct {
		name   string
		action string
	}{
		{
			name:   "local action",
			action: "./.github/actions/setup",
		},
		{
			name:   "unsupported docker action",
			action: "docker://alpine:3.20",
		},
		{
			name:   "unsupported external action",
			action: "actions/checkout",
		},
	}

	for _, tt := range tests {
		tt := tt
		for _, latestMode := range []bool{false, true} {
			latestMode := latestMode
			modeName := "default"
			if latestMode {
				modeName = "latest"
			}
			t.Run(tt.name+" in "+modeName+" mode", func(t *testing.T) {
				cache := NewActionResolutionCache()
				got, shouldUpdate, err := ProcessActionWithCache(tt.action, latestMode, cache, versionResolver, branchResolver, nil)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if shouldUpdate {
					t.Fatal("Expected action to remain unchanged")
				}
				if got != "" {
					t.Errorf("Expected no replacement action, got %q", got)
				}
			})
		}
	}

	if versionCalls != 0 {
		t.Fatalf("Expected version resolver not to be called, got %d", versionCalls)
	}
	if branchCalls != 0 {
		t.Fatalf("Expected branch resolver not to be called, got %d", branchCalls)
	}
}

func TestProcessActionWithCacheFormatMatrix(t *testing.T) {
	// Tests all documented action formats through ProcessActionWithCache in both modes,
	// verifying correct repo is sent to resolver and subpath is preserved in output.
	versionResolver := func(repository, version string) (string, string, error) {
		return "abc123sha", "v5.0.0", nil
	}
	branchResolver := func(repository, branch string) (string, error) {
		return "def456sha", nil
	}

	tests := []struct {
		name           string
		action         string
		latestMode     bool
		expectedOutput string
		expectedUpdate bool
		expectedRepo   string // repo sent to resolver
	}{
		// owner/repo@version — default mode
		{
			name:           "owner/repo@version default",
			action:         "actions/checkout@v3",
			latestMode:     false,
			expectedOutput: "actions/checkout@abc123sha #v5.0.0",
			expectedUpdate: true,
			expectedRepo:   "actions/checkout",
		},
		// owner/repo@version — latest mode
		{
			name:           "owner/repo@version latest",
			action:         "actions/checkout@v3",
			latestMode:     true,
			expectedOutput: "actions/checkout@abc123sha #v5.0.0",
			expectedUpdate: true,
			expectedRepo:   "actions/checkout",
		},
		// owner/repo/sub-action@version — default mode (subpath preserved)
		{
			name:           "owner/repo/sub-action@version default",
			action:         "owner/repo/sub-action@v2",
			latestMode:     false,
			expectedOutput: "owner/repo/sub-action@abc123sha #v5.0.0",
			expectedUpdate: true,
			expectedRepo:   "owner/repo",
		},
		// owner/repo/sub-action@version — latest mode (subpath preserved)
		{
			name:           "owner/repo/sub-action@version latest",
			action:         "owner/repo/sub-action@v2",
			latestMode:     true,
			expectedOutput: "owner/repo/sub-action@abc123sha #v5.0.0",
			expectedUpdate: true,
			expectedRepo:   "owner/repo",
		},
		// owner/repo@branch — default mode
		{
			name:           "owner/repo@branch default",
			action:         "actions/checkout@main",
			latestMode:     false,
			expectedOutput: "actions/checkout@def456sha #main",
			expectedUpdate: true,
			expectedRepo:   "actions/checkout",
		},
		// owner/repo@branch — latest mode (same behavior)
		{
			name:           "owner/repo@branch latest",
			action:         "actions/checkout@main",
			latestMode:     true,
			expectedOutput: "actions/checkout@def456sha #main",
			expectedUpdate: true,
			expectedRepo:   "actions/checkout",
		},
		// owner/repo/sub-action@branch — default mode
		{
			name:           "owner/repo/sub-action@branch default",
			action:         "owner/repo/sub-action@develop",
			latestMode:     false,
			expectedOutput: "owner/repo/sub-action@def456sha #develop",
			expectedUpdate: true,
			expectedRepo:   "owner/repo",
		},
		// owner/repo@sha — default mode (skip)
		{
			name:           "owner/repo@sha default skipped",
			action:         "actions/checkout@0123456789abcdef0123456789abcdef01234567",
			latestMode:     false,
			expectedOutput: "",
			expectedUpdate: false,
			expectedRepo:   "",
		},
		// owner/repo@sha — latest mode (repin)
		{
			name:           "owner/repo@sha latest repin",
			action:         "actions/checkout@0123456789abcdef0123456789abcdef01234567",
			latestMode:     true,
			expectedOutput: "actions/checkout@abc123sha #v5.0.0",
			expectedUpdate: true,
			expectedRepo:   "actions/checkout",
		},
		// owner/repo/sub-action@sha — latest mode (subpath preserved)
		{
			name:           "owner/repo/sub-action@sha latest repin",
			action:         "owner/repo/sub-action@0123456789abcdef0123456789abcdef01234567",
			latestMode:     true,
			expectedOutput: "owner/repo/sub-action@abc123sha #v5.0.0",
			expectedUpdate: true,
			expectedRepo:   "owner/repo",
		},
		// owner/repo/sub-action@branchName with slash
		{
			name:           "owner/repo/sub-action@branch-with-slash",
			action:         "owner/repo/sub-action@release/v1.2",
			latestMode:     false,
			expectedOutput: "owner/repo/sub-action@def456sha #release/v1.2",
			expectedUpdate: true,
			expectedRepo:   "owner/repo",
		},
		// version without v prefix
		{
			name:           "version without v prefix",
			action:         "actions/setup-go@3.2",
			latestMode:     false,
			expectedOutput: "actions/setup-go@abc123sha #v5.0.0",
			expectedUpdate: true,
			expectedRepo:   "actions/setup-go",
		},
		// major-only version
		{
			name:           "major-only version",
			action:         "actions/checkout@4",
			latestMode:     false,
			expectedOutput: "actions/checkout@abc123sha #v5.0.0",
			expectedUpdate: true,
			expectedRepo:   "actions/checkout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRepo string
			testVersionResolver := func(repository, version string) (string, string, error) {
				capturedRepo = repository
				return versionResolver(repository, version)
			}
			testBranchResolver := func(repository, branch string) (string, error) {
				capturedRepo = repository
				return branchResolver(repository, branch)
			}

			cache := NewActionResolutionCache()
			got, updated, err := ProcessActionWithCache(tt.action, tt.latestMode, cache, testVersionResolver, testBranchResolver, nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if updated != tt.expectedUpdate {
				t.Errorf("Expected updated=%t, got %t", tt.expectedUpdate, updated)
			}
			if got != tt.expectedOutput {
				t.Errorf("Expected output %q, got %q", tt.expectedOutput, got)
			}
			if tt.expectedRepo != "" && capturedRepo != tt.expectedRepo {
				t.Errorf("Expected repo sent to resolver %q, got %q", tt.expectedRepo, capturedRepo)
			}
		})
	}
}
