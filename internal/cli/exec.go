package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/LynnColeArt/gpm/internal/manifest"
	"github.com/LynnColeArt/gpm/internal/project"
	"github.com/LynnColeArt/gpm/internal/tooling"
)

func runExec(args []string, stdout, stderr io.Writer) int {
	toolName, passthrough, err := parseRunArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "parse exec arguments: %v\n", err)
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
			fmt.Fprintf(stderr, "no %s found from %s upward\n", manifest.FileName, cwd)
			return 1
		}
		fmt.Fprintf(stderr, "discover project root: %v\n", err)
		return 1
	}

	file, err := manifest.Load(root.ManifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "load manifest: %v\n", err)
		return 1
	}

	if err := tooling.Exec(root.Dir, file, toolName, passthrough, stdout, stderr); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "exec tool: %v\n", err)
		return 1
	}

	return 0
}
