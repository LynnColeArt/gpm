package tooling

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/LynnColeArt/gpm/internal/gotool"
	"github.com/LynnColeArt/gpm/internal/manifest"
)

const (
	DirName = ".gpm"
)

func BinDir(projectRoot string) string {
	return filepath.Join(projectRoot, DirName, "tools", "bin")
}

func ToolTarget(tool manifest.Tool) string {
	return tool.Module + "@" + tool.Version
}

func BinaryName(tool manifest.Tool) string {
	if strings.TrimSpace(tool.Binary) != "" {
		return strings.TrimSpace(tool.Binary)
	}

	module := strings.TrimSpace(tool.Module)
	module = strings.TrimSuffix(module, "/")
	if idx := strings.LastIndex(module, "/"); idx >= 0 {
		return module[idx+1:]
	}
	return module
}

func BinaryPath(projectRoot string, tool manifest.Tool) string {
	path := filepath.Join(BinDir(projectRoot), BinaryName(tool))
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	return path
}

func InstallAll(projectRoot string, file *manifest.File, stdout, stderr io.Writer) error {
	if file == nil {
		return errors.New("manifest is nil")
	}

	if len(file.Tools) == 0 {
		return nil
	}

	binDir := BinDir(projectRoot)
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create tool bin directory: %w", err)
	}

	names := make([]string, 0, len(file.Tools))
	for name := range file.Tools {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		tool := file.Tools[name]
		target := ToolTarget(tool)
		fmt.Fprintf(stdout, "installing tool %s from %s\n", name, target)
		if err := gotool.Run(projectRoot, stdout, stderr, []string{"GOBIN=" + binDir}, "install", target); err != nil {
			return fmt.Errorf("install tool %q: %w", name, err)
		}
	}

	return nil
}

func Exec(projectRoot string, file *manifest.File, name string, args []string, stdout, stderr io.Writer) error {
	if file == nil {
		return errors.New("manifest is nil")
	}

	if _, ok := file.Tools[name]; !ok {
		return fmt.Errorf("tool %q not found in gpm.json", name)
	}

	tool := file.Tools[name]
	binaryPath := BinaryPath(projectRoot, tool)
	if _, err := os.Stat(binaryPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("tool %q is not installed at %s; run gpm install", name, binaryPath)
		}
		return fmt.Errorf("stat tool binary: %w", err)
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = projectRoot
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = AppendBinToEnv(os.Environ(), projectRoot)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run tool %q: %w", name, err)
	}

	return nil
}

func AppendBinToEnv(env []string, projectRoot string) []string {
	binDir := BinDir(projectRoot)
	result := make([]string, 0, len(env)+2)
	foundPath := false
	foundBin := false

	for _, entry := range env {
		if len(entry) >= 5 && entry[:5] == "PATH=" {
			foundPath = true
			value := entry[5:]
			result = append(result, "PATH="+binDir+string(os.PathListSeparator)+value)
			continue
		}
		if len(entry) >= 14 && entry[:14] == "GPM_TOOLS_BIN=" {
			foundBin = true
			result = append(result, "GPM_TOOLS_BIN="+binDir)
			continue
		}
		result = append(result, entry)
	}

	if !foundPath {
		result = append(result, "PATH="+binDir)
	}
	if !foundBin {
		result = append(result, "GPM_TOOLS_BIN="+binDir)
	}

	return result
}
