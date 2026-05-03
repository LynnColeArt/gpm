package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/LynnColeArt/gpm/internal/apps"
	"github.com/LynnColeArt/gpm/internal/config"
	"github.com/LynnColeArt/gpm/internal/lockfile"
	"github.com/LynnColeArt/gpm/internal/manifest"
	"github.com/LynnColeArt/gpm/internal/project"
)

type doctorStatus string

const (
	doctorOK   doctorStatus = "ok"
	doctorWarn doctorStatus = "warn"
	doctorFail doctorStatus = "fail"
)

type doctorCheck struct {
	Name    string
	Status  doctorStatus
	Details string
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)

	if err := fs.Parse(args); err != nil {
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "resolve working directory: %v\n", err)
		return 1
	}

	checks := make([]doctorCheck, 0, 6)

	if path, err := exec.LookPath("go"); err != nil {
		checks = append(checks, doctorCheck{Name: "go binary", Status: doctorFail, Details: err.Error()})
	} else {
		checks = append(checks, doctorCheck{Name: "go binary", Status: doctorOK, Details: path})
	}

	root, rootErr := project.Find(cwd)
	if rootErr != nil {
		if errors.Is(rootErr, project.ErrNotFound) {
			checks = append(checks, doctorCheck{Name: "gpm manifest", Status: doctorFail, Details: "no gpm.json found from current directory upward"})
		} else {
			checks = append(checks, doctorCheck{Name: "gpm manifest", Status: doctorFail, Details: rootErr.Error()})
		}
	} else {
		checks = append(checks, doctorCheck{Name: "gpm manifest", Status: doctorOK, Details: root.ManifestPath})

		var file *manifest.File
		if loaded, err := manifest.Load(root.ManifestPath); err != nil {
			checks = append(checks, doctorCheck{Name: "manifest validation", Status: doctorFail, Details: err.Error()})
		} else {
			file = loaded
			checks = append(checks, doctorCheck{Name: "manifest validation", Status: doctorOK, Details: fmt.Sprintf("name=%s version=%s scripts=%d", file.Name, file.Version, len(file.Scripts))})
		}

		if project.HasGoMod(root.Dir) {
			checks = append(checks, doctorCheck{Name: "go.mod", Status: doctorOK, Details: root.GoModPath})

			if file != nil {
				result, err := lockfile.Check(root.Dir, file)
				if err != nil {
					checks = append(checks, doctorCheck{Name: "gpm.lock", Status: doctorFail, Details: err.Error()})
				} else if !result.Exists {
					checks = append(checks, doctorCheck{Name: "gpm.lock", Status: doctorWarn, Details: fmt.Sprintf("missing %s; run gpm install", result.LockPath)})
				} else if !result.Current() {
					checks = append(checks, doctorCheck{Name: "gpm.lock", Status: doctorFail, Details: result.Summary()})
				} else {
					checks = append(checks, doctorCheck{Name: "gpm.lock", Status: doctorOK, Details: result.LockPath})
				}
			}
		} else {
			checks = append(checks, doctorCheck{Name: "go.mod", Status: doctorWarn, Details: "no go.mod found in project root"})
		}
	}

	if rootErr != nil && !errors.Is(rootErr, project.ErrNotFound) {
		checks = append(checks, doctorCheck{Name: "project discovery", Status: doctorFail, Details: rootErr.Error()})
	}

	if cfg, path, err := config.Load(); err != nil {
		checks = append(checks, doctorCheck{Name: "user config", Status: doctorFail, Details: err.Error()})
	} else if cfg == nil {
		checks = append(checks, doctorCheck{Name: "user config", Status: doctorWarn, Details: fmt.Sprintf("optional config not found at %s", path)})
		if userBinDir, err := apps.ResolveBinDir(apps.ScopeUser, nil); err != nil {
			checks = append(checks, doctorCheck{Name: "user bin dir", Status: doctorFail, Details: err.Error()})
		} else {
			checks = append(checks, doctorCheck{Name: "user bin dir", Status: doctorOK, Details: userBinDir})
			if pathContains(userBinDir) {
				checks = append(checks, doctorCheck{Name: "user bin dir on PATH", Status: doctorOK, Details: userBinDir})
			} else {
				checks = append(checks, doctorCheck{Name: "user bin dir on PATH", Status: doctorWarn, Details: fmt.Sprintf("%s is not on PATH", userBinDir)})
			}
		}
	} else {
		checks = append(checks, doctorCheck{Name: "user config", Status: doctorOK, Details: fmt.Sprintf("%s registries=%d", path, len(cfg.Registries))})
		if userBinDir, err := apps.ResolveBinDir(apps.ScopeUser, cfg); err != nil {
			checks = append(checks, doctorCheck{Name: "user bin dir", Status: doctorFail, Details: err.Error()})
		} else {
			checks = append(checks, doctorCheck{Name: "user bin dir", Status: doctorOK, Details: userBinDir})
			if pathContains(userBinDir) {
				checks = append(checks, doctorCheck{Name: "user bin dir on PATH", Status: doctorOK, Details: userBinDir})
			} else {
				checks = append(checks, doctorCheck{Name: "user bin dir on PATH", Status: doctorWarn, Details: fmt.Sprintf("%s is not on PATH", userBinDir)})
			}
		}
	}

	if state, statePath, err := apps.LoadState(); err != nil {
		checks = append(checks, doctorCheck{Name: "app state", Status: doctorFail, Details: err.Error()})
	} else if len(state.Installs) == 0 {
		checks = append(checks, doctorCheck{Name: "app state", Status: doctorWarn, Details: fmt.Sprintf("optional app state empty at %s", statePath)})
	} else {
		checks = append(checks, doctorCheck{Name: "app state", Status: doctorOK, Details: fmt.Sprintf("%s installs=%d", statePath, len(state.Installs))})
	}

	exitCode := 0
	for _, check := range checks {
		fmt.Fprintf(stdout, "[%s] %s: %s\n", check.Status, check.Name, check.Details)
		if check.Status == doctorFail {
			exitCode = 1
		}
	}

	return exitCode
}

func pathContains(target string) bool {
	for _, entry := range filepathList(os.Getenv("PATH")) {
		if entry == target {
			return true
		}
	}
	return false
}

func filepathList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.Split(value, string(os.PathListSeparator))
}
