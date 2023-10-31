# gh-pin-actions

`gh-pin-actions` is a GitHub CLI extension that allows you to pin GitHub Actions based on their version.

## Installation

To install `gh-pin-actions`, run the following command:

`gh extension install amenocal/gh-pin-actions`

## Usage

### Single Action

To use `gh pin-actions`, run the following command:

```sh
gh pin-actions --help
gh pin-actions is a CLI tool that pins actions to a specific sha
                You can specify the repository and the version of the action you want to pin to and it will 
                return the pinnable action in the format owner/repo@sha #version

Usage:
  gh pin-actions [flags]
  gh [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  workflows   Updates all .github/workflows to pin actions to a specific sha

Flags:
  -d, --debug               debug mode - set logger to debug level
  -h, --help                help for gh
  -r, --repository string   repository in the owner/repo format
  -v, --version string      version of the tag to pin to (ex. 3; 3.1; 3.1.1) (default "latest")

Use "gh [command] --help" for more information about a command.
```

Example: 

```sh
gh pin-actions -r actions/checkout -v 3
```

### GitHub Actions Workflows

```sh
 gh pin-actions workflows -h
Update all workflow files in .github/workflows and reads every 
                action with version in the workflow file and replaces it with the sha of the specific version

Usage:
  gh workflows [flags]

Flags:
  -h, --help   help for workflows

Global Flags:
  -d, --debug   debug mode - set logger to debug level
```

Example:

```sh
gh pin-actions workflows
```

## Contributing

Contributions to `gh-pin-actions` are welcome! Please submit a pull request or create an issue to contribute.

## License

`gh-pin-actions` is licensed under the [MIT License](LICENSE).