package lockfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LynnColeArt/gpm/internal/gotool"
	"github.com/LynnColeArt/gpm/internal/manifest"
)

const (
	FileName    = "gpm.lock"
	LockVersion = 1
)

var ErrNotFound = errors.New("gpm lockfile not found")

type File struct {
	LockVersion  int                 `json:"lockVersion"`
	ModulePath   string              `json:"modulePath"`
	GoVersion    string              `json:"goVersion,omitempty"`
	Dependencies []Dependency        `json:"dependencies"`
	Tools        map[string]ToolLock `json:"tools"`
}

type Dependency struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	Indirect bool   `json:"indirect,omitempty"`
}

type ToolLock struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Binary  string `json:"binary,omitempty"`
	Target  string `json:"target"`
}

type CheckResult struct {
	LockPath string
	Exists   bool
	Issues   []string
}

func Build(projectRoot string, file *manifest.File) (*File, error) {
	if file == nil {
		return nil, fmt.Errorf("manifest is nil")
	}

	modData, err := gotool.Output(projectRoot, nil, "mod", "edit", "-json")
	if err != nil {
		return nil, fmt.Errorf("read go.mod state: %w", err)
	}

	var modFile struct {
		Module struct {
			Path string `json:"Path"`
		} `json:"Module"`
		Go      string `json:"Go"`
		Require []struct {
			Path     string `json:"Path"`
			Version  string `json:"Version"`
			Indirect bool   `json:"Indirect"`
		} `json:"Require"`
	}
	if err := json.Unmarshal(modData, &modFile); err != nil {
		return nil, fmt.Errorf("parse go.mod state: %w", err)
	}

	deps := make([]Dependency, 0, len(modFile.Require))
	for _, requirement := range modFile.Require {
		deps = append(deps, Dependency{
			Path:     requirement.Path,
			Version:  requirement.Version,
			Indirect: requirement.Indirect,
		})
	}
	sort.Slice(deps, func(i, j int) bool {
		return deps[i].Path < deps[j].Path
	})

	tools := make(map[string]ToolLock, len(file.Tools))
	for name, tool := range file.Tools {
		tools[name] = ToolLock{
			Module:  tool.Module,
			Version: tool.Version,
			Binary:  tool.Binary,
			Target:  tool.Module + "@" + tool.Version,
		}
	}

	return &File{
		LockVersion:  LockVersion,
		ModulePath:   modFile.Module.Path,
		GoVersion:    modFile.Go,
		Dependencies: deps,
		Tools:        tools,
	}, nil
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse lockfile: %w", err)
	}

	if file.Dependencies == nil {
		file.Dependencies = []Dependency{}
	}
	if file.Tools == nil {
		file.Tools = map[string]ToolLock{}
	}

	return &file, nil
}

func Check(projectRoot string, file *manifest.File) (CheckResult, error) {
	lockPath := Path(projectRoot)

	current, err := Build(projectRoot, file)
	if err != nil {
		return CheckResult{}, err
	}

	locked, err := Load(lockPath)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return CheckResult{
				LockPath: lockPath,
				Exists:   false,
			}, nil
		}
		return CheckResult{}, err
	}

	return CheckResult{
		LockPath: lockPath,
		Exists:   true,
		Issues:   compareFiles(locked, current),
	}, nil
}

func (result CheckResult) Current() bool {
	return result.Exists && len(result.Issues) == 0
}

func (result CheckResult) Summary() string {
	if len(result.Issues) == 0 {
		return ""
	}
	return strings.Join(result.Issues, "; ")
}

func Save(path string, file *File) error {
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}
	return nil
}

func Path(projectRoot string) string {
	return filepath.Join(projectRoot, FileName)
}

func compareFiles(locked, current *File) []string {
	issues := []string{}

	if locked.LockVersion != current.LockVersion {
		issues = append(issues, fmt.Sprintf("lockVersion mismatch: lock=%d current=%d", locked.LockVersion, current.LockVersion))
	}
	if locked.ModulePath != current.ModulePath {
		issues = append(issues, fmt.Sprintf("modulePath mismatch: lock=%s current=%s", locked.ModulePath, current.ModulePath))
	}
	if locked.GoVersion != current.GoVersion {
		issues = append(issues, fmt.Sprintf("goVersion mismatch: lock=%s current=%s", locked.GoVersion, current.GoVersion))
	}

	if dependencyIssue := compareDependencies(locked.Dependencies, current.Dependencies); dependencyIssue != "" {
		issues = append(issues, dependencyIssue)
	}

	for _, toolIssue := range compareTools(locked.Tools, current.Tools) {
		issues = append(issues, toolIssue)
	}

	return issues
}

func compareDependencies(locked, current []Dependency) string {
	if len(locked) != len(current) {
		return fmt.Sprintf("dependency count mismatch: lock=%d current=%d", len(locked), len(current))
	}

	for i := range locked {
		if locked[i] != current[i] {
			return fmt.Sprintf(
				"dependency mismatch at index %d: lock=%s@%s current=%s@%s",
				i,
				locked[i].Path,
				locked[i].Version,
				current[i].Path,
				current[i].Version,
			)
		}
	}

	return ""
}

func compareTools(locked, current map[string]ToolLock) []string {
	issues := []string{}

	seen := map[string]struct{}{}
	names := make([]string, 0, len(locked)+len(current))
	for name := range locked {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for name := range current {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		lockedTool, lockedOK := locked[name]
		currentTool, currentOK := current[name]

		switch {
		case !lockedOK:
			issues = append(issues, fmt.Sprintf("tool %q missing from lockfile", name))
		case !currentOK:
			issues = append(issues, fmt.Sprintf("tool %q missing from current manifest state", name))
		case lockedTool != currentTool:
			issues = append(
				issues,
				fmt.Sprintf(
					"tool %q mismatch: lock=%s current=%s",
					name,
					lockedTool.Target,
					currentTool.Target,
				),
			)
		}
	}

	return issues
}
