package deps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/LynnColeArt/gpm/internal/gotool"
)

type OutdatedModule struct {
	Path       string
	Version    string
	Latest     string
	Indirect   bool
	Deprecated string
	Retracted  []string
	Problem    string
	Replaced   bool
}

func ListOutdated(dir string, includeIndirect bool) ([]OutdatedModule, error) {
	output, err := gotool.Output(dir, nil, "list", "-mod=mod", "-m", "-u", "-json", "all")
	if err != nil {
		return nil, fmt.Errorf("inspect module updates: %w", err)
	}

	modules, err := parseOutdatedModules(output, includeIndirect)
	if err != nil {
		return nil, err
	}

	return modules, nil
}

func parseOutdatedModules(data []byte, includeIndirect bool) ([]OutdatedModule, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	modules := []OutdatedModule{}

	for {
		var record struct {
			Path       string   `json:"Path"`
			Version    string   `json:"Version"`
			Main       bool     `json:"Main"`
			Indirect   bool     `json:"Indirect"`
			Deprecated string   `json:"Deprecated"`
			Retracted  []string `json:"Retracted"`
			Update     *struct {
				Version string `json:"Version"`
			} `json:"Update"`
			Replace *struct {
				Path string `json:"Path"`
				Dir  string `json:"Dir"`
			} `json:"Replace"`
			Error *struct {
				Err string `json:"Err"`
			} `json:"Error"`
		}

		if err := decoder.Decode(&record); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse outdated module list: %w", err)
		}

		if record.Main {
			continue
		}
		if record.Indirect && !includeIndirect {
			continue
		}

		latest := ""
		if record.Update != nil {
			latest = record.Update.Version
		}

		problem := ""
		if record.Error != nil {
			problem = record.Error.Err
		}

		if latest == "" && record.Deprecated == "" && len(record.Retracted) == 0 && problem == "" {
			continue
		}

		modules = append(modules, OutdatedModule{
			Path:       record.Path,
			Version:    record.Version,
			Latest:     latest,
			Indirect:   record.Indirect,
			Deprecated: record.Deprecated,
			Retracted:  record.Retracted,
			Problem:    problem,
			Replaced:   record.Replace != nil,
		})
	}

	return modules, nil
}
