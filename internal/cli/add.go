package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/LynnColeArt/gpm/internal/deps"
	"github.com/LynnColeArt/gpm/internal/manifest"
	"github.com/LynnColeArt/gpm/internal/project"
)

func runAdd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(stderr)

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "add requires exactly one module target")
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
			fmt.Fprintln(stderr, "gpm add requires a gpm project; run gpm init first")
			return 1
		}
		fmt.Fprintf(stderr, "discover project root: %v\n", err)
		return 1
	}

	if !project.HasGoMod(root.Dir) {
		fmt.Fprintf(stderr, "gpm add requires go.mod in the project root: %s\n", root.Dir)
		return 1
	}

	file, err := manifest.Load(root.ManifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "load manifest: %v\n", err)
		return 1
	}

	target, err := deps.NormalizeTarget(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "resolve dependency target: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "adding %s to %s\n", target, root.GoModPath)
	if err := deps.Add(root.Dir, target, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "add dependency: %v\n", err)
		return 1
	}

	if err := writeLockfile(root.Dir, file, stdout); err != nil {
		fmt.Fprintf(stderr, "sync lockfile: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "added %s\n", target)
	return 0
}
