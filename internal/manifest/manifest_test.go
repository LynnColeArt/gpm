package manifest

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, FileName)

	original := Default("example.com/demo")
	original.Description = "demo project"

	if err := Save(path, original); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	if loaded.Name != original.Name {
		t.Fatalf("name mismatch: got %q want %q", loaded.Name, original.Name)
	}
	if loaded.Version != original.Version {
		t.Fatalf("version mismatch: got %q want %q", loaded.Version, original.Version)
	}
	if loaded.Scripts["build"] == "" {
		t.Fatal("expected build script to round-trip")
	}
}

func TestValidateRejectsEmptyName(t *testing.T) {
	t.Parallel()

	file := Default("demo")
	file.Name = " "

	if err := Validate(file); err == nil {
		t.Fatal("expected validation error for empty name")
	}
}
