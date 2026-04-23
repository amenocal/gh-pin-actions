package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/pterm/pterm"
)

func ensureTestLogger() {
	if logger == nil {
		logger = pterm.DefaultLogger.WithLevel(pterm.LogLevelWarn)
	}
}

func TestGetWorkflowFiles(t *testing.T) {
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

	originalDir, _ := os.Getwd()
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
	tempDir := t.TempDir()
	originalFile := filepath.Join(tempDir, "workflow.yml")
	content := []byte("test content")
	if err := os.WriteFile(originalFile, content, 0644); err != nil {
		t.Fatalf("Unexpected error writing source file: %v", err)
	}

	newFileName, err := createTempYAMLFile(originalFile)
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

func TestWorkflowsLatestFlagWiring(t *testing.T) {
	latestFlag := workflowsCmd.Flags().Lookup("latest")
	if latestFlag == nil {
		t.Fatal("expected workflows latest flag to be registered")
	}

	if latestFlag.DefValue != "false" {
		t.Errorf("Expected default latest flag value false, got %s", latestFlag.DefValue)
	}

	if latestFlag.Usage == "" {
		t.Error("Expected latest flag to include help text")
	}
}

func TestDefaultWorkflowResolverBackendsUseCmdRootResolvers(t *testing.T) {
	backends := defaultWorkflowResolverBackends()
	if backends.resolveByVersion == nil {
		t.Fatal("expected version resolver backend")
	}
	if backends.resolveByBranch == nil {
		t.Fatal("expected branch resolver backend")
	}

	versionResolverName := runtime.FuncForPC(reflect.ValueOf(backends.resolveByVersion).Pointer()).Name()
	if !strings.HasSuffix(versionResolverName, ".GetActionHashByVersion") {
		t.Fatalf("expected GetActionHashByVersion backend, got %s", versionResolverName)
	}

	branchResolverName := runtime.FuncForPC(reflect.ValueOf(backends.resolveByBranch).Pointer()).Name()
	if !strings.HasSuffix(branchResolverName, ".GetBranchHash") {
		t.Fatalf("expected GetBranchHash backend, got %s", branchResolverName)
	}
}

func TestProcessActionWithCacheWithResolversReusesLatestLookupBetweenVersionAndSHA(t *testing.T) {
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

	cache := newActionResolutionCache()
	versionPinned, versionUpdated, versionErr := processActionWithCacheWithResolvers("owner/repo/path/to/action@v1", true, cache, versionResolver, branchResolver)
	if versionErr != nil {
		t.Fatalf("Unexpected error for version action: %v", versionErr)
	}
	if !versionUpdated {
		t.Fatal("Expected version action to be updated")
	}

	shaPinned, shaUpdated, shaErr := processActionWithCacheWithResolvers("owner/repo/another/path/action@0123456789abcdef0123456789abcdef01234567", true, cache, versionResolver, branchResolver)
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

func TestProcessActionWithCacheWithResolversCachesResolutionFailures(t *testing.T) {
	var calls int
	versionResolver := func(repository, version string) (string, string, error) {
		calls++
		return "", "", errors.New("boom")
	}
	branchResolver := func(repository, branch string) (string, error) {
		return "", nil
	}

	cache := newActionResolutionCache()
	firstPinned, firstUpdated, firstErr := processActionWithCacheWithResolvers("actions/checkout@v4", true, cache, versionResolver, branchResolver)
	secondPinned, secondUpdated, secondErr := processActionWithCacheWithResolvers("actions/checkout@v2", true, cache, versionResolver, branchResolver)

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
}



func TestWriteModifiedWorkflowToFile(t *testing.T) {
ensureTestLogger()

tests := []struct {
name          string
initialLine   string
action        string
actionWithSha string
expectedLine  string
}{
{
name:          "first pin: version ref without existing comment",
initialLine:   "      uses: actions/checkout@v4\n",
action:        "actions/checkout@v4",
actionWithSha: "actions/checkout@abc123 #v4.1.0",
expectedLine:  "      uses: actions/checkout@abc123 #v4.1.0\n",
},
{
name:          "re-pin with --latest: removes existing #comment",
initialLine:   "      uses: actions/setup-go@oldsha #v4.1.0\n",
action:        "actions/setup-go@oldsha",
actionWithSha: "actions/setup-go@newsha #v6.4.0",
expectedLine:  "      uses: actions/setup-go@newsha #v6.4.0\n",
},
{
name:          "re-pin with --latest: removes existing #comment with extra spaces",
initialLine:   "      uses: actions/checkout@oldsha    #v4.1.0\n",
action:        "actions/checkout@oldsha",
actionWithSha: "actions/checkout@newsha #v6.0.2",
expectedLine:  "      uses: actions/checkout@newsha #v6.0.2\n",
},
{
name:          "re-pin with --latest: subpath action with existing comment",
initialLine:   "      uses: owner/repo/sub-action@oldsha #v1.2.3\n",
action:        "owner/repo/sub-action@oldsha",
actionWithSha: "owner/repo/sub-action@newsha #v2.0.0",
expectedLine:  "      uses: owner/repo/sub-action@newsha #v2.0.0\n",
},
{
name:          "no trailing comment on line: replaces cleanly",
initialLine:   "      uses: actions/checkout@v3  \n",
action:        "actions/checkout@v3",
actionWithSha: "actions/checkout@abc123 #v3.5.3",
expectedLine:  "      uses: actions/checkout@abc123 #v3.5.3  \n",
},
{
name:          "does not consume comment on a following line",
initialLine:   "      uses: actions/checkout@v3\n      # keep this comment\n",
action:        "actions/checkout@v3",
actionWithSha: "actions/checkout@abc123 #v3.5.3",
expectedLine:  "      uses: actions/checkout@abc123 #v3.5.3\n      # keep this comment\n",
},
{
name:          "action not present in file: file unchanged",
initialLine:   "      uses: actions/checkout@v4\n",
action:        "actions/setup-go@v5",
actionWithSha: "actions/setup-go@abc #v5.0.0",
expectedLine:  "      uses: actions/checkout@v4\n",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
tmpFile, err := os.CreateTemp("", "workflow-*.yml")
if err != nil {
t.Fatalf("failed to create temp file: %v", err)
}
defer os.Remove(tmpFile.Name())

if _, err := tmpFile.WriteString(tt.initialLine); err != nil {
t.Fatalf("failed to write temp file: %v", err)
}
tmpFile.Close()

writeModifiedWorkflowToFile(tmpFile.Name(), tt.action, tt.actionWithSha)

got, err := os.ReadFile(tmpFile.Name())
if err != nil {
t.Fatalf("failed to read temp file: %v", err)
}
if string(got) != tt.expectedLine {
t.Errorf("writeModifiedWorkflowToFile mismatch\n got:  %q\n want: %q", string(got), tt.expectedLine)
}
})
}
}
