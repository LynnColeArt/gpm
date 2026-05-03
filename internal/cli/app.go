package cli

import (
	"fmt"
	"io"
)

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "add":
		return runAdd(args[1:], stdout, stderr)
	case "remove":
		return runRemove(args[1:], stdout, stderr)
	case "install":
		return runInstall(args[1:], stdout, stderr)
	case "update":
		return runUpdate(args[1:], stdout, stderr)
	case "exec":
		return runExec(args[1:], stdout, stderr)
	case "run":
		return runScript(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "app":
		return runApp(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "gpm is an npm-style package and workspace manager for Go.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gpm init [--force] [--name <name>] [--private=<true|false>]")
	fmt.Fprintln(w, "  gpm add <module>[@version]")
	fmt.Fprintln(w, "  gpm remove <module>[@version]")
	fmt.Fprintln(w, "  gpm install")
	fmt.Fprintln(w, "  gpm update [<module>[@version] ...]")
	fmt.Fprintln(w, "  gpm exec <tool-name> [-- <arg>...]")
	fmt.Fprintln(w, "  gpm run <script-name> [-- <arg>...]")
	fmt.Fprintln(w, "  gpm doctor")
	fmt.Fprintln(w, "  gpm app install <package-or-module>[@<version>] [--scope user|global] [--bin-dir <path>]")
	fmt.Fprintln(w, "  gpm app list [--scope user|global|all]")
	fmt.Fprintln(w, "  gpm app uninstall <name-or-target> [--scope user|global]")
}
