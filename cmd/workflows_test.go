package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetWorkflowFiles(t *testing.T) {
	// Setup: create a temporary directory with test files
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755)
	testFiles := []string{
		"workflow1.yml",
		"workflow2.yaml",
		"workflow-pin.yml",
		"workflow-pin.yaml",
	}
	for _, file := range testFiles {
		os.WriteFile(filepath.Join(tempDir, ".github", "workflows", file), []byte{}, 0644)
	}

	// Override the directory for testing
	originalDir, _ := os.Getwd()
	defer func() { os.Chdir(originalDir) }()
	os.Chdir(tempDir)

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
