/*
Copyright © 2023 Alejandro Menocal
*/
package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/amenocal/gh-pin-actions/pkg"
	"github.com/cli/go-gh/v2"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var (
	rootCmd = &cobra.Command{
		Use:   "gh pin-actions",
		Short: "pins actions to a specific sha",
		Long: `gh pin-actions is a CLI tool that pins actions to a specific sha
		You can specify the repository and the version of the action you want to pin to and it will 
		return the pinnable action in the format owner/repo@sha #version`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		Run: ActionsPin,
	}
	repository string
	version    string
	debug      bool
	branchName string
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gh-pin-actions.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringVarP(&repository, "repository", "r", "", "repository in the owner/repo format")
	if err := rootCmd.MarkFlagRequired("repository"); err != nil {
		logger.Fatal("error marking repository flag as required", logger.Args("error:", err))
	}

	rootCmd.Flags().StringVarP(&version, "version", "v", "latest", "version of the tag to pin to (ex. 3; 3.1; 3.1.1)")

	rootCmd.Flags().StringVarP(&branchName, "branch", "b", "", "branch name to pin to")

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug mode - set logger to debug level")
}

func ActionsPin(cmd *cobra.Command, args []string) {
	var shaCommit string
	var tagVersion string
	var err error
	if debug {
		logger = pterm.DefaultLogger.WithLevel(pterm.LogLevelDebug)
	} else {
		logger = pterm.DefaultLogger.WithLevel(pterm.LogLevelWarn)
	}

	if branchName != "" {
		fmt.Println("Branch name:", branchName)
		tagVersion = branchName
		shaCommit, err = GetBranchHash(repository, branchName)
	} else {
		isVersionFormat, _ := regexp.MatchString(`^v?\d+(\.\d+)?(\.\d+)?$`, version)
		if version == "latest" || version == "" || isVersionFormat {
			fmt.Println("Version:", version)
			shaCommit, tagVersion, err = GetActionHashByVersion(repository, version)
		} else {
			logger.Fatal("version flag must be in the format v1, v1.1, or v1.1.1", logger.Args("version received", version),
				logger.Args("recommendation", "use the --branch flag to pin to a branch"))
		}
	}

	if err != nil {
		logger.Error("Unable to get sha of Version", logger.Args("version:", version), logger.Args("error:", err))
		os.Exit(0)
	}
	pinnableAction := fmt.Sprintf("%s@%s #%s", repository, strings.TrimSpace(shaCommit), strings.TrimSpace(tagVersion))
	fmt.Println(pinnableAction)
	// fmt.Println(string(shaCommit.String()))
}

func GetActionHashByVersion(repository string, version string) (string, string, error) {
	var tagVersionBuffer bytes.Buffer
	var tagVersion string
	var std_err bytes.Buffer
	var err error

	// Remove 'v' from the version string if sent through command line
	version = strings.TrimPrefix(version, "v")
	// Check to see if value received is latest version or a specific version
	if version == "latest" || version == "" {
		tagVersionBuffer, std_err, err = gh.Exec("release", "view", "-R", repository, "--json", "tagName", "--jq", ".tagName")
		tagVersion = tagVersionBuffer.String()
	} else {
		tagVersion, err = GetLatestPatchVersion(repository, version)
	}
	if err != nil {
		logger.Error("Unable to get latest release tag", logger.Args("error:", std_err.String()))
		return "", tagVersion, err
	}
	// for me the cliArgs to get the commit sha based on the tag Version
	cliArgs := fmt.Sprintf(".[] | select(.name == \"%s\") | .commit.sha", strings.TrimSpace(tagVersion))
	cliOptions := fmt.Sprintf("repos/%s/tags", repository)
	shaCommit, std_err, err := gh.Exec("api", cliOptions, "--jq", cliArgs)
	if err != nil {
		return "", tagVersion, err
	}
	if shaCommit.String() == "" {
		//logger.Error("Version tag does not exist, no hash found", logger.Args("version", tagVersion))
		return "", tagVersion, errors.New("version tag does not exist")
	}
	sha := strings.TrimSpace(shaCommit.String())
	return sha, tagVersion, nil
}

func GetLatestPatchVersion(repository string, version string) (string, error) {
	if strings.Count(version, ".") == 2 {
		return fmt.Sprintf("v%s", version), nil
	}
	cliArgs := ".[] | .name"
	cliOptions := fmt.Sprintf("repos/%s/tags", repository)
	tagsBuffer, std_err, err := gh.Exec("api", cliOptions, "--jq", cliArgs)
	if err != nil {
		logger.Error("Issue with gh api and getting specific major.minor.version tag", logger.Args("error:", std_err.String(), "action", fmt.Sprintf("%s@%s", repository, version)))
		return fmt.Sprintf("v%s", version), err
	}

	tagsString := tagsBuffer.String()
	tags := strings.Split(tagsString, "\n")

	newVersion, err := pkg.FindHighestPatchVersion(tags, version)
	if err != nil {
		logger.Error("Unable to find highest patch version", logger.Args("error:", err))
		return "", err
	}

	return newVersion, nil

}

func GetBranchHash(repository string, branch string) (string, error) {
	cliArgs := ".commit.sha"
	cliOptions := fmt.Sprintf("repos/%s/branches/%s", repository, branch)
	shaCommit, std_err, err := gh.Exec("api", cliOptions, "--jq", cliArgs)
	if err != nil {
		logger.Error("Issue with gh api and getting specific branch hash", logger.Args("error:", std_err.String(), "action", fmt.Sprintf("%s@%s", repository, branch)))
		return "", err
	}
	return shaCommit.String(), nil
}
