package lockfile

import (
	"os"
	"path/filepath"
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
