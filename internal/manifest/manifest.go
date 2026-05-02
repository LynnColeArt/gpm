package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const FileName = "gpm.json"

type File struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Private     bool              `json:"private"`
	Description string            `json:"description,omitempty"`
	Scripts     map[string]string `json:"scripts,omitempty"`
	Workspaces  []string          `json:"workspaces,omitempty"`
	Registries  map[string]any    `json:"registries,omitempty"`
	Tools       map[string]any    `json:"tools,omitempty"`
}

func Default(name string) *File {
	return &File{
		Name:        strings.TrimSpace(name),
		Version:     "0.1.0",
		Private:     true,
		Description: "Go-native package and workspace manager",
		Scripts: map[string]string{
			"build": "go build ./...",
			"test":  "go test ./...",
			"fmt":   "gofmt -w .",
			"tidy":  "go mod tidy",
		},
		Workspaces: []string{},
		Registries: map[string]any{},
		Tools:      map[string]any{},
	}
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if err := Validate(&file); err != nil {
		return nil, err
	}

	normalize(&file)
	return &file, nil
}

func Save(path string, file *File) error {
	if err := Validate(file); err != nil {
		return err
	}

	normalize(file)

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

func Validate(file *File) error {
	if file == nil {
		return errors.New("manifest is nil")
	}
	if strings.TrimSpace(file.Name) == "" {
		return errors.New("manifest name is required")
	}
	if strings.TrimSpace(file.Version) == "" {
		return errors.New("manifest version is required")
	}
	for name, command := range file.Scripts {
		if strings.TrimSpace(name) == "" {
			return errors.New("script name cannot be empty")
		}
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("script %q command cannot be empty", name)
		}
	}
	return nil
}

func normalize(file *File) {
	file.Name = strings.TrimSpace(file.Name)
	file.Version = strings.TrimSpace(file.Version)

	if file.Scripts == nil {
		file.Scripts = map[string]string{}
	}
	if file.Workspaces == nil {
		file.Workspaces = []string{}
	}
	if file.Registries == nil {
		file.Registries = map[string]any{}
	}
	if file.Tools == nil {
		file.Tools = map[string]any{}
	}
}
