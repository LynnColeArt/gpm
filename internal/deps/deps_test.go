package deps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeTargetRejectsEmptyValue(t *testing.T) {
	t.Parallel()

	if _, err := NormalizeTarget("   "); err == nil {
		t.Fatal("expected empty dependency target to fail")
	}
}

func TestNormalizeTargetKeepsVersionedModule(t *testing.T) {
	t.Parallel()

	target, err := NormalizeTarget("github.com/google/uuid@v1.6.0")
	if err != nil {
		t.Fatalf("normalize target: %v", err)
	}

	if target != "github.com/google/uuid@v1.6.0" {
		t.Fatalf("unexpected normalized target: %q", target)
	}
}

func TestModulePathStripsVersionSuffix(t *testing.T) {
	t.Parallel()

	module := ModulePath("github.com/google/uuid@v1.6.0")
	if module != "github.com/google/uuid" {
		t.Fatalf("unexpected module path: %q", module)
	}
}

func TestRequirementExistsReportsRequiredModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goMod := "module example.com/demo\n\ngo 1.25.4\n\nrequire github.com/google/uuid v1.6.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	exists, err := RequirementExists(root, "github.com/google/uuid")
	if err != nil {
		t.Fatalf("check requirement exists: %v", err)
	}
	if !exists {
		t.Fatal("expected dependency to be required")
	}
}

func TestRequirementExistsReportsMissingModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goMod := "module example.com/demo\n\ngo 1.25.4\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	exists, err := RequirementExists(root, "github.com/google/uuid")
	if err != nil {
		t.Fatalf("check requirement exists: %v", err)
	}
	if exists {
		t.Fatal("expected dependency to be absent")
	}
}

func TestRequirementsIncludeVersionAndIndirectState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goMod := "module example.com/demo\n\ngo 1.25.4\n\nrequire (\n\tgithub.com/google/uuid v1.6.0\n\tgolang.org/x/text v0.23.0 // indirect\n)\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	requirements, err := Requirements(root)
	if err != nil {
		t.Fatalf("read requirements: %v", err)
	}

	if len(requirements) != 2 {
		t.Fatalf("unexpected requirements: %#v", requirements)
	}
	if requirements[0].Path != "github.com/google/uuid" || requirements[0].Version != "v1.6.0" || requirements[0].Indirect {
		t.Fatalf("unexpected first requirement: %#v", requirements[0])
	}
	if requirements[1].Path != "golang.org/x/text" || requirements[1].Version != "v0.23.0" || !requirements[1].Indirect {
		t.Fatalf("unexpected second requirement: %#v", requirements[1])
	}
}

func TestBuildUpdatePlanUsesDirectRequirementsWhenNoTargetsProvided(t *testing.T) {
	t.Parallel()

	plan, err := BuildUpdatePlan([]Requirement{
		{Path: "golang.org/x/text", Version: "v0.23.0", Indirect: true},
		{Path: "github.com/google/uuid", Version: "v1.6.0"},
		{Path: "github.com/charmbracelet/glow", Version: "v0.10.0"},
	}, nil)
	if err != nil {
		t.Fatalf("build update plan: %v", err)
	}

	if len(plan.ExactTargets) != 0 {
		t.Fatalf("expected no exact targets: %#v", plan.ExactTargets)
	}
	if len(plan.UpgradeTargets) != 2 {
		t.Fatalf("unexpected upgrade targets: %#v", plan.UpgradeTargets)
	}
	if plan.UpgradeTargets[0] != "github.com/charmbracelet/glow" || plan.UpgradeTargets[1] != "github.com/google/uuid" {
		t.Fatalf("unexpected direct upgrade targets: %#v", plan.UpgradeTargets)
	}
}

func TestBuildUpdatePlanSeparatesUpgradeAndExactTargets(t *testing.T) {
	t.Parallel()

	plan, err := BuildUpdatePlan(nil, []string{
		"github.com/google/uuid",
		"github.com/charmbracelet/glow@v1.5.0",
		"github.com/google/uuid",
		"github.com/charmbracelet/glow@v1.5.0",
	})
	if err != nil {
		t.Fatalf("build update plan: %v", err)
	}

	if len(plan.UpgradeTargets) != 1 || plan.UpgradeTargets[0] != "github.com/google/uuid" {
		t.Fatalf("unexpected upgrade targets: %#v", plan.UpgradeTargets)
	}
	if len(plan.ExactTargets) != 1 || plan.ExactTargets[0] != "github.com/charmbracelet/glow@v1.5.0" {
		t.Fatalf("unexpected exact targets: %#v", plan.ExactTargets)
	}
}
