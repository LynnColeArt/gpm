package apps

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/LynnColeArt/gpm/internal/config"
)

const (
	ScopeUser   = "user"
	ScopeGlobal = "global"
)

func NormalizeTarget(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New("install target is required")
	}

	if isLocalPath(trimmed) || strings.Contains(trimmed, "@") {
		return trimmed, nil
	}

	return trimmed + "@latest", nil
}

func ResolveBinDir(scope string, file *config.File) (string, error) {
	switch scope {
	case ScopeUser:
		if file != nil && strings.TrimSpace(file.Binaries.UserDir) != "" {
			return expandPath(file.Binaries.UserDir)
		}
		return defaultUserBinDir()
	case ScopeGlobal:
		if file != nil && strings.TrimSpace(file.Binaries.GlobalDir) != "" {
			return expandPath(file.Binaries.GlobalDir)
		}
		return defaultGlobalBinDir()
	default:
		return "", fmt.Errorf("unsupported scope %q", scope)
	}
}

func Install(target string, binDir string, stdout, stderr io.Writer) error {
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create bin directory: %w", err)
	}

	cmd := exec.Command("go", "install", target)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "GOBIN="+binDir)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install %s: %w", target, err)
	}
	return nil
}

func isLocalPath(value string) bool {
	return strings.HasPrefix(value, ".") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\`)
}

func expandPath(value string) (string, error) {
	if !strings.HasPrefix(value, "~") {
		return filepath.Clean(value), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	if value == "~" {
		return home, nil
	}

	trimmed := strings.TrimPrefix(value, "~/")
	return filepath.Join(home, trimmed), nil
}

func defaultUserBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Local", "gpm", "bin"), nil
	}

	return filepath.Join(home, ".local", "bin"), nil
}

func defaultGlobalBinDir() (string, error) {
	if runtime.GOOS == "windows" {
		if programFiles := os.Getenv("ProgramFiles"); strings.TrimSpace(programFiles) != "" {
			return filepath.Join(programFiles, "gpm", "bin"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, "AppData", "Local", "gpm", "global-bin"), nil
	}

	return "/usr/local/bin", nil
}
