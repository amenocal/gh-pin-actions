---
name: local-validation
description: Run the repository's CI checks locally (go build/vet/test, gofmt, Prettier, markdownlint, super-linter) before pushing. Use when preparing a PR or when the Source Linter / Build checks fail.
---

# Local validation

CI runs two workflows: `.github/workflows/ci.yml` (Build) and `.github/workflows/linter.yml`
(Source Linter, super-linter `slim-v7`). Reproduce both locally before pushing.

## Line endings

The repository is LF only (`.gitattributes`, `.editorconfig`). Do not use `core.autocrlf`:

```sh
git config core.autocrlf false
```

If files in your working tree still have CRLF, rewrite them from the index:
`git rm --cached -r -q . && git reset --hard -q` (commit or stash your changes first).

## Go (mirrors ci.yml)

```sh
go build -v .
go vet ./...
go test -v ./...
gofmt -l .        # must print nothing
```

golangci-lint (errcheck, gocyclo, goimports, ...) is what super-linter's Go checks use:

```sh
golangci-lint run ./...
```

Keep cyclomatic complexity of functions at 15 or lower and always check returned errors,
including in tests.

## Markdown and YAML/JSON formatting

```sh
npx --yes prettier --check .
npx --yes prettier --write README.md      # fix a single file
npx --yes markdownlint-cli README.md
```

Common markdownlint pitfalls: MD028 (empty line between two blockquotes; join them with a
bare `>` line) and Prettier rewriting blockquotes.

## Super-linter with the repository config

The configuration lives in the `env:` block of `.github/workflows/linter.yml`. Mirror it when
running the same image. Use `wslc` (WSL containers) or `docker`; the commands are the same.

```sh
# one-time: ghcr.io needs authentication
gh auth token | wslc registry login --password-stdin -u <github-user> ghcr.io
wslc pull ghcr.io/super-linter/super-linter:slim-v7

wslc run --rm -v "${PWD}:/tmp/lint" \
  -e RUN_LOCAL=true \
  -e VALIDATE_ALL_CODEBASE=true \
  -e DEFAULT_BRANCH=main \
  -e VALIDATE_JSCPD=false \
  -e VALIDATE_GO=false \
  ghcr.io/super-linter/super-linter:slim-v7
```

PowerShell: replace `${PWD}` with `"$((Get-Location).Path)"` and put the command on one line.

- Keep the `-e` flags in sync with `linter.yml` (`VALIDATE_JSCPD`, `VALIDATE_GO`, ...).
- CI only lints changed files (`VALIDATE_ALL_CODEBASE: false`); local runs lint everything, so
  warnings in files you did not touch can be ignored.
- To lint a single language, add e.g. `-e VALIDATE_MARKDOWN=true` (this disables the others),
  or `-e VALIDATE_MARKDOWN_PRETTIER=true`.
- Look for `[ERROR]` lines and the `Errors found in <LINTER>` summary at the end.
