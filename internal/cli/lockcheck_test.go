package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LynnColeArt/gpm/internal/lockfile"
	"github.com/LynnColeArt/gpm/internal/manifest"
	"github.com/LynnColeArt/gpm/internal/project"
)

func TestRequireCurrentLockfileAllowsProjectsWithoutGoMod(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	file := manifest.Default("example.com/demo")
	if err := manifest.Save(filepath.Join(root, manifest.FileName), file); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	if err := requireCurrentLockfile(projectRoot(root), file); err != nil {
		t.Fatalf("expected no lockfile requirement without go.mod, got %v", err)
	}
}

func TestRequireCurrentLockfileRejectsMissingLockfile(t *testing.T) {
	t.Parallel()

	root, file := writeCLIProject(t, "module example.com/demo\n\ngo 1.25.4\n")

	err := requireCurrentLockfile(projectRoot(root), file)
	if err == nil {
		t.Fatal("expected missing lockfile to fail")
	}
	if !strings.Contains(err.Error(), "run gpm install") {
		t.Fatalf("unexpected missing lockfile error: %v", err)
	}
}

func TestRequireCurrentLockfileRejectsDriftedLockfile(t *testing.T) {
	t.Parallel()

	root, file := writeCLIProject(t, "module example.com/demo\n\ngo 1.25.4\n\nrequire github.com/google/uuid v1.5.0\n")

	lock, err := lockfile.Build(root, file)
	if err != nil {
		t.Fatalf("build lockfile: %v", err)
	}
	if err := lockfile.Save(lockfile.Path(root), lock); err != nil {
		t.Fatalf("save lockfile: %v", err)
	}

	updated := "module example.com/demo\n\ngo 1.25.4\n\nrequire github.com/google/uuid v1.6.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(updated), 0o644); err != nil {
		t.Fatalf("rewrite go.mod: %v", err)
	}

	err = requireCurrentLockfile(projectRoot(root), file)
	if err == nil {
		t.Fatal("expected drifted lockfile to fail")
	}
	if !strings.Contains(err.Error(), "out of date") {
		t.Fatalf("unexpected drift error: %v", err)
	}
}

func TestRequireCurrentLockfileAllowsCurrentLockfile(t *testing.T) {
	t.Parallel()

	root, file := writeCLIProject(t, "module example.com/demo\n\ngo 1.25.4\n")

	lock, err := lockfile.Build(root, file)
	if err != nil {
		t.Fatalf("build lockfile: %v", err)
	}
	if err := lockfile.Save(lockfile.Path(root), lock); err != nil {
		t.Fatalf("save lockfile: %v", err)
	}

	if err := requireCurrentLockfile(projectRoot(root), file); err != nil {
		t.Fatalf("expected current lockfile to pass, got %v", err)
	}
}

func writeCLIProject(t *testing.T, goMod string) (string, *manifest.File) {
	t.Helper()

	root := t.TempDir()
	file := manifest.Default("example.com/demo")
	if err := manifest.Save(filepath.Join(root, manifest.FileName), file); err != nil {
		t.Fatalf("save manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	return root, file
}

func projectRoot(dir string) project.Root {
	return project.Root{
		Dir:          dir,
		ManifestPath: filepath.Join(dir, manifest.FileName),
		GoModPath:    filepath.Join(dir, "go.mod"),
	}
}
