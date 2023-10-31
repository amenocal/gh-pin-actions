package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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
	for _, job := range wf.Jobs {
		for i, step := range job.Steps {
			logger.Info("Processing Step", logger.Args("step:", fmt.Sprintf("%d %s", i+1, step.Name)))
			if step.Uses != "" {
				actionWithVersion := step.Uses
				if !strings.Contains(actionWithVersion, "@v") {
					logger.Warn("actions is not in `@v` version format... Skipping", logger.Args("action:", actionWithVersion))
					continue
				}
				actionSplit := strings.Split(actionWithVersion, "@v")
				if len(actionSplit) > 1 {
					actionVersion := actionSplit[1]
					actionVersion = pkg.ProcessActionsVersion(actionVersion)
					commitSha, tagVersion, err := GetActionHashByVersion(actionSplit[0], actionVersion)
					actionWithSha := fmt.Sprintf("%s@%s #%s", actionSplit[0], commitSha, tagVersion)
					if err != nil {
						logger.Error("Error getting commit sha for action", logger.Args("action:", actionWithVersion, "error:", err))
						logger.Warn("Nothing will be updated")
						actionWithSha = actionWithVersion
					}
					logger.Info("Replacing action with sha", logger.Args("action:", actionWithVersion, "sha:", actionWithSha))
					writeModifiedWorkflowToFile(pinnedWorkflow, actionWithVersion, actionWithSha)
				}
			}
		}
	}
	if err == nil {
		fmt.Println("Done! Please review the changes in the following file:", pinnedWorkflow)
	}

}

func writeModifiedWorkflowToFile(fileName string, actionWithVersion string, actionWithSha string) {

	newFileContent, err := os.ReadFile(fileName)
	if err != nil {
		logger.Warn("Error reading file", logger.Args("file:", fileName, "error:", err))
	}
	modifiedContent := strings.Replace(string(newFileContent), actionWithVersion, actionWithSha, 1)
	err = os.WriteFile(fileName, []byte(modifiedContent), 0644)
	if err != nil {
		logger.Warn("Error creating file", logger.Args("file:", fileName, "error:", err))
	}
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
