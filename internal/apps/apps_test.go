package apps

import (
	"strings"
	"testing"

	"github.com/LynnColeArt/gpm/internal/config"
)

func TestNormalizeTargetAddsLatest(t *testing.T) {
	t.Parallel()

	target, err := NormalizeTarget("github.com/charmbracelet/glow")
	if err != nil {
		t.Fatalf("normalize target: %v", err)
	}

	if target != "github.com/charmbracelet/glow@latest" {
		t.Fatalf("unexpected target: %q", target)
	}
}

func TestNormalizeTargetKeepsLocalPath(t *testing.T) {
	t.Parallel()

	target, err := NormalizeTarget("./cmd/gpm")
	if err != nil {
		t.Fatalf("normalize target: %v", err)
	}

	if target != "./cmd/gpm" {
		t.Fatalf("unexpected target: %q", target)
	}
}

func TestResolveBinDirUsesConfigOverride(t *testing.T) {
	t.Parallel()

	dir, err := ResolveBinDir(ScopeUser, &config.File{
		Binaries: config.BinaryConfig{UserDir: "~/custom/bin"},
	})
	if err != nil {
		t.Fatalf("resolve bin dir: %v", err)
	}

	if !strings.Contains(dir, "custom") {
		t.Fatalf("expected custom bin dir, got %q", dir)
	}
}

func TestListAllReturnsSortedInstalls(t *testing.T) {
	t.Parallel()

	state := &State{
		Installs: []InstallRecord{
			{Scope: ScopeUser, BinaryName: "zeta", BinaryPath: "/tmp/zeta"},
			{Scope: ScopeGlobal, BinaryName: "alpha", BinaryPath: "/usr/local/bin/alpha"},
		},
	}

	records := List(state, "all")
	if len(records) != 2 {
		t.Fatalf("expected 2 installs, got %d", len(records))
	}
	if records[0].Scope != ScopeGlobal {
		t.Fatalf("expected global installs to sort first, got %q", records[0].Scope)
	}
}
