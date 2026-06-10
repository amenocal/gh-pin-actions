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
	tempFile, err := os.CreateTemp("", "tempfile")
	if err != nil {
		t.Fatalf("Unexpected error creating temp file: %v", err)
	}
	defer tempFile.Close()

	content := []byte("test content")
	if _, err := tempFile.Write(content); err != nil {
		t.Fatalf("Unexpected error writing to temp file: %v", err)
	}

	tempFileName := tempFile.Name()
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
