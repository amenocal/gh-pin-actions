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
	logger *pterm.Logger
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
	// rootCmd.Flags().StringVarP(&repository, "repository", "r", "", "repository in the owner/repo format")
	// rootCmd.MarkFlagRequired("repository")

	// rootCmd.Flags().StringVarP(&version, "version", "v", "latest", "version of the tag to pin to (ex. 3; 3.1; 3.1.1)")
	// rootCmd.MarkFlagRequired("repository")

}

func processWorkflows(cmd *cobra.Command, args []string) {
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

	// Compile the regular expression before the loop
	regHash, err := regexp.Compile(`@[0-9a-f]{40}`)
	if err != nil {
		logger.Warn("Error compiling Hash regex", logger.Args("error:", err))
	}

	// Loop through all the jobs and steps
	for _, job := range wf.Jobs {
		for i, step := range job.Steps {
			logger.Trace("Processing Step", logger.Args("step:", fmt.Sprintf("%d %s", i+1, step.Name)))
			if action := step.Uses; action != "" {
				if matched := regHash.MatchString(action); matched {
					// Action already has a hash
					logger.Info("Action already has a hash", logger.Args("action:", action))
					continue
				} else {
					// Actions doesn't have a hash
					actionWithSha, err := processAction(action)
					if err != nil {
						logger.Warn("Nothing will be updated")
					}
					writeModifiedWorkflowToFile(pinnedWorkflow, action, actionWithSha)
				}
			}
		}
	}
	if err == nil {
		fmt.Println("Done! Please review the changes in the following file:", pinnedWorkflow)
	}

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
	err = os.WriteFile(fileName, []byte(modifiedContent), 0644)
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

	err = os.WriteFile(newFileName, []byte(content), 0644)
	if err != nil {
		logger.Warn("Error writing to file", logger.Args("file:", newFileName, "error:", err))
		return "", err
	}
	return newFileName, nil
}

func processActionWithVersion(actionWithVersion string) (string, error) {
	repoWithOwner, versionParsed, err := pkg.SplitActionString(actionWithVersion, "@v")
	if err != nil {
		return "", err
	}
	actionVersion := pkg.FormatVersion(versionParsed)
	commitSha, tagVersion, err := GetActionHashByVersion(repoWithOwner, actionVersion)
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
	branchRegex, err := regexp.Compile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+@[a-zA-Z0-9_-]+$`)
	if err != nil {
		logger.Warn("Error compiling branch regex", logger.Args("error:", err))
	}
	if strings.Contains(action, "@v") {
		// Action has a version
		actionWithVersion := action
		actionWithSha, err = processActionWithVersion(actionWithVersion)
		if err != nil {
			logger.Error("Error getting commit sha for action", logger.Args("action:", actionWithVersion, "error:", err))
			actionWithSha = actionWithVersion
			return actionWithSha, err
		}

	} else if branchRegex.MatchString(action) {
		// Action has a branch
		actionWithBranch := action
		actionWithSha, err = processActionWithBranch(actionWithBranch)
		if err != nil {
			logger.Error("Error getting commit sha for action with Branch", logger.Args("action:", actionWithBranch, "error:", err))
			actionWithSha = actionWithBranch
			return actionWithSha, err
		}
	}
	return actionWithSha, nil
}
