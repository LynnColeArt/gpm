package deps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LynnColeArt/gpm/internal/gotool"
)

type Requirement struct {
	Path     string
	Version  string
	Indirect bool
}

type UpdatePlan struct {
	UpgradeTargets []string
	ExactTargets   []string
}

type RemoveResult struct {
	Module        string
	StillRequired bool
}

var errFoundGoSource = errors.New("found go source")

func NormalizeTarget(target string) (string, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return "", errors.New("dependency target is required")
	}
	return trimmed, nil
}

func ModulePath(target string) string {
	normalized := strings.TrimSpace(target)
	if at := strings.Index(normalized, "@"); at >= 0 {
		return normalized[:at]
	}
	return normalized
}

func Add(dir, target string, stdout, stderr io.Writer) error {
	normalized, err := NormalizeTarget(target)
	if err != nil {
		return err
	}

	if err := gotool.Run(dir, stdout, stderr, nil, "get", normalized); err != nil {
		return fmt.Errorf("add dependency: %w", err)
	}

	return nil
}

func Update(dir string, targets []string, stdout, stderr io.Writer) (UpdatePlan, error) {
	requirements, err := Requirements(dir)
	if err != nil {
		return UpdatePlan{}, err
	}

	plan, err := BuildUpdatePlan(requirements, targets)
	if err != nil {
		return UpdatePlan{}, err
	}

	if len(plan.UpgradeTargets) > 0 {
		args := append([]string{"get", "-u"}, plan.UpgradeTargets...)
		if err := gotool.Run(dir, stdout, stderr, nil, args...); err != nil {
			return plan, fmt.Errorf("upgrade dependencies: %w", err)
		}
	}

	if len(plan.ExactTargets) > 0 {
		args := append([]string{"get"}, plan.ExactTargets...)
		if err := gotool.Run(dir, stdout, stderr, nil, args...); err != nil {
			return plan, fmt.Errorf("set dependency versions: %w", err)
		}
	}

	return plan, nil
}

func Install(dir string, stdout, stderr io.Writer) error {
	if err := gotool.Run(dir, stdout, stderr, nil, "mod", "download"); err != nil {
		return fmt.Errorf("install dependencies: %w", err)
	}

	return nil
}

func Remove(dir, target string, stdout, stderr io.Writer) (RemoveResult, error) {
	normalized, err := NormalizeTarget(target)
	if err != nil {
		return RemoveResult{}, err
	}

	module := ModulePath(normalized)
	if module == "" {
		return RemoveResult{}, errors.New("dependency module path is required")
	}

	if err := gotool.Run(dir, stdout, stderr, nil, "get", module+"@none"); err != nil {
		return RemoveResult{}, fmt.Errorf("remove dependency: %w", err)
	}

	hasSource, err := hasGoSource(dir)
	if err != nil {
		return RemoveResult{}, fmt.Errorf("scan project source: %w", err)
	}

	if hasSource {
		if err := gotool.Run(dir, stdout, stderr, nil, "mod", "tidy"); err != nil {
			return RemoveResult{}, fmt.Errorf("tidy module: %w", err)
		}
	}

	stillRequired, err := RequirementExists(dir, module)
	if err != nil {
		return RemoveResult{}, err
	}

	return RemoveResult{
		Module:        module,
		StillRequired: stillRequired,
	}, nil
}

func Requirements(dir string) ([]Requirement, error) {
	output, err := gotool.Output(dir, nil, "mod", "edit", "-json")
	if err != nil {
		return nil, fmt.Errorf("inspect go.mod: %w", err)
	}

	var modFile struct {
		Require []struct {
			Path     string `json:"Path"`
			Version  string `json:"Version"`
			Indirect bool   `json:"Indirect"`
		} `json:"Require"`
	}

	if err := json.Unmarshal(output, &modFile); err != nil {
		return nil, fmt.Errorf("parse go.mod json: %w", err)
	}

	requirements := make([]Requirement, 0, len(modFile.Require))
	for _, requirement := range modFile.Require {
		requirements = append(requirements, Requirement{
			Path:     requirement.Path,
			Version:  requirement.Version,
			Indirect: requirement.Indirect,
		})
	}

	return requirements, nil
}

func BuildUpdatePlan(requirements []Requirement, targets []string) (UpdatePlan, error) {
	plan := UpdatePlan{}

	if len(targets) == 0 {
		seen := map[string]struct{}{}
		for _, requirement := range requirements {
			if requirement.Indirect {
				continue
			}

			module := strings.TrimSpace(requirement.Path)
			if module == "" {
				continue
			}
			if _, ok := seen[module]; ok {
				continue
			}
			seen[module] = struct{}{}
			plan.UpgradeTargets = append(plan.UpgradeTargets, module)
		}
		sort.Strings(plan.UpgradeTargets)
		return plan, nil
	}

	upgradeSeen := map[string]struct{}{}
	exactSeen := map[string]struct{}{}

	for _, rawTarget := range targets {
		normalized, err := NormalizeTarget(rawTarget)
		if err != nil {
			return UpdatePlan{}, err
		}

		if strings.Contains(normalized, "@") {
			if _, ok := exactSeen[normalized]; ok {
				continue
			}
			exactSeen[normalized] = struct{}{}
			plan.ExactTargets = append(plan.ExactTargets, normalized)
			continue
		}

		module := ModulePath(normalized)
		if _, ok := upgradeSeen[module]; ok {
			continue
		}
		upgradeSeen[module] = struct{}{}
		plan.UpgradeTargets = append(plan.UpgradeTargets, module)
	}

	sort.Strings(plan.UpgradeTargets)
	sort.Strings(plan.ExactTargets)
	return plan, nil
}

func RequirementExists(dir, module string) (bool, error) {
	requirements, err := Requirements(dir)
	if err != nil {
		return false, err
	}

	for _, requirement := range requirements {
		if requirement.Path == module {
			return true, nil
		}
	}

	return false, nil
}

func hasGoSource(dir string) (bool, error) {
	found := false

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(entry.Name(), ".go") {
			found = true
			return errFoundGoSource
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, errFoundGoSource) {
			return true, nil
		}
		return false, err
	}

	return found, nil
}
