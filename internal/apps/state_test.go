package apps

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildInstallRecordForModuleTarget(t *testing.T) {
	t.Parallel()

	record, err := BuildInstallRecord(
		"github.com/charmbracelet/glow",
		"github.com/charmbracelet/glow@latest",
		ScopeUser,
		"/tmp/bin",
		"/tmp/project",
		time.Unix(10, 0),
	)
	if err != nil {
		t.Fatalf("build install record: %v", err)
	}

	if record.BinaryName != "glow" {
		t.Fatalf("unexpected binary name: %q", record.BinaryName)
	}
	if record.BinaryPath != filepath.Join("/tmp/bin", "glow") {
		t.Fatalf("unexpected binary path: %q", record.BinaryPath)
	}
}

func TestBuildInstallRecordForLocalTarget(t *testing.T) {
	t.Parallel()

	record, err := BuildInstallRecord(
		"./cmd/gpm",
		"./cmd/gpm",
		ScopeUser,
		"/tmp/bin",
		"/workspace/repo",
		time.Unix(20, 0),
	)
	if err != nil {
		t.Fatalf("build install record: %v", err)
	}

	if record.BinaryName != "gpm" {
		t.Fatalf("unexpected binary name: %q", record.BinaryName)
	}
}

func TestRemoveInstallMatchesRequestedTargetAndName(t *testing.T) {
	t.Parallel()

	state := &State{
		Installs: []InstallRecord{
			{
				RequestedTarget: "github.com/charmbracelet/glow",
				Target:          "github.com/charmbracelet/glow@latest",
				Scope:           ScopeUser,
				BinaryName:      "glow",
				BinaryPath:      "/tmp/bin/glow",
			},
		},
	}

	if _, ok := RemoveInstall(state, ScopeUser, "glow"); !ok {
		t.Fatal("expected remove by binary name to succeed")
	}

	state.Installs = []InstallRecord{
		{
			RequestedTarget: "github.com/charmbracelet/glow",
			Target:          "github.com/charmbracelet/glow@latest",
			Scope:           ScopeUser,
			BinaryName:      "glow",
			BinaryPath:      "/tmp/bin/glow",
		},
	}
	if _, ok := RemoveInstall(state, ScopeUser, "github.com/charmbracelet/glow"); !ok {
		t.Fatal("expected remove by requested target to succeed")
	}
}
