package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type File struct {
	DefaultRegistry string         `json:"defaultRegistry,omitempty"`
	Registries      map[string]any `json:"registries,omitempty"`
	Binaries        BinaryConfig   `json:"binaries,omitempty"`
}

type BinaryConfig struct {
	UserDir   string `json:"userDir,omitempty"`
	GlobalDir string `json:"globalDir,omitempty"`
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "gpm"), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*File, string, error) {
	path, err := Path()
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, path, nil
		}
		return nil, path, fmt.Errorf("read config: %w", err)
	}

	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, path, fmt.Errorf("parse config: %w", err)
	}
	if file.Registries == nil {
		file.Registries = map[string]any{}
	}
	return &file, path, nil
}
