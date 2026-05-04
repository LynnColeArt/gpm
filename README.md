# gpm

`gpm` is an npm-style package, workspace, and binary app manager for Go, written entirely in Go. It adds scripts, app installs, and workspace-oriented tooling on top of the standard Go toolchain without trying to replace `go.mod`.

The goal is to make common Go workflows feel more cohesive and discoverable: initialize a project, run scripts, install Go CLI apps at user or global scope, and eventually manage dependencies, workspaces, publishing, and registry flows from one Go-native CLI.

## Install

```bash
go install github.com/LynnColeArt/gpm/cmd/gpm@latest
```

## Status

`gpm` has completed the original Phase 0 scaffold and now has a working first pass on the Phase 1 dependency UX layer.

Implemented today:

- project bootstrap with `gpm init`
- script execution with `gpm run`
- environment diagnostics with `gpm doctor`
- user/global Go app installs with `gpm app install`, `gpm app list`, and `gpm app uninstall`
- module dependency management with `gpm add`, `gpm remove`, `gpm install`, `gpm update`, and `gpm outdated`
- project-local tool pinning and execution through `.gpm/tools/bin`
- `gpm.lock` generation, drift detection, and stale-lock enforcement for `gpm run` and `gpm exec`

## Current Commands

```text
gpm init
gpm add
gpm remove
gpm install
gpm update
gpm outdated
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

`gpm add`, `gpm remove`, `gpm install`, and `gpm update` keep `gpm.lock` in sync with the current `go.mod` state plus declared tool dependencies. `gpm doctor` checks whether `gpm.lock` has drifted, and `gpm run`/`gpm exec` now refuse to use stale project state until `gpm install` refreshes it. The initial lockfile is intentionally narrow: it captures resolved module requirements and pinned tool targets without trying to replace `go.sum`.

## Dependency Visibility

`gpm outdated` shows actionable module dependency state from the current build list. By default it reports direct dependencies with available updates or other upgrade-related issues such as retraction or deprecation notices. Use `gpm outdated --all` to include indirect dependencies too.

## Documentation

- [docs/current-state.md](docs/current-state.md): current shipped surface, validation snapshot, and next likely work
- [docs/gpm-spec.md](docs/gpm-spec.md): product and architecture spec
- [docs/phase-0-contract.md](docs/phase-0-contract.md): original Phase 0 implementation contract
- [docs/registry-index-design.md](docs/registry-index-design.md): registry/index architecture direction
- [docs/registry-metadata-schema.md](docs/registry-metadata-schema.md): proposed registry metadata schema

## Notes

The repo has a pushed checkpoint tag at `phase-1-foundation`, and development is currently proceeding directly on `main` in small slices.
