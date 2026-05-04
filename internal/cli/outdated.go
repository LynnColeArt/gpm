package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/LynnColeArt/gpm/internal/deps"
	"github.com/LynnColeArt/gpm/internal/lockfile"
	"github.com/LynnColeArt/gpm/internal/manifest"
	"github.com/LynnColeArt/gpm/internal/project"
)

func runOutdated(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("outdated", flag.ContinueOnError)
	fs.SetOutput(stderr)

	includeIndirect := fs.Bool("all", false, "include indirect dependencies")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "outdated does not accept positional arguments in this phase")
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
			fmt.Fprintln(stderr, "gpm outdated requires a gpm project; run gpm init first")
			return 1
		}
		fmt.Fprintf(stderr, "discover project root: %v\n", err)
		return 1
	}

	if !project.HasGoMod(root.Dir) {
		fmt.Fprintf(stderr, "gpm outdated requires go.mod in the project root: %s\n", root.Dir)
		return 1
	}

	file, err := manifest.Load(root.ManifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "load manifest: %v\n", err)
		return 1
	}

	if warning, err := lockfileWarning(root, file); err != nil {
		fmt.Fprintf(stderr, "check project lock state: %v\n", err)
		return 1
	} else if warning != "" {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}

	modules, err := deps.ListOutdated(root.Dir, *includeIndirect)
	if err != nil {
		fmt.Fprintf(stderr, "inspect outdated dependencies: %v\n", err)
		return 1
	}

	scope := "direct"
	if *includeIndirect {
		scope = "all"
	}

	if len(modules) == 0 {
		fmt.Fprintf(stdout, "all checked %s dependencies are up to date\n", scope)
		return 0
	}

	tw := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "MODULE\tCURRENT\tLATEST\tNOTES")
	for _, module := range modules {
		latest := module.Latest
		if latest == "" {
			latest = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", module.Path, module.Version, latest, formatOutdatedNotes(module))
	}
	_ = tw.Flush()

	fmt.Fprintf(stdout, "%d actionable %s dependenc", len(modules), scope)
	if len(modules) == 1 {
		fmt.Fprintln(stdout, "y found")
	} else {
		fmt.Fprintln(stdout, "ies found")
	}

	return 0
}

func formatOutdatedNotes(module deps.OutdatedModule) string {
	notes := []string{}

	if module.Indirect {
		notes = append(notes, "indirect")
	}
	if module.Latest != "" {
		notes = append(notes, "update available")
	}
	if len(module.Retracted) > 0 {
		notes = append(notes, "retracted")
	}
	if module.Deprecated != "" {
		notes = append(notes, "deprecated")
	}
	if module.Replaced {
		notes = append(notes, "replaced")
	}
	if module.Problem != "" {
		notes = append(notes, "problem: "+module.Problem)
	}
	if len(notes) == 0 {
		return "-"
	}

	return strings.Join(notes, ", ")
}

func lockfileWarning(root project.Root, file *manifest.File) (string, error) {
	if !project.HasGoMod(root.Dir) {
		return "", nil
	}

	result, err := lockfile.Check(root.Dir, file)
	if err != nil {
		return "", fmt.Errorf("check gpm.lock: %w", err)
	}

	if !result.Exists {
		return fmt.Sprintf("missing %s; results reflect current go.mod state", result.LockPath), nil
	}
	if !result.Current() {
		return fmt.Sprintf("gpm.lock is out of date: %s; results reflect current go.mod state", result.Summary()), nil
	}

	return "", nil
}
