package cmd

import "testing"

func TestBuildPinnedRef(t *testing.T) {
	if got := BuildPinnedRef("a/b", "abc ", "v1.2.3\n", true); got != "a/b@v1.2.3" {
		t.Errorf("immutable: got %q", got)
	}
	if got := BuildPinnedRef("a/b", "abc ", "v1.2.3\n", false); got != "a/b@abc #v1.2.3" {
		t.Errorf("mutable: got %q", got)
	}
}
