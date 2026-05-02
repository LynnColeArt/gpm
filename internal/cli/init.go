package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/LynnColeArt/gpm/internal/manifest"
	"github.com/LynnColeArt/gpm/internal/project"
)

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var force bool
	var name string
	var privateOverride *bool

	fs.BoolVar(&force, "force", false, "overwrite an existing gpm.json")
	fs.StringVar(&name, "name", "", "manifest name override")
	fs.Func("private", "set the private flag", func(value string) error {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		privateOverride = &parsed
		return nil
	})

	if err := fs.Parse(args); err != nil {
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "resolve working directory: %v\n", err)
		return 1
	}

	manifestPath := filepath.Join(cwd, manifest.FileName)
	if !force {
		if _, err := os.Stat(manifestPath); err == nil {
			fmt.Fprintf(stderr, "%s already exists; use --force to overwrite it\n", manifest.FileName)
			return 1
		}
	}

	resolvedName := name
	if resolvedName == "" {
		if modulePath, err := project.ReadModulePath(cwd); err == nil && modulePath != "" {
			resolvedName = modulePath
		} else {
			resolvedName = filepath.Base(cwd)
		}
	}

	file := manifest.Default(resolvedName)
	if privateOverride != nil {
		file.Private = *privateOverride
	}

	if err := manifest.Save(manifestPath, file); err != nil {
		fmt.Fprintf(stderr, "write manifest: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "created %s\n", manifestPath)
	if !project.HasGoMod(cwd) {
		fmt.Fprintln(stdout, "warning: no go.mod found; gpm init did not create one")
	}
	return 0
}
