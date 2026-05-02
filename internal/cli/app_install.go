package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/LynnColeArt/gpm/internal/apps"
	"github.com/LynnColeArt/gpm/internal/config"
)

func runApp(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "missing app subcommand")
		return 1
	}

	switch args[0] {
	case "install":
		return runAppInstall(args[1:], stdout, stderr)
	case "list":
		return runAppList(args[1:], stdout, stderr)
	case "uninstall":
		return runAppUninstall(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown app subcommand %q\n", args[0])
		return 1
	}
}

func runAppInstall(args []string, stdout, stderr io.Writer) int {
	targetArg, scope, binDir, err := parseAppInstallArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "parse app install arguments: %v\n", err)
		return 1
	}

	target, err := apps.NormalizeTarget(targetArg)
	if err != nil {
		fmt.Fprintf(stderr, "resolve install target: %v\n", err)
		return 1
	}

	cfg, _, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "load user config: %v\n", err)
		return 1
	}

	if binDir == "" {
		binDir, err = apps.ResolveBinDir(scope, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "resolve bin directory: %v\n", err)
			return 1
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "resolve working directory: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "installing %s to %s\n", target, binDir)
	if err := apps.Install(target, binDir, stdout, stderr); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "install app: %v\n", err)
		return 1
	}

	record, err := apps.BuildInstallRecord(targetArg, target, scope, binDir, cwd, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "record install: %v\n", err)
		return 1
	}

	state, statePath, err := apps.LoadState()
	if err != nil {
		fmt.Fprintf(stderr, "load app state: %v\n", err)
		return 1
	}
	apps.UpsertInstall(state, record)
	if err := apps.SaveState(statePath, state); err != nil {
		fmt.Fprintf(stderr, "save app state: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "installed %s to %s\n", target, binDir)
	return 0
}

func runAppList(args []string, stdout, stderr io.Writer) int {
	scope := "all"
	switch len(args) {
	case 0:
	case 1:
		if !strings.HasPrefix(args[0], "--scope=") {
			fmt.Fprintf(stderr, "parse app list arguments: unknown argument %q\n", args[0])
			return 1
		}
		scope = strings.TrimPrefix(args[0], "--scope=")
	default:
		if len(args) != 2 || args[0] != "--scope" {
			fmt.Fprintf(stderr, "parse app list arguments: unknown arguments %q\n", strings.Join(args, " "))
			return 1
		}
		scope = args[1]
	}

	if scope != "all" && scope != apps.ScopeUser && scope != apps.ScopeGlobal {
		fmt.Fprintf(stderr, "parse app list arguments: unsupported scope %q\n", scope)
		return 1
	}

	state, _, err := apps.LoadState()
	if err != nil {
		fmt.Fprintf(stderr, "load app state: %v\n", err)
		return 1
	}

	records := apps.List(state, scope)
	if len(records) == 0 {
		fmt.Fprintln(stdout, "no installed apps recorded")
		return 0
	}

	for _, record := range records {
		status := "missing"
		if apps.BinaryExists(record) {
			status = "ok"
		}
		fmt.Fprintf(stdout, "[%s] [%s] %s -> %s (from %s)\n", status, record.Scope, record.BinaryName, record.BinaryPath, record.RequestedTarget)
	}
	return 0
}

func runAppUninstall(args []string, stdout, stderr io.Writer) int {
	query, scope, err := parseAppUninstallArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "parse app uninstall arguments: %v\n", err)
		return 1
	}

	state, statePath, err := apps.LoadState()
	if err != nil {
		fmt.Fprintf(stderr, "load app state: %v\n", err)
		return 1
	}

	record, ok := apps.RemoveInstall(state, scope, query)
	if !ok {
		fmt.Fprintf(stderr, "no installed app matched %q in scope %q\n", query, scope)
		return 1
	}

	if err := os.Remove(record.BinaryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stderr, "remove binary %s: %v\n", record.BinaryPath, err)
		return 1
	}

	if err := apps.SaveState(statePath, state); err != nil {
		fmt.Fprintf(stderr, "save app state: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "uninstalled %s from %s\n", record.BinaryName, record.BinaryPath)
	return 0
}

func parseAppInstallArgs(args []string) (string, string, string, error) {
	scope := apps.ScopeUser
	binDir := ""
	target := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--scope":
			i++
			if i >= len(args) {
				return "", "", "", errors.New("missing value for --scope")
			}
			scope = args[i]
		case strings.HasPrefix(arg, "--scope="):
			scope = strings.TrimPrefix(arg, "--scope=")
		case arg == "--bin-dir":
			i++
			if i >= len(args) {
				return "", "", "", errors.New("missing value for --bin-dir")
			}
			binDir = args[i]
		case strings.HasPrefix(arg, "--bin-dir="):
			binDir = strings.TrimPrefix(arg, "--bin-dir=")
		case strings.HasPrefix(arg, "-"):
			return "", "", "", fmt.Errorf("unknown flag %q", arg)
		default:
			if target != "" {
				return "", "", "", errors.New("app install requires exactly one package or module target")
			}
			target = arg
		}
	}

	if strings.TrimSpace(target) == "" {
		return "", "", "", errors.New("app install requires exactly one package or module target")
	}

	return target, scope, binDir, nil
}

func parseAppUninstallArgs(args []string) (string, string, error) {
	scope := apps.ScopeUser
	query := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--scope":
			i++
			if i >= len(args) {
				return "", "", errors.New("missing value for --scope")
			}
			scope = args[i]
		case strings.HasPrefix(arg, "--scope="):
			scope = strings.TrimPrefix(arg, "--scope=")
		case strings.HasPrefix(arg, "-"):
			return "", "", fmt.Errorf("unknown flag %q", arg)
		default:
			if query != "" {
				return "", "", errors.New("app uninstall requires exactly one name or target")
			}
			query = arg
		}
	}

	if strings.TrimSpace(query) == "" {
		return "", "", errors.New("app uninstall requires exactly one name or target")
	}

	return query, scope, nil
}
