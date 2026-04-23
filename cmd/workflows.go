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
	latestWorkflows    bool
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
	workflowsCmd.Flags().BoolVar(&latestWorkflows, "latest", false, "pin actions to the latest available release tags")
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

	resolutionCache := newActionResolutionCache()
	for _, file := range workflowFiles {
		processActionsYaml(file, latestWorkflows, resolutionCache)
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

func processActionsYaml(workflow string, latestMode bool, resolutionCache pkg.ActionResolutionCache) {
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
				actionWithSha, shouldUpdate, err := processActionWithCache(action, latestMode, resolutionCache)
				if err != nil {
					logger.Warn("Nothing will be updated", logger.Args("action:", action, "error:", err))
					continue
				}
				if !shouldUpdate {
					continue
				}
				if latestMode {
					logger.Info("Updating to latest", logger.Args("from:", action, "to:", actionWithSha))
				}
				writeModifiedWorkflowToFile(pinnedWorkflow, action, actionWithSha)
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

func writeModifiedWorkflowToFile(fileName string, action string, actionWithSha string) {
	if action == "" || actionWithSha == "" {
		logger.Error("action or actionWithSha are empty", logger.Args("action:", action, "actionWithSha:", actionWithSha))
		return
	}
	newFileContent, err := os.ReadFile(fileName)
	if err != nil {
		logger.Warn("Error reading file", logger.Args("file:", fileName, "error:", err))
		return
	}
	// Match the action plus any trailing inline #comment on the same line so that
	// re-pinning an already-pinned action (e.g. with --latest) replaces the old
	// version comment instead of leaving it alongside the new one.
	pattern := regexp.MustCompile(regexp.QuoteMeta(action) + `(?:[ \t]+#[^\r\n]*)?`)
	modifiedContent := pattern.ReplaceAllLiteralString(string(newFileContent), actionWithSha)
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

func newActionResolutionCache() pkg.ActionResolutionCache {
	return pkg.NewActionResolutionCache()
}

func warnUsesResolutionFailure(action string, err error) {
	if logger == nil {
		return
	}
	logger.Warn("Unable to resolve action, keeping original uses value", logger.Args("action:", action, "error:", err))
}

type workflowResolverBackends struct {
	resolveByVersion pkg.ActionVersionResolver
	resolveByBranch  pkg.ActionBranchResolver
}

func defaultWorkflowResolverBackends() workflowResolverBackends {
	return workflowResolverBackends{
		resolveByVersion: GetActionHashByVersion,
		resolveByBranch:  GetBranchHash,
	}
}

func processActionWithCache(action string, latestMode bool, resolutionCache pkg.ActionResolutionCache) (string, bool, error) {
	backends := defaultWorkflowResolverBackends()
	return processActionWithCacheWithResolvers(action, latestMode, resolutionCache, backends.resolveByVersion, backends.resolveByBranch)
}

func processActionWithCacheWithResolvers(action string, latestMode bool, resolutionCache pkg.ActionResolutionCache, resolveByVersion pkg.ActionVersionResolver, resolveByBranch pkg.ActionBranchResolver) (string, bool, error) {
	return pkg.ProcessActionWithCache(action, latestMode, resolutionCache, resolveByVersion, resolveByBranch, warnUsesResolutionFailure)
}
