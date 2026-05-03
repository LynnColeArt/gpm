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

func runRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	fs.SetOutput(stderr)

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "remove requires exactly one module target")
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
			fmt.Fprintln(stderr, "gpm remove requires a gpm project; run gpm init first")
			return 1
		}
		fmt.Fprintf(stderr, "discover project root: %v\n", err)
		return 1
	}

	if !project.HasGoMod(root.Dir) {
		fmt.Fprintf(stderr, "gpm remove requires go.mod in the project root: %s\n", root.Dir)
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

	module := deps.ModulePath(target)
	fmt.Fprintf(stdout, "removing %s from %s\n", module, root.GoModPath)
	result, err := deps.Remove(root.Dir, target, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "remove dependency: %v\n", err)
		return 1
	}

	if err := writeLockfile(root.Dir, file, stdout); err != nil {
		fmt.Fprintf(stderr, "sync lockfile: %v\n", err)
		return 1
	}

	if result.StillRequired {
		fmt.Fprintf(stderr, "warning: %s is still required by the current source tree and remains in go.mod\n", result.Module)
		return 0
	}

	fmt.Fprintf(stdout, "removed %s\n", result.Module)
	return 0
}
