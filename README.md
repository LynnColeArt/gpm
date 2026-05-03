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
gpm add
gpm remove
gpm install
gpm update
gpm exec
gpm run
gpm doctor
gpm app install
gpm app list
gpm app uninstall
```

## Tool Dependencies

`gpm` can pin repo-local tools in `gpm.json` and install them into `.gpm/tools/bin`:

```json
{
  "tools": {
    "golangci-lint": {
      "module": "github.com/golangci/golangci-lint/cmd/golangci-lint",
      "version": "v1.64.8"
    }
  }
}
```

Run `gpm install` to hydrate the tool binaries, then `gpm exec golangci-lint -- run` or call the tool directly from `gpm run` scripts. `gpm run` automatically prepends the managed tool bin dir to `PATH`.

If the installed executable name differs from the manifest key, set `binary` explicitly:

```json
{
  "tools": {
    "strgen": {
      "module": "golang.org/x/tools/cmd/stringer",
      "version": "v0.38.0",
      "binary": "stringer"
    }
  }
}
```

## Lockfile

`gpm add`, `gpm remove`, `gpm install`, and `gpm update` keep `gpm.lock` in sync with the current `go.mod` state plus declared tool dependencies. The initial lockfile is intentionally narrow: it captures resolved module requirements and pinned tool targets without trying to replace `go.sum`.

## Project Status

This repo is still in early scaffolding. The product direction and Phase 0 contract live in [docs/gpm-spec.md](docs/gpm-spec.md) and [docs/phase-0-contract.md](docs/phase-0-contract.md).
