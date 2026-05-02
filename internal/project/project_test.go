package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindWalksUpward(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "gpm.json"), []byte(`{"name":"demo","version":"0.1.0","private":true}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	found, err := Find(nested)
	if err != nil {
		t.Fatalf("find root: %v", err)
	}

	if found.Dir != root {
		t.Fatalf("root mismatch: got %q want %q", found.Dir, root)
	}
}

func TestReadModulePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	content := []byte("module example.com/demo\n\ngo 1.25.4\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), content, 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	modulePath, err := ReadModulePath(root)
	if err != nil {
		t.Fatalf("read module path: %v", err)
	}

	if modulePath != "example.com/demo" {
		t.Fatalf("module path mismatch: got %q", modulePath)
	}
}
