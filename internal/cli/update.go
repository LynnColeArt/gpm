package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/LynnColeArt/gpm/internal/deps"
	"github.com/LynnColeArt/gpm/internal/manifest"
	"github.com/LynnColeArt/gpm/internal/project"
	"github.com/LynnColeArt/gpm/internal/tooling"
)

func runUpdate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)

	if err := fs.Parse(args); err != nil {
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "resolve working directory: %v\n", err)
		return 1
	}

	root, err := project.Find(cwd)
	if err != nil {
		if errors.Is(err, project.ErrNotFound) {
			fmt.Fprintln(stderr, "gpm update requires a gpm project; run gpm init first")
			return 1
		}
		fmt.Fprintf(stderr, "discover project root: %v\n", err)
		return 1
	}

	if !project.HasGoMod(root.Dir) {
		fmt.Fprintf(stderr, "gpm update requires go.mod in the project root: %s\n", root.Dir)
		return 1
	}

	file, err := manifest.Load(root.ManifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "load manifest: %v\n", err)
		return 1
	}

	targets := fs.Args()
	if len(targets) == 0 {
		fmt.Fprintf(stdout, "updating direct dependencies in %s\n", root.GoModPath)
	} else {
		fmt.Fprintf(stdout, "updating %s in %s\n", strings.Join(targets, ", "), root.GoModPath)
	}

	plan, err := deps.Update(root.Dir, targets, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "update dependencies: %v\n", err)
		return 1
	}

	if len(plan.UpgradeTargets) == 0 && len(plan.ExactTargets) == 0 {
		fmt.Fprintln(stdout, "no direct module dependencies to update")
	} else {
		fmt.Fprintf(stdout, "updated %d upgrade target(s) and %d pinned target(s)\n", len(plan.UpgradeTargets), len(plan.ExactTargets))
	}

	if err := tooling.InstallAll(root.Dir, file, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "install tools: %v\n", err)
		return 1
	}

	if err := writeLockfile(root.Dir, file, stdout); err != nil {
		fmt.Fprintf(stderr, "sync lockfile: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "dependencies updated")
	return 0
}
