package project

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LynnColeArt/gpm/internal/manifest"
)

var ErrNotFound = errors.New("gpm project not found")

type Root struct {
	Dir          string
	ManifestPath string
	GoModPath    string
}

func Find(start string) (Root, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return Root{}, fmt.Errorf("resolve start directory: %w", err)
	}

	info, err := os.Stat(current)
	if err != nil {
		return Root{}, fmt.Errorf("stat start directory: %w", err)
	}
	if !info.IsDir() {
		current = filepath.Dir(current)
	}

	for {
		manifestPath := filepath.Join(current, manifest.FileName)
		if _, err := os.Stat(manifestPath); err == nil {
			return Root{
				Dir:          current,
				ManifestPath: manifestPath,
				GoModPath:    filepath.Join(current, "go.mod"),
			}, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return Root{}, ErrNotFound
		}
		current = parent
	}
}

func ReadModulePath(dir string) (string, error) {
	path := filepath.Join(dir, "go.mod")
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan go.mod: %w", err)
	}

	return "", errors.New("module directive not found")
}

func HasGoMod(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}
