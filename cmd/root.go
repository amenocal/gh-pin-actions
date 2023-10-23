/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var (
	rootCmd = &cobra.Command{
		Use:   "gh-pin-actions",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		Run: ActionsPin,
	}
	repository string
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
	rootCmd.MarkFlagRequired("repository")
}

func ActionsPin(cmd *cobra.Command, args []string) {
	latestRelease, std_err, err := gh.Exec("release", "view", "-R", repository, "--json", "tagName", "--jq", ".tagName")
	if err != nil {

		fmt.Println("error", std_err.String())
		return
	}
	cliArgs := fmt.Sprintf(".[] | select(.name == \"%s\") | .commit.sha", strings.TrimSpace(latestRelease.String()))
	cliOptions := fmt.Sprintf("repos/%s/tags", repository)
	shaCommit, std_err, err := gh.Exec("api", cliOptions, "--jq", cliArgs)
	if err != nil {
		fmt.Println("error", std_err.String())
		return
	}
	pinnableAction := fmt.Sprintf("%s@%s #%s", repository, strings.TrimSpace(shaCommit.String()), strings.TrimSpace(latestRelease.String()))
	fmt.Println(pinnableAction)
	// fmt.Println(string(shaCommit.String()))

}
