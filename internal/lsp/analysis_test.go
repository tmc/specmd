package lsp

import "testing"

func TestAnalyzeSpecDiagnostics(t *testing.T) {
	diags := analyze("file:///repo/openspec/specs/auth/spec.md", "# Auth\n\n## Purpose\n")
	if len(diags) != 1 {
		t.Fatalf("len(diags) = %d, want 1: %+v", len(diags), diags)
	}
	if got, want := diags[0].Message, "missing ## Requirements section"; got != want {
		t.Fatalf("Message = %q, want %q", got, want)
	}
}

func TestAnalyzeExtensionDiagnostics(t *testing.T) {
	diags := analyze("file:///repo/openspec/extensions/example-mapping/auth.md", "# Example Mapping\n\n## Rules\n\n## Examples\n")
	if len(diags) != 2 {
		t.Fatalf("len(diags) = %d, want 2: %+v", len(diags), diags)
	}
	if !hasDiagnostic(diags, "missing ## Story section") || !hasDiagnostic(diags, "missing ## Questions section") {
		t.Fatalf("missing expected diagnostics: %+v", diags)
	}
}

func TestCompletionsForExtension(t *testing.T) {
	tests := []struct {
		uri   string
		label string
	}{
		{"file:///repo/openspec/extensions/ooux/model.md", "## Objects"},
		{"file:///repo/openspec/extensions/eventstorm/model.md", "## Events"},
		{"file:///repo/openspec/extensions/contexts/map.md", "## Relationships"},
		{"file:///repo/openspec/extensions/domain-story/model.md", "## Story"},
		{"file:///repo/openspec/extensions/example-mapping/auth.md", "## Questions"},
		{"file:///repo/openspec/extensions/jobs/stories.md", "## Stories"},
		{"file:///repo/openspec/extensions/journey/login.md", "## Actor"},
		{"file:///repo/openspec/extensions/opportunity-tree/auth.md", "## Experiments"},
		{"file:///repo/openspec/extensions/service-blueprint/login.md", "## Blueprint"},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			items := completions(tt.uri, "")
			if !hasCompletion(items, tt.label) {
				t.Fatalf("missing %s completion: %+v", tt.label, items)
			}
		})
	}
}

func TestCompletionsPutMissingSectionsFirst(t *testing.T) {
	items := completions("file:///repo/openspec/specs/auth/spec.md", "# Auth\n\n## Purpose\n")
	if len(items) == 0 {
		t.Fatal("no completions")
	}
	if got, want := items[0].Label, "## Requirements"; got != want {
		t.Fatalf("first completion = %q, want %q", got, want)
	}
	if got, want := items[0].Detail, "missing required OpenSpec section"; got != want {
		t.Fatalf("first detail = %q, want %q", got, want)
	}
}

func TestHoverAtHeading(t *testing.T) {
	text := "# Auth\n\n## Requirements\n\n### Requirement: Login\n"
	if got, want := hoverAt("file:///repo/openspec/specs/auth/spec.md", text, position{Line: 2}), "Requirements contain user-visible behavior and scenarios."; got != want {
		t.Fatalf("hover = %q, want %q", got, want)
	}
	if got, want := hoverAt("file:///repo/openspec/specs/auth/spec.md", text, position{Line: 4}), "Requirement headings name one behavior contract."; got != want {
		t.Fatalf("hover = %q, want %q", got, want)
	}
}

func TestSymbols(t *testing.T) {
	syms := symbols("# Title\n\n## Purpose\n\n### Requirement: Login\n")
	if got, want := len(syms), 3; got != want {
		t.Fatalf("len(symbols) = %d, want %d", got, want)
	}
	if got, want := syms[2].Name, "Requirement: Login"; got != want {
		t.Fatalf("Name = %q, want %q", got, want)
	}
}

func hasCompletion(items []completionItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}

func hasDiagnostic(diags []diagnostic, msg string) bool {
	for _, diag := range diags {
		if diag.Message == msg {
			return true
		}
	}
	return false
}
