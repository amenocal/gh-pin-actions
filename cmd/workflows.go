package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pterm/pterm"

	"github.com/amenocal/gh-pin-actions/pkg"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	workflowsCmd = &cobra.Command{
		Use:   "workflows",
		Short: "Updates all .github/workflows to pin actions to a specific sha",
		Long: `Update all workflow files in .github/workflows and reads every 
		action with version in the workflow file and replaces it with the sha of the specific version`,
		Run: processWorkflows,
	}
	logger             *pterm.Logger
	overwriteWorkflows bool
	pinLatest          bool

	hashRegexp   = regexp.MustCompile(`@[0-9a-f]{40}`)
	branchRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+@[a-zA-Z0-9_-]+$`)
)

type Step struct {
	Name string `yaml:"name"`
	Uses string `yaml:"uses"`
}

type Job struct {
	Steps []Step `yaml:"steps"`
}

type Workflow struct {
	Jobs map[string]Job `yaml:"jobs"`
}

func init() {
	rootCmd.AddCommand(workflowsCmd)

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	workflowsCmd.Flags().BoolVarP(&overwriteWorkflows, "overwrite", "o", false, "overwrite existing workflow files")
	workflowsCmd.Flags().BoolVarP(&pinLatest, "latest", "l", false, "pin actions to the latest release across all major versions instead of the declared version")
	// rootCmd.MarkFlagRequired("repository")

	// rootCmd.Flags().StringVarP(&version, "version", "v", "latest", "version of the tag to pin to (ex. 3; 3.1; 3.1.1)")
	// rootCmd.MarkFlagRequired("repository")

}

func processWorkflows(_ *cobra.Command, _ []string) {
	debug = rootCmd.Flag("debug").Value.String() == "true"
	if debug {
		logger = pterm.DefaultLogger.WithLevel(pterm.LogLevelDebug)
	} else {
		logger = pterm.DefaultLogger.WithLevel(pterm.LogLevelWarn)
	}

	workflowFiles, err := getWorkflowFiles()
	if err != nil {
		logger.Error("Error reading .github/workflow files", logger.Args("error:", err))
		return
	}

	for _, file := range workflowFiles {
		processActionsYaml(file)
	}
}

func getWorkflowFiles() ([]string, error) {
	var workflowFiles []string

	files, err := os.ReadDir(".github/workflows")
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if (strings.HasSuffix(file.Name(), ".yml") || strings.HasSuffix(file.Name(), ".yaml")) &&
			!strings.Contains(file.Name(), "-pin.yml") && !strings.Contains(file.Name(), "-pin.yaml") {
			workflowFiles = append(workflowFiles, filepath.Join(".github", "workflows", file.Name()))
		}
	}

	return workflowFiles, nil
}

func processActionsYaml(workflow string) {
	var wf Workflow
	data, err := os.ReadFile(workflow)
	logger.Print("Processing workflow", logger.Args("file:", workflow))
	if err != nil {
		logger.Warn("Error reading YAML", logger.Args("file:", workflow, "error:", err))

	}
	err = yaml.Unmarshal(data, &wf)
	if err != nil {
		logger.Warn("Error unmarshalling YAML", logger.Args("file:", workflow, "error:", err))
	}
	pinnedWorkflow, err := createTempYAMLFile(workflow)
	if err != nil {
		logger.Warn("Error creating temp file", logger.Args("file:", workflow, "error:", err))
	}

	// Loop through all the jobs and steps
	for _, job := range wf.Jobs {
		for i, step := range job.Steps {
			logger.Trace("Processing Step", logger.Args("step:", fmt.Sprintf("%d %s", i+1, step.Name)))
			if action := step.Uses; action != "" {
				pinActionInWorkflow(pinnedWorkflow, action)
			}
		}
	}
	if overwriteWorkflows {
		err := os.Rename(pinnedWorkflow, workflow)
		if err != nil {
			logger.Warn("Error renaming file", logger.Args("file:", pinnedWorkflow, "error:", err))
		}
		pinnedWorkflow = workflow
	}
	if err == nil {
		fmt.Println("Done! Please review the changes in the following file:", pinnedWorkflow)
	}

}

// pinActionInWorkflow pins a single action reference in pinnedWorkflow. Already-hashed actions are
// left untouched unless --latest is set, in which case they are re-pinned to the newest release;
// version- or branch-tagged actions are resolved to their commit SHA.
func pinActionInWorkflow(pinnedWorkflow string, action string) {
	if hashRegexp.MatchString(action) {
		if !pinLatest {
			logger.Info("Action already has a hash", logger.Args("action:", action))
			return
		}
		updated, changed, err := processPinnedActionToLatest(action)
		if err != nil {
			logger.Warn("Could not re-pin already-pinned action to latest; leaving as-is",
				logger.Args("action:", action, "error:", err))
			return
		}
		if !changed {
			logger.Info("Action already pinned to latest", logger.Args("action:", action))
			return
		}
		writePinnedActionUpdate(pinnedWorkflow, action, updated)
		return
	}
	// Action doesn't have a hash
	actionWithSha, err := processAction(action)
	if err != nil {
		logger.Warn("Nothing will be updated")
	}
	writeModifiedWorkflowToFile(pinnedWorkflow, action, actionWithSha)
}

func writeModifiedWorkflowToFile(fileName string, action string, actionWithSha string) {
	if action == "" || actionWithSha == "" {
		logger.Error("action or actionWithSha are empty", logger.Args("action:", action, "actionWithSha:", actionWithSha))
	}
	newFileContent, err := os.ReadFile(fileName)
	if err != nil {
		logger.Warn("Error reading file", logger.Args("file:", fileName, "error:", err))
	}
	modifiedContent := strings.Replace(string(newFileContent), action, actionWithSha, 1)
	err = os.WriteFile(fileName, []byte(modifiedContent), 0600)
	if err != nil {
		logger.Warn("Error creating file", logger.Args("file:", fileName, "error:", err))
	}
	logger.Info("Replacing action with sha", logger.Args("action:", action, "sha:", actionWithSha))
}

func createTempYAMLFile(fileName string) (string, error) {
	content, err := os.ReadFile(fileName)
	if err != nil {
		logger.Warn("Error reading file", logger.Args("file:", fileName, "error:", err))
		return "", err
	}
	tempFileName := strings.TrimSuffix(fileName, ".yml")
	newFileName := tempFileName + "-pin.yml"

	err = os.WriteFile(newFileName, content, 0600)
	if err != nil {
		logger.Warn("Error writing to file", logger.Args("file:", newFileName, "error:", err))
		return "", err
	}
	return newFileName, nil
}

// selectVersion returns "latest" when pinLatest is set, otherwise the declared version.
// "latest" overrides the default latest-patch-within-declared-major resolution in GetActionHashByVersion.
func selectVersion(declared string, pinLatest bool) string {
	if pinLatest {
		return latestVersion
	}
	return declared
}

func processActionWithVersion(actionWithVersion string) (string, error) {
	repoWithOwner, versionParsed, err := pkg.SplitActionString(actionWithVersion, "@v")
	if err != nil {
		return "", err
	}
	actionVersion := pkg.FormatVersion(versionParsed)
	commitSha, tagVersion, err := GetActionHashByVersion(repoWithOwner, selectVersion(actionVersion, pinLatest))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s@%s #%s", repoWithOwner, strings.TrimSpace(commitSha), strings.TrimSpace(tagVersion)), nil

}

func processActionWithBranch(actionWithBranch string) (string, error) {
	repoWithOwner, branchName, err := pkg.SplitActionString(actionWithBranch, "@")
	if err != nil {
		return "", err
	}
	commitSha, err := GetBranchHash(repoWithOwner, branchName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s@%s #%s", repoWithOwner, strings.TrimSpace(commitSha), strings.TrimSpace(branchName)), nil
}

func processAction(action string) (string, error) {
	var actionWithSha string
	var err error
	switch {
	case strings.Contains(action, "@v"):
		// Action has a version
		actionWithVersion := action
		actionWithSha, err = processActionWithVersion(actionWithVersion)
		if err != nil {
			logger.Error("Error getting commit sha for action", logger.Args("action:", actionWithVersion, "error:", err))
			actionWithSha = actionWithVersion
			return actionWithSha, err
		}
	case branchRegexp.MatchString(action):
		// Action has a branch
		actionWithBranch := action
		actionWithSha, err = processActionWithBranch(actionWithBranch)
		if err != nil {
			logger.Error("Error getting commit sha for action with Branch", logger.Args("action:", actionWithBranch, "error:", err))
			actionWithSha = actionWithBranch
			return actionWithSha, err
		}
	case strings.Contains(action, "./"):
		// Action is local
		logger.Info("Action is local", logger.Args("action:", action))
		return action, nil
	}
	return actionWithSha, nil
}

type latestResult struct {
	sha string
	tag string
	err error
}

// latestCache memoizes "latest" lookups per run, keyed by owner/repo so sub-paths share a lookup.
var latestCache = map[string]latestResult{}

func resolveLatest(repoWithOwner string) (string, string, error) {
	key := pkg.ExtractOwnerRepo(repoWithOwner)
	if r, ok := latestCache[key]; ok {
		return r.sha, r.tag, r.err
	}
	sha, tag, err := GetActionHashByVersion(repoWithOwner, latestVersion)
	latestCache[key] = latestResult{sha: sha, tag: tag, err: err}
	return sha, tag, err
}

// processPinnedActionToLatest resolves an already-pinned action to the newest release. It returns the
// rewritten "owner/repo@<sha> #<tag>" ref and whether the resolved SHA differs from the existing pin
// (changed == false means the action is already on the latest SHA and can be skipped).
func processPinnedActionToLatest(action string) (string, bool, error) {
	repoWithOwner, err := pkg.RepoFromPinnedRef(action)
	if err != nil {
		return "", false, err
	}
	sha, tag, err := resolveLatest(repoWithOwner)
	if err != nil {
		return "", false, err
	}
	resolvedSha := strings.TrimSpace(sha)
	oldSha := strings.TrimPrefix(action, repoWithOwner+"@")
	changed := resolvedSha != oldSha
	return fmt.Sprintf("%s@%s #%s", repoWithOwner, resolvedSha, strings.TrimSpace(tag)), changed, nil
}

// writePinnedActionUpdate rewrites the first occurrence of a SHA-pinned ref (and its trailing
// comment) in fileName with actionWithSha, warning without writing when the ref cannot be located.
func writePinnedActionUpdate(fileName string, action string, actionWithSha string) {
	if action == "" || actionWithSha == "" {
		logger.Error("action or actionWithSha are empty", logger.Args("action:", action, "actionWithSha:", actionWithSha))
		return
	}
	content, err := os.ReadFile(fileName)
	if err != nil {
		logger.Warn("Error reading file", logger.Args("file:", fileName, "error:", err))
		return
	}
	modifiedContent, matched := pkg.ReplaceActionRef(string(content), action, actionWithSha)
	if !matched {
		logger.Warn("Resolved latest but could not locate pinned ref in file text; leaving unchanged",
			logger.Args("file:", fileName, "action:", action))
		return
	}
	if err := os.WriteFile(fileName, []byte(modifiedContent), 0600); err != nil {
		logger.Warn("Error writing file", logger.Args("file:", fileName, "error:", err))
		return
	}
	logger.Info("Re-pinning action to latest", logger.Args("action:", action, "updated:", actionWithSha))
}
