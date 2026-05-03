package lockfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LynnColeArt/gpm/internal/manifest"
)

func TestBuildIncludesDependenciesAndTools(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goMod := "module example.com/demo\n\ngo 1.25.4\n\nrequire github.com/google/uuid v1.6.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	file := manifest.Default("example.com/demo")
	file.Tools["strgen"] = manifest.Tool{
		Module:  "golang.org/x/tools/cmd/stringer",
		Version: "v0.38.0",
		Binary:  "stringer",
	}

	lock, err := Build(root, file)
	if err != nil {
		t.Fatalf("build lockfile: %v", err)
	}

	if lock.ModulePath != "example.com/demo" {
		t.Fatalf("unexpected module path: %q", lock.ModulePath)
	}
	if len(lock.Dependencies) != 1 || lock.Dependencies[0].Path != "github.com/google/uuid" {
		t.Fatalf("unexpected dependencies: %#v", lock.Dependencies)
	}
	if lock.Tools["strgen"].Target != "golang.org/x/tools/cmd/stringer@v0.38.0" {
		t.Fatalf("unexpected tool target: %#v", lock.Tools["strgen"])
	}
}

func TestLoadMissingReturnsErrNotFound(t *testing.T) {
	t.Parallel()

	_, err := Load(filepath.Join(t.TempDir(), FileName))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCheckReportsCurrentLockfile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goMod := "module example.com/demo\n\ngo 1.25.4\n\nrequire github.com/google/uuid v1.6.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	file := manifest.Default("example.com/demo")
	file.Tools["strgen"] = manifest.Tool{
		Module:  "golang.org/x/tools/cmd/stringer",
		Version: "v0.38.0",
		Binary:  "stringer",
	}

	lock, err := Build(root, file)
	if err != nil {
		t.Fatalf("build lockfile: %v", err)
	}
	if err := Save(Path(root), lock); err != nil {
		t.Fatalf("save lockfile: %v", err)
	}

	result, err := Check(root, file)
	if err != nil {
		t.Fatalf("check lockfile: %v", err)
	}
	if !result.Current() {
		t.Fatalf("expected current lockfile, got issues: %#v", result.Issues)
	}
}

func TestCheckReportsMissingLockfile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goMod := "module example.com/demo\n\ngo 1.25.4\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	result, err := Check(root, manifest.Default("example.com/demo"))
	if err != nil {
		t.Fatalf("check lockfile: %v", err)
	}
	if result.Exists {
		t.Fatal("expected lockfile to be missing")
	}
	if result.Current() {
		t.Fatal("missing lockfile should not be current")
	}
}

func TestCheckReportsDependencyDrift(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goMod := "module example.com/demo\n\ngo 1.25.4\n\nrequire github.com/google/uuid v1.5.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	file := manifest.Default("example.com/demo")
	lock, err := Build(root, file)
	if err != nil {
		t.Fatalf("build lockfile: %v", err)
	}
	if err := Save(Path(root), lock); err != nil {
		t.Fatalf("save lockfile: %v", err)
	}

	updatedGoMod := "module example.com/demo\n\ngo 1.25.4\n\nrequire github.com/google/uuid v1.6.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(updatedGoMod), 0o644); err != nil {
		t.Fatalf("rewrite go.mod: %v", err)
	}

	result, err := Check(root, file)
	if err != nil {
		t.Fatalf("check lockfile: %v", err)
	}
	if result.Current() {
		t.Fatal("expected dependency drift to be detected")
	}
	if !strings.Contains(result.Summary(), "dependency mismatch") {
		t.Fatalf("unexpected dependency drift summary: %q", result.Summary())
	}
}

func TestCheckReportsToolDrift(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goMod := "module example.com/demo\n\ngo 1.25.4\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	file := manifest.Default("example.com/demo")
	file.Tools["strgen"] = manifest.Tool{
		Module:  "golang.org/x/tools/cmd/stringer",
		Version: "v0.38.0",
	}

	lock, err := Build(root, file)
	if err != nil {
		t.Fatalf("build lockfile: %v", err)
	}
	if err := Save(Path(root), lock); err != nil {
		t.Fatalf("save lockfile: %v", err)
	}

	file.Tools["strgen"] = manifest.Tool{
		Module:  "golang.org/x/tools/cmd/stringer",
		Version: "v0.39.0",
	}

	result, err := Check(root, file)
	if err != nil {
		t.Fatalf("check lockfile: %v", err)
	}
	if result.Current() {
		t.Fatal("expected tool drift to be detected")
	}
	if !strings.Contains(result.Summary(), "tool \"strgen\" mismatch") {
		t.Fatalf("unexpected tool drift summary: %q", result.Summary())
	}
}
