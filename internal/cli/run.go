package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/LynnColeArt/gpm/internal/manifest"
	"github.com/LynnColeArt/gpm/internal/project"
	"github.com/LynnColeArt/gpm/internal/tooling"
)

func runScript(args []string, stdout, stderr io.Writer) int {
	scriptName, passthrough, err := parseRunArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "parse run arguments: %v\n", err)
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

	if err := requireCurrentLockfile(root, file); err != nil {
		fmt.Fprintf(stderr, "check project lock state: %v\n", err)
		return 1
	}

	commandText, ok := file.Scripts[scriptName]
	if !ok {
		fmt.Fprintf(stderr, "script %q not found in %s\n", scriptName, root.ManifestPath)
		return 1
	}

	shell, shellFlag := platformShell()
	commandText = appendArgs(commandText, passthrough)

	cmd := exec.Command(shell, shellFlag, commandText)
	cmd.Dir = root.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = tooling.AppendBinToEnv(os.Environ(), root.Dir)
	cmd.Env = append(cmd.Env,
		"GPM_PROJECT_ROOT="+root.Dir,
		"GPM_MANIFEST_PATH="+root.ManifestPath,
		"GPM_SCRIPT_NAME="+scriptName,
	)

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "run script %q: %v\n", scriptName, err)
		return 1
	}

	return 0
}

func parseRunArgs(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, errors.New("missing script name")
	}
	if len(args) == 1 {
		return args[0], nil, nil
	}
	if args[1] != "--" {
		return "", nil, errors.New("extra arguments must be separated by --")
	}
	return args[0], args[2:], nil
}

func platformShell() (string, string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", "/C"
	}
	return "/bin/sh", "-c"
}

func appendArgs(command string, args []string) string {
	if len(args) == 0 {
		return command
	}

	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return command + " " + strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	if runtime.GOOS == "windows" {
		if value == "" {
			return "\"\""
		}
		replacer := strings.NewReplacer(`"`, `\"`)
		return `"` + replacer.Replace(value) + `"`
	}

	if value == "" {
		return "''"
	}

	if !strings.ContainsAny(value, " \t\n'\"\\$`!&*?[]{}()<>|;") {
		return value
	}

	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
