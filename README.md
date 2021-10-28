# Brev CLI

[![Keep a Changelog](https://img.shields.io/badge/changelog-Keep%20a%20Changelog-%23E05735)](CHANGELOG.md)
[![GitHub Release](https://img.shields.io/github/v/release/brevdev/brev-cli)](https://github.com/brevdev/brev-cli/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/brevdev/brev-cli.svg)](https://pkg.go.dev/github.com/brevdev/brev-cli)
[![go.mod](https://img.shields.io/github/go-mod/go-version/brevdev/brev-cli)](go.mod)
[![LICENSE](https://img.shields.io/github/license/brevdev/brev-cli)](LICENSE)
[![Build Status](https://img.shields.io/github/workflow/status/brevdev/brev-cli/build)](https://github.com/brevdev/brev-cli/actions?query=workflow%3Abuild+branch%3Amain)
[![Go Report Card](https://goreportcard.com/badge/github.com/brevdev/brev-cli)](https://goreportcard.com/report/github.com/brevdev/brev-cli)
[![Codecov](https://codecov.io/gh/brevdev/brev-cli/branch/main/graph/badge.svg)](https://codecov.io/gh/brevdev/brev-cli)

`Star` this repository if you find it valuable and worth maintaining.

`Watch` this repository to get notified about new releases, issues, etc.

# Install

!!! warning these commands run as sudo !!!

Linux & Mac

```
curl -sf -L https://raw.githubusercontent.com/brevdev/brev-cli/main/bin/install-latest.sh  -o install-brev.sh
chmod +x install-brev.sh
sudo ./install-brev.sh
```
or

```
sudo sh -c "`curl -sf -L https://raw.githubusercontent.com/brevdev/brev-cli/main/bin/install-latest.sh`"
```

# Uninstall

```
sudo rm /usr/local/bin/brev
rm -rf ~/.brev
rm -rf /tmp/brev
```

# Usage

## Completion


### zsh


```
mkdir -p ~/.zsh/completions
brev completion zsh > ~/.zsh/completions/_brev
echo fpath=~/.zsh/completions $fpath >> ~/.zshrc && fpath=(~/.zsh/completions $fpath)
autoload -U compinit && compinit
```

### bash

```
brev completion bash > /usr/local/etc/bash_completion.d/brev
```

# Development

## Build

make build

## example .env file

```
VERSION=unknown
BREV_API_URL=http://localhost:8080
# BREV_API_URL=https://ade5dtvtaa.execute-api.us-east-1.amazonaws.com
```

### Terminal

- `make` - execute the build pipeline.
- `make help` - print help for the [Make targets](Makefile).

### Visual Studio Code

`F1` → `Tasks: Run Build Task (Ctrl+Shift+B or ⇧⌘B)` to execute the build pipeline.

## Release

The release workflow is triggered each time a tag with `v` prefix is pushed.

_CAUTION_: Make sure to understand the consequences before you bump the major version. More info: [Go Wiki](https://github.com/golang/go/wiki/Modules#releasing-modules-v2-or-higher), [Go Blog](https://blog.golang.org/v2-go-modules).

## Maintainance

Remember to update Go version in [.github/workflows](.github/workflows), [Makefile](Makefile) and [devcontainer.json](.devcontainer/devcontainer.json).

Notable files:

- [devcontainer.json](.devcontainer/devcontainer.json) - Visual Studio Code Remote Container configuration,
- [.github/workflows](.github/workflows) - GitHub Actions workflows,
- [.github/dependabot.yml](.github/dependabot.yml) - Dependabot configuration,
- [.vscode](.vscode) - Visual Studio Code configuration files,
- [.golangci.yml](.golangci.yml) - golangci-lint configuration,
- [.goreleaser.yml](.goreleaser.yml) - GoReleaser configuration,
- [Dockerfile](Dockerfile) - Dockerfile used by GoReleaser to create a container image,
- [Makefile](Makefile) - Make targets used for development, [CI build](.github/workflows) and [.vscode/tasks.json](.vscode/tasks.json),
- [go.mod](go.mod) - [Go module definition](https://github.com/golang/go/wiki/Modules#gomod),
- [tools.go](tools.go) - [build tools](https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module).

## Contributing

Simply create an issue or a pull request.

## Distribute to Homebrew

Step 1: bump version (see top of Makefile)

Step 2: create homebrew distribution
```
> make dist-homebrew
```

Step 3: create GitHub release

Step 4: upload resultant tar.gz to GitHub release

Step 5: copy sha256 (output from step 2) and use it in a new update to https://github.com/brevdev/homebrew-tap
