package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pterm/pterm"
)

func TestMain(m *testing.M) {
	// writePinnedActionUpdate and friends log via the package-level logger.
	logger = pterm.DefaultLogger.WithLevel(pterm.LogLevelWarn)
	// Guard against latestCache state leaking across the test binary's runs.
	latestCache = map[string]latestResult{}
	os.Exit(m.Run())
}

func TestProcessActionSkipsLocalAndDocker(t *testing.T) {
	actions := []string{
		"./",
		"./local-action",
		"./.github/actions/foo",
		"./local@v1",
		"./local@main",
		"../shared/action",
		"../shared/action@v1",
		"$/local",
		"$/local/action@v1",
		"$/local@main",
		"docker://alpine:3.22",
		"docker://alpine@sha256:0123456789abcdef",
		"docker://ghcr.io/foo/bar:latest",
		"docker://ghcr.io/foo/bar@v1",
	}
	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			got, err := processAction(action)
			if err != nil || got != action {
				t.Errorf("processAction(%q) = %q, %v; want unchanged", action, got, err)
			}
		})
	}
}

func TestSelectVersion(t *testing.T) {
	tests := []struct {
		name      string
		declared  string
		pinLatest bool
		want      string
	}{
		{name: "declared kept when pinLatest false", declared: "v4.0.", pinLatest: false, want: "v4.0."},
		{name: "latest when pinLatest true", declared: "v4.0.", pinLatest: true, want: "latest"},
		{name: "empty declared with pinLatest true", declared: "", pinLatest: true, want: "latest"},
		{name: "numeric declared kept when pinLatest false", declared: "3.1.1", pinLatest: false, want: "3.1.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectVersion(tt.declared, tt.pinLatest); got != tt.want {
				t.Errorf("selectVersion(%q, %v) = %q, want %q", tt.declared, tt.pinLatest, got, tt.want)
			}
		})
	}
}

func TestGetWorkflowFiles(t *testing.T) {
	// Setup: create a temporary directory with test files
	tempDir := t.TempDir()
	err := os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)
	if err != nil {
		t.Fatalf("Unexpected error creating directory: %v", err)
	}
	testFiles := []string{
		"workflow1.yml",
		"workflow2.yaml",
		"workflow-pin.yml",
		"workflow-pin.yaml",
	}
	for _, file := range testFiles {
		err := os.WriteFile(filepath.Join(tempDir, ".github", "workflows", file), []byte{}, 0644)
		if err != nil {
			t.Fatalf("Unexpected error writing file: %v", err)
		}
	}

	// Override the directory for testing
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Unexpected error getting working directory: %v", err)
	}
	defer func() {
		err := os.Chdir(originalDir)
		if err != nil {
			t.Fatalf("Unexpected error changing directory: %v", err)
		}
	}()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Unexpected error changing directory: %v", err)
	}

	workflowFiles, err := getWorkflowFiles()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedFiles := []string{
		filepath.Join(".github", "workflows", "workflow1.yml"),
		filepath.Join(".github", "workflows", "workflow2.yaml"),
	}

	if len(workflowFiles) != len(expectedFiles) {
		t.Errorf("Expected %d workflow files, got %d", len(expectedFiles), len(workflowFiles))
	}

	for _, expected := range expectedFiles {
		found := false
		for _, actual := range workflowFiles {
			if strings.HasSuffix(actual, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %s not found", expected)
		}
	}
}

func TestCreateTempYAMLFile(t *testing.T) {
	// Setup: create a temporary file
	tempFileName := filepath.Join(t.TempDir(), "tempfile")
	content := []byte("test content")
	if err := os.WriteFile(tempFileName, content, 0600); err != nil {
		t.Fatalf("Unexpected error writing temp file: %v", err)
	}

	newFileName, err := createTempYAMLFile(tempFileName)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.HasSuffix(newFileName, "-pin.yml") {
		t.Errorf("Expected new file name to end with '-pin.yml', got %s", newFileName)
	}

	newContent, err := os.ReadFile(newFileName)
	if err != nil {
		t.Fatalf("Unexpected error reading new file: %v", err)
	}

	if string(newContent) != string(content) {
		t.Errorf("Expected content %s, got %s", string(content), string(newContent))
	}
}

