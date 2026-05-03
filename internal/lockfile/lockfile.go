package lockfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/LynnColeArt/gpm/internal/gotool"
	"github.com/LynnColeArt/gpm/internal/manifest"
)

const (
	FileName    = "gpm.lock"
	LockVersion = 1
)

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
