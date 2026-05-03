package tooling

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/LynnColeArt/gpm/internal/manifest"
)

func TestToolTargetIncludesVersion(t *testing.T) {
	t.Parallel()

	target := ToolTarget(manifest.Tool{
		Module:  "github.com/golangci/golangci-lint/cmd/golangci-lint",
		Version: "v1.64.8",
	})

	if target != "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8" {
		t.Fatalf("unexpected tool target: %q", target)
	}
}

func TestBinaryNameDefaultsToModuleBasename(t *testing.T) {
	t.Parallel()

	name := BinaryName(manifest.Tool{
		Module:  "golang.org/x/tools/cmd/stringer",
		Version: "v0.1.0",
	})

	if name != "stringer" {
		t.Fatalf("unexpected binary name: %q", name)
	}
}

func TestBinaryNameUsesExplicitOverride(t *testing.T) {
	t.Parallel()

	name := BinaryName(manifest.Tool{
		Module:  "golang.org/x/tools/cmd/stringer",
		Version: "v0.1.0",
		Binary:  "strgen",
	})

	if name != "strgen" {
		t.Fatalf("unexpected binary override: %q", name)
	}
}

func TestAppendBinToEnvPrependsPathAndSetsVariable(t *testing.T) {
	t.Parallel()

	env := AppendBinToEnv([]string{"PATH=/usr/bin", "HOME=/tmp/demo"}, "/workspace/demo")
	joined := strings.Join(env, "\n")

	if !strings.Contains(joined, "GPM_TOOLS_BIN="+filepath.Join("/workspace/demo", ".gpm", "tools", "bin")) {
		t.Fatal("expected GPM_TOOLS_BIN to be set")
	}
	if !strings.Contains(joined, "PATH="+filepath.Join("/workspace/demo", ".gpm", "tools", "bin")+string(os.PathListSeparator)+"/usr/bin") {
		t.Fatal("expected tool bin dir to be prepended to PATH")
	}
}

func TestBinaryPathUsesExecutableSuffixOnWindows(t *testing.T) {
	t.Parallel()

	path := BinaryPath("/workspace/demo", manifest.Tool{
		Module:  "github.com/golangci/golangci-lint/cmd/golangci-lint",
		Version: "v1.64.8",
	})
	if runtime.GOOS == "windows" && !strings.HasSuffix(path, ".exe") {
		t.Fatal("expected .exe suffix on windows")
	}
}