func TestWritePinnedActionUpdate(t *testing.T) {
	const (
		shaA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		shaB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		shaX = "cccccccccccccccccccccccccccccccccccccccc"
	)
	tests := []struct {
		name          string
		initial       string
		action        string
		actionWithSha string
		want          string
	}{
		{
			name:          "rewrites matching ref in file",
			initial:       "      - uses: actions/checkout@" + shaA + " # v4.1.1\n",
			action:        "actions/checkout@" + shaA,
			actionWithSha: "actions/checkout@" + shaB + " #v4.2.2",
			want:          "      - uses: actions/checkout@" + shaB + " #v4.2.2\n",
		},
		{
			name:          "no match leaves file unchanged",
			initial:       "      - uses: \"actions/checkout@" + shaA + "\"\n",
			action:        "actions/missing@" + shaX,
			actionWithSha: "actions/missing@" + shaB + " #v1.0.0",
			want:          "      - uses: \"actions/checkout@" + shaA + "\"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "workflow.yml")
			if err := os.WriteFile(file, []byte(tt.initial), 0644); err != nil {
				t.Fatalf("Unexpected error writing initial file: %v", err)
			}
			writePinnedActionUpdate(file, tt.action, tt.actionWithSha)
			got, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("Unexpected error reading file: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("writePinnedActionUpdate result = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestReplaceFullUsesRepeatedReference(t *testing.T) {
	content := "steps:\n  - uses: o/r@v1\n  - uses: o/r@v1\n  - uses: o/r@v10\n"
	got := replaceFullUses(content, "o/r@v1", "o/r@v1.2.3")
	got = replaceFullUses(got, "o/r@v1", "o/r@v1.2.3")
	want := "steps:\n  - uses: o/r@v1.2.3\n  - uses: o/r@v1.2.3\n  - uses: o/r@v10\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPinActionInWorkflowKeepsUnsupportedRef(t *testing.T) {
	file := filepath.Join(t.TempDir(), "w.yml")
	content := "steps:\n  - uses: o/r@1.2.3\n"
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	pinActionInWorkflow(file, "o/r@1.2.3")
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("file changed: %q", got)
	}
}

func TestPinActionInWorkflowSkipsImmutableTags(t *testing.T) {
	for _, ref := range []string{"1", "1.2.3", "v1.2.3"} {
		immutableCache["o/r@"+ref] = true
		file := filepath.Join(t.TempDir(), "w.yml")
		content := "steps:\n  - uses: o/r@" + ref + "\n"
		if err := os.WriteFile(file, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		pinActionInWorkflow(file, "o/r@"+ref)
		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != content {
			t.Errorf("%s: file changed: %q", ref, got)
		}
	}
}

func TestPinActionInWorkflowLatestUpgradesImmutableTag(t *testing.T) {
	oldLatest := pinLatest
	pinLatest = true
	defer func() { pinLatest = oldLatest }()
	sha := strings.Repeat("a", 40)
	immutableCache["lt/repo@1.0.0"] = true
	immutableCache["lt/repo@3.0.0"] = true
	immutableCache["lt/mut@1.0.0"] = true
	immutableCache["lt/mut@v3.0.0"] = false
	latestCache["lt/repo"] = latestResult{sha: sha, tag: "3.0.0"}
	latestCache["lt/mut"] = latestResult{sha: sha, tag: "v3.0.0"}
	tests := []struct{ action, want string }{
		{"lt/repo@1.0.0", "lt/repo@3.0.0"},
		{"lt/mut@1.0.0", "lt/mut@" + sha + " #v3.0.0"},
	}
	for _, tt := range tests {
		file := filepath.Join(t.TempDir(), "w.yml")
		if err := os.WriteFile(file, []byte("  - uses: "+tt.action+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		pinActionInWorkflow(file, tt.action)
		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "  - uses: "+tt.want+"\n" {
			t.Errorf("%s: got %q, want %q", tt.action, got, tt.want)
		}
	}
}
