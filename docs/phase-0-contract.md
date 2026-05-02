# gpm Phase 0 Implementation Contract

Status: Draft 0.1
Date: 2026-05-02

## 1. Scope

Phase 0 establishes the smallest useful version of `gpm`:

- a stable project manifest
- project discovery rules
- a usable script runner
- a basic environment diagnostics command
- a CLI skeleton that can grow without rewrites

Phase 0 includes exactly these commands:

- `gpm init`
- `gpm run`
- `gpm doctor`
- `gpm app install`
- `gpm app list`
- `gpm app uninstall`

Phase 0 explicitly does not include:

- `gpm add`
- `gpm remove`
- `gpm install`
- `gpm update`
- lockfile generation
- workspace orchestration
- registry authentication
- publishing
- vulnerability audit

## 2. Files and Ownership

Phase 0 introduces these files:

- `gpm.json`: project manifest
- `~/.config/gpm/config.json`: optional user config
- `~/.config/gpm/apps.json`: installed app state

Phase 0 does not create `gpm.lock`.

`go.mod` remains the canonical module definition if it exists.

## 3. Root Discovery Rules

When a command needs project context, `gpm` searches upward from the current working directory for `gpm.json`.

Rules:

1. The first directory containing `gpm.json` is the project root.
2. `gpm run` requires a project root and fails if none is found.
3. `gpm doctor` runs from the current directory even if no project root is found, but reports the missing manifest as a failure.
4. `gpm init` always writes `gpm.json` into the current working directory.

Phase 0 does not support nested workspace selection.

## 4. Manifest Schema

The Phase 0 manifest format is JSON and is stored at `gpm.json`.

Supported fields:

```json
{
  "name": "gpm",
  "version": "0.1.0",
  "private": true,
  "description": "Go-native package and workspace manager",
  "scripts": {
    "build": "go build ./...",
    "test": "go test ./...",
    "fmt": "gofmt -w ."
  },
  "workspaces": [],
  "registries": {},
  "tools": {}
}
```

Field contracts:

- `name`: required non-empty string
- `version`: required non-empty string
- `private`: required boolean
- `description`: optional string
- `scripts`: optional map of script name to raw command string
- `workspaces`: reserved for future phases
- `registries`: reserved for future phases
- `tools`: reserved for future phases

Important:

- `gpm.json` is not a dependency source of truth.
- Phase 0 does not store dependency declarations in `gpm.json`.

## 5. User Config Contract

Phase 0 uses a JSON user config file to stay within the Go standard library.

Path:

- `~/.config/gpm/config.json`

Phase 0 config is optional.

Supported fields:

```json
{
  "defaultRegistry": "",
  "registries": {},
  "binaries": {
    "userDir": "",
    "globalDir": ""
  }
}
```

If the file exists but is invalid JSON, `gpm doctor` reports a failure.

## 6. Command Grammar

### 6.1 `gpm init`

Grammar:

```text
gpm init [--force] [--name <name>] [--private=<true|false>]
```

Behavior:

1. Write `gpm.json` into the current directory.
2. If `gpm.json` already exists and `--force` is not set, fail.
3. If `--name` is provided, use it as the manifest `name`.
4. Otherwise, if `go.mod` exists in the current directory, use its module path.
5. Otherwise, use the current directory basename.
6. Default `private` to `true`.
7. Seed default scripts for `build`, `test`, `fmt`, and `tidy`.
8. Do not create or edit `go.mod`.

Phase 0 default scripts:

```json
{
  "build": "go build ./...",
  "test": "go test ./...",
  "fmt": "gofmt -w .",
  "tidy": "go mod tidy"
}
```

### 6.2 `gpm run`

Grammar:

```text
gpm run <script-name> [-- <arg>...]
```

Behavior:

1. Discover the project root.
2. Load and validate `gpm.json`.
3. Resolve the named script from `scripts`.
4. Execute the script from the project root directory.
5. Pass through trailing arguments only when separated by `--`.
6. Inherit stdin, stdout, stderr, and ambient environment.
7. Add these environment variables:

```text
GPM_PROJECT_ROOT=<absolute path>
GPM_MANIFEST_PATH=<absolute path to gpm.json>
GPM_SCRIPT_NAME=<selected script name>
```

