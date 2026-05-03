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
	"github.com/LynnColeArt/gpm/internal/tooling"
)

func runInstall(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(stderr)

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "install does not accept positional arguments in this phase")
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
			fmt.Fprintln(stderr, "gpm install requires a gpm project; run gpm init first")
			return 1
		}
		fmt.Fprintf(stderr, "discover project root: %v\n", err)
		return 1
	}

	if !project.HasGoMod(root.Dir) {
		fmt.Fprintf(stderr, "gpm install requires go.mod in the project root: %s\n", root.Dir)
		return 1
	}

	file, err := manifest.Load(root.ManifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "load manifest: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "installing dependencies for %s\n", root.GoModPath)
	if err := deps.Install(root.Dir, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "install dependencies: %v\n", err)
		return 1
	}

	if err := tooling.InstallAll(root.Dir, file, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "install tools: %v\n", err)
		return 1
	}

	if err := writeLockfile(root.Dir, file, stdout); err != nil {
		fmt.Fprintf(stderr, "sync lockfile: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "dependencies installed")
	return 0
}
