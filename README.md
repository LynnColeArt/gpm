# gpm

`gpm` is an npm-style package, workspace, and binary app manager for Go, written entirely in Go. It adds scripts, app installs, and workspace-oriented tooling on top of the standard Go toolchain without trying to replace `go.mod`.

The goal is to make common Go workflows feel more cohesive and discoverable: initialize a project, run scripts, install Go CLI apps at user or global scope, and eventually manage dependencies, workspaces, publishing, and registry flows from one Go-native CLI.

## Install

```bash
go install github.com/LynnColeArt/gpm/cmd/gpm@latest
```

## Current Commands

```text
gpm init
gpm run
gpm doctor
gpm app install
gpm app list
gpm app uninstall
```

## Project Status

This repo is still in early scaffolding. The product direction and Phase 0 contract live in [docs/gpm-spec.md](docs/gpm-spec.md) and [docs/phase-0-contract.md](docs/phase-0-contract.md).