Phase 0 execution model:

- raw script strings are executed through the platform shell
- Unix: `/bin/sh -c`
- Windows: `cmd.exe /C`

This is intentionally a thin compatibility layer for Phase 0, not the final structured scripts model.

### 6.3 `gpm doctor`

Grammar:

```text
gpm doctor
```

Behavior:

1. Inspect the current directory and nearest project root if available.
2. Report human-readable checks with `ok`, `warn`, or `fail`.
3. Exit non-zero if any `fail` checks are present.

Checks:

- `go` binary available on `PATH`
- `gpm.json` present
- manifest parses and validates
- `go.mod` present in the project root
- user config file path resolves
- user config file parses if present
- app state file parses if present
- user binary directory resolves
- user binary directory appears on `PATH`

Severity rules:

- missing `go` binary: `fail`
- missing `gpm.json`: `fail`
- invalid `gpm.json`: `fail`
- missing `go.mod`: `warn`
- missing user config: `warn`
- invalid user config: `fail`
- missing app state: `warn`
- invalid app state: `fail`
- user bin directory not on `PATH`: `warn`

### 6.4 `gpm app install`

Grammar:

```text
gpm app install <package-or-module>[@<version>] [--scope user|global] [--bin-dir <path>]
```

Behavior:

1. Resolve the installation target.
2. If the target does not include `@<version>` and is not a local path, append `@latest`.
3. Resolve the destination bin directory from `--bin-dir`, user config, or scope defaults.
4. Create the bin directory if needed.
5. Run `go install` with `GOBIN` pointing at the resolved bin directory.
6. Stream install output to the terminal.

Scope defaults:

- `user`: `~/.local/bin`
- `global`: `/usr/local/bin`

Phase 0 notes:

- `gpm app install` is distinct from project dependency installation.
- `gpm app install` may require elevated privileges when `--scope global` is used.
- local path installs like `./cmd/gpm` are allowed and do not get `@latest` appended.
- successful installs are recorded in `~/.config/gpm/apps.json`

### 6.5 `gpm app list`

Grammar:

```text
gpm app list [--scope user|global|all]
```

Behavior:

1. Load installed app state from `~/.config/gpm/apps.json`.
2. Filter records by scope.
3. Print installed apps with binary path and existence status.
4. Succeed with a friendly message when no installed apps are recorded.

### 6.6 `gpm app uninstall`

Grammar:

```text
gpm app uninstall <name-or-target> [--scope user|global]
```

Behavior:

1. Load installed app state.
2. Match the argument against binary name, requested target, or normalized target.
3. Remove the installed binary.
4. Remove the matching receipt from `~/.config/gpm/apps.json`.
5. Exit non-zero if no matching install record is found.

## 7. Exit Codes

- `0`: success
- `1`: command or validation failure
- subprocess exit code from `gpm run` when a script exits non-zero

## 8. Initial Package Layout

Phase 0 code layout:

```text
cmd/gpm/main.go
internal/apps/
internal/cli/
internal/config/
internal/manifest/
internal/project/
```

Responsibilities:

- `cli`: argument parsing, command dispatch, terminal output
- `apps`: binary app install target resolution and bin-directory logic
- `config`: user config path resolution and loading
- `manifest`: `gpm.json` data model, validation, load, save
- `project`: root discovery and local Go module inspection

## 9. Deferred Decisions

These are intentionally deferred past Phase 0:

- TOML vs JSON for long-term config
- lockfile format
- workspace filtering syntax
- registry client protocol
- publish pipeline behavior
- structured script declarations

## 10. Acceptance Criteria

Phase 0 is complete when:

1. `go build ./...` succeeds.
2. `gpm init` creates a valid `gpm.json`.
3. `gpm run <script>` executes project scripts from the project root.
4. `gpm app install` can install a Go binary into a resolved user bin directory.
5. `gpm app list` reports recorded installs and missing binaries clearly.
6. `gpm app uninstall` removes an installed binary and its recorded state.
7. `gpm doctor` reports actionable status for a project directory.
8. The implementation stays aligned with the product boundary from `docs/gpm-spec.md`.
