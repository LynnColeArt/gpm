package deps

import "testing"

func TestParseOutdatedModulesFiltersDirectByDefault(t *testing.T) {
	t.Parallel()

	data := []byte(`
{"Path":"example.com/main","Main":true}
{"Path":"github.com/google/uuid","Version":"v1.5.0","Update":{"Version":"v1.6.0"}}
{"Path":"golang.org/x/text","Version":"v0.23.0","Indirect":true,"Update":{"Version":"v0.24.0"}}
`)

	modules, err := parseOutdatedModules(data, false)
	if err != nil {
		t.Fatalf("parse outdated modules: %v", err)
	}

	if len(modules) != 1 {
		t.Fatalf("unexpected modules: %#v", modules)
	}
	if modules[0].Path != "github.com/google/uuid" || modules[0].Latest != "v1.6.0" {
		t.Fatalf("unexpected direct module: %#v", modules[0])
	}
}

func TestParseOutdatedModulesIncludesIndirectWithFlag(t *testing.T) {
	t.Parallel()

	data := []byte(`
{"Path":"github.com/google/uuid","Version":"v1.5.0","Update":{"Version":"v1.6.0"}}
{"Path":"golang.org/x/text","Version":"v0.23.0","Indirect":true,"Update":{"Version":"v0.24.0"}}
`)

	modules, err := parseOutdatedModules(data, true)
	if err != nil {
		t.Fatalf("parse outdated modules: %v", err)
	}

	if len(modules) != 2 {
		t.Fatalf("unexpected modules: %#v", modules)
	}
	if !modules[1].Indirect {
		t.Fatalf("expected indirect module, got %#v", modules[1])
	}
}

func TestParseOutdatedModulesIncludesRetractedDeprecatedAndProblemStates(t *testing.T) {
	t.Parallel()

	data := []byte(`
{"Path":"github.com/old/retracted","Version":"v1.0.0","Retracted":["broken release"]}
{"Path":"github.com/old/deprecated","Version":"v1.0.0","Deprecated":"use github.com/new/module"}
{"Path":"github.com/old/problem","Version":"v1.0.0","Error":{"Err":"unknown revision"}}
`)

	modules, err := parseOutdatedModules(data, true)
	if err != nil {
		t.Fatalf("parse outdated modules: %v", err)
	}

	if len(modules) != 3 {
		t.Fatalf("unexpected modules: %#v", modules)
	}
	if len(modules[0].Retracted) == 0 {
		t.Fatalf("expected retracted entry: %#v", modules[0])
	}
	if modules[1].Deprecated == "" {
		t.Fatalf("expected deprecated entry: %#v", modules[1])
	}
	if modules[2].Problem == "" {
		t.Fatalf("expected problem entry: %#v", modules[2])
	}
}

func TestParseOutdatedModulesSkipsUpToDateEntries(t *testing.T) {
	t.Parallel()

	data := []byte(`
{"Path":"github.com/ok/module","Version":"v1.0.0"}
`)

	modules, err := parseOutdatedModules(data, true)
	if err != nil {
		t.Fatalf("parse outdated modules: %v", err)
	}
	if len(modules) != 0 {
		t.Fatalf("expected no actionable modules, got %#v", modules)
	}
}
