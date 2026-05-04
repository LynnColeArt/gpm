# gpm Current State

Status: Active Phase 1 work
Date: 2026-05-04

## Summary

`gpm` is no longer just a product spec and CLI skeleton. The repo now has a working Go CLI with a usable Phase 0 baseline and a meaningful first pass at the Phase 1 dependency UX layer.

The current implementation still follows the core product boundary:

- `go.mod` and `go.sum` remain the canonical module state
- `gpm.json` is the product manifest
- `gpm.lock` is the higher-level package-manager view
- `gpm` orchestrates the Go toolchain rather than replacing it

## Repo Facts

- module path: `github.com/LynnColeArt/gpm`
- primary branch: `main`
- remote: `https://github.com/LynnColeArt/gpm.git`
- milestone tag: `phase-1-foundation`

Recent pushed milestones:

- `5312f66`: initial CLI scaffold
- `495524d`: Phase 1 dependency foundation
- `7b381f8`: `gpm.lock` drift detection in `doctor`
- `607fa4e`: stale-lock enforcement in `run` and `exec`
- `b9e1236`: `gpm outdated`

## Implemented Surface

Current commands:

- `gpm init`
- `gpm add`
- `gpm remove`
- `gpm install`
- `gpm update`
- `gpm outdated`
- `gpm exec`
- `gpm run`
- `gpm doctor`
- `gpm app install`
- `gpm app list`
- `gpm app uninstall`

Current behaviors worth preserving:

- `gpm add` uses the Go toolchain to add a module and then rewrites `gpm.lock`
- `gpm remove` uses `go get <module>@none`, may run `go mod tidy`, and then rewrites `gpm.lock`
- `gpm install` hydrates module dependencies, installs pinned tools into `.gpm/tools/bin`, and writes `gpm.lock`
- `gpm update` upgrades direct module requirements by default, accepts exact targets, reinstalls tools, and rewrites `gpm.lock`
- `gpm outdated` reports actionable module updates and upgrade-related notices, defaulting to direct dependencies and supporting `--all`
- `gpm doctor` reports missing or drifted `gpm.lock` state
- `gpm run` and `gpm exec` refuse to proceed when `gpm.lock` is missing or stale and direct the user to `gpm install`
- `gpm app install` supports user and global install scopes through explicit `GOBIN` management

## Manifest and Lockfile State

`gpm.json` currently supports:

- package metadata
- scripts
- reserved workspace and registry fields
- typed `tools` declarations with `module`, `version`, and optional `binary`

`gpm.lock` currently captures:

- `lockVersion`
- module path
- Go version
- resolved `go.mod` requirements
- declared tool targets

What `gpm.lock` does not yet do:

- registry source pinning
- workspace graph state
- richer integrity or policy metadata beyond what the Go toolchain already owns

## Validation Snapshot

The implementation has been repeatedly validated with:

- `go test ./...`
- `go build ./...`

Additional live verification that has already been exercised during this phase:

- `gpm add -> gpm update -> gpm remove` with `gpm.lock` changing accordingly
- project-local tool installation into `.gpm/tools/bin`
- `gpm exec` and `gpm run` using pinned tool binaries
- `gpm doctor` failing on drifted `gpm.lock`
- `gpm run` refusing stale lock state until `gpm install` refreshes it
- `gpm outdated` reporting available updates from live module state
- user-scope and custom-bin-dir `gpm app install` flows

## Known Gaps

Phase 1 is stronger now, but it is not complete in the broader product sense.

Important missing or incomplete areas:

- no workspace commands yet
- no registry/login/publish implementation yet
- no audit or policy engine yet
- no first-class tool upgrade/outdated story yet
- lockfile semantics are still intentionally narrow
- `gpm outdated` currently focuses on module dependencies, not declared tool targets

## Next Likely Slices

The most natural next steps from here are:

1. extend dependency visibility so tool declarations can surface upgrade information too
2. decide whether `gpm install` should grow a clearer reconcile-vs-refresh mode split
3. begin Phase 2 workspace commands once the Phase 1 dependency surface feels stable

## Canonical Docs

Use these docs as the primary re-entry points:

- [README.md](../README.md)
- [docs/gpm-spec.md](./gpm-spec.md)
- [docs/phase-0-contract.md](./phase-0-contract.md)
- [docs/registry-index-design.md](./registry-index-design.md)
- [docs/registry-metadata-schema.md](./registry-metadata-schema.md)
