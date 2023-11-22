package pkg

import (
	"fmt"
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		version  string
		expected Semver
		err      error
	}{
		{"v1.2.3", Semver{Major: 1, Minor: 2, Patch: 3}, nil},
		{"v2.0.0", Semver{Major: 2, Minor: 0, Patch: 0}, nil},
		{"v1.0.0-alpha", Semver{}, fmt.Errorf("invalid semver: v1.0.0-alpha")},
		{"v1.2", Semver{}, fmt.Errorf("invalid semver: v1.2")},
		{"v1.2.3.4", Semver{}, fmt.Errorf("invalid semver: v1.2.3.4")},
	}

	for _, test := range tests {
		result, err := ParseSemver(test.version)
		if err != nil && test.err == nil {
			t.Errorf("Unexpected error for version %s: %v", test.version, err)
		} else if err == nil && test.err != nil {
			t.Errorf("Expected error for version %s: %v", test.version, test.err)
		} else if result != test.expected {
			t.Errorf("Unexpected result for version %s: got %v, want %v", test.version, result, test.expected)
		}
	}
}

func TestFindHighestPatchVersion(t *testing.T) {
	tags := []string{"v1.2.3", "v2.0.0", "v1.0.0-alpha", "v1.2", "v1.3.1", "v3.5.0", "v3.4.0"}

	tests := []struct {
		version  string
		expected string
	}{
		{"1.2", "v1.2.3"},
		{"2", "v2.0.0"},
		{"1.3", "v1.3.1"},
		{"1", "v1.3.1"},
		{"3", "v3.5.0"},
	}

	for _, test := range tests {
		result, err := FindHighestPatchVersion(tags, test.version)
		if err != nil {
			t.Errorf("Unexpected error for version %s: %v", test.version, err)
		} else if result != test.expected {
			t.Errorf("Unexpected result for version %s: got %s, want %s", test.version, result, test.expected)
		}
	}
}
