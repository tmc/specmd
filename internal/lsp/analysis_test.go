package lsp

import (
	"strings"
	"testing"
)

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

func TestCompletionsIncludeBlocksAndFields(t *testing.T) {
	tests := []struct {
		name  string
		uri   string
		text  string
		label string
	}{
		{"requirement block", "file:///repo/openspec/specs/auth/spec.md", "", "Requirement block"},
		{"scenario field", "file:///repo/openspec/specs/auth/spec.md", "", "GIVEN field"},
		{"proposal section", "file:///repo/openspec/changes/add-auth/proposal.md", "", "## What Changes"},
		{"delta block", "file:///repo/openspec/changes/add-auth/specs/auth/spec.md", "", "ADDED requirement block"},
		{"ooux subheading", "file:///repo/openspec/extensions/ooux/model.md", "", "#### Attributes"},
		{"ooux block", "file:///repo/openspec/extensions/ooux/model.md", "", "OOUX object block"},
		{"context relationship", "file:///repo/openspec/extensions/contexts/map.md", "", "context relationship"},
		{"domain story step", "file:///repo/openspec/extensions/domain-story/model.md", "", "domain story step"},
		{"eventstorm field", "file:///repo/openspec/extensions/eventstorm/model.md", "", "command field"},
		{"example field", "file:///repo/openspec/extensions/example-mapping/auth.md", "", "question field"},
		{"job story", "file:///repo/openspec/extensions/jobs/stories.md", "", "job story"},
		{"opportunity field", "file:///repo/openspec/extensions/opportunity-tree/login.md", "", "experiment field"},
		{"journey stage", "file:///repo/openspec/extensions/journey/login.md", "", "stage block"},
		{"blueprint field", "file:///repo/openspec/extensions/service-blueprint/login.md", "", "frontstage field"},
		{"stratmd objective", "file:///repo/openspec/extensions/stratmd/strategy.md", "", "strategy objective"},
		{"magi typed block", "file:///repo/openspec/extensions/magi/context.md", "", "magi typed block"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := completions(tt.uri, tt.text)
			if !hasCompletion(items, tt.label) {
				t.Fatalf("missing %s completion: %+v", tt.label, items)
			}
		})
	}
}

func TestCompletionSnippetsUseSnippetFormat(t *testing.T) {
	items := completions("file:///repo/openspec/specs/auth/spec.md", "")
	item, ok := completionByLabel(items, "Requirement block")
	if !ok {
		t.Fatalf("missing Requirement block completion: %+v", items)
	}
	if got, want := item.InsertTextFormat, insertTextSnippet; got != want {
		t.Fatalf("InsertTextFormat = %d, want %d", got, want)
	}
	if got, want := item.InsertText, "### Requirement: ${1:name}\n\n#### Scenario: ${2:name}\n\n- GIVEN ${3:context}\n- WHEN ${4:action}\n- THEN ${5:outcome}\n"; got != want {
		t.Fatalf("InsertText = %q, want %q", got, want)
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

func TestHoverForExtensionFamilies(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{"file:///repo/openspec/extensions/domain-story/model.md", "Domain Storytelling extension"},
		{"file:///repo/openspec/extensions/stratmd/strategy.md", "StratMD extension"},
		{"file:///repo/openspec/extensions/magi/context.md", "MAGI extension"},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			if got := hoverFor(tt.uri); !strings.Contains(got, tt.want) {
				t.Fatalf("hoverFor(%q) = %q, want substring %q", tt.uri, got, tt.want)
			}
		})
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

func TestSymbolsUseUTF16Ranges(t *testing.T) {
	syms := symbols("# 🔐 Auth\n")
	if got, want := syms[0].Range.End.Character, 9; got != want {
		t.Fatalf("End.Character = %d, want %d", got, want)
	}
}

func TestCodeActions(t *testing.T) {
	uri := "file:///repo/openspec/specs/auth/spec.md"
	text := "# Auth\n\n## Purpose\n\n## Requiement\n"
	actions := codeActions(uri, text, analyze(uri, text))
	for _, label := range []string{"Insert ## Requirements", "Fix heading to ## Requirements", "Insert requirement scenario skeleton"} {
		if !hasCodeAction(actions, label) {
			t.Fatalf("missing %q action: %+v", label, actions)
		}
	}
}

func TestDocumentPathLinks(t *testing.T) {
	links := pathLinks("See openspec/specs/auth/spec.md for details.\n")
	if len(links) != 1 {
		t.Fatalf("len(pathLinks) = %d, want 1", len(links))
	}
	if got, want := links[0].Target, "openspec/specs/auth/spec.md"; got != want {
		t.Fatalf("Target = %q, want %q", got, want)
	}
}

func TestWorkspaceSymbols(t *testing.T) {
	s := NewServer(strings.NewReader(""), nil)
	s.docs["file:///repo/openspec/specs/auth/spec.md"] = "# Auth\n\n## Requirements\n\n### Requirement: Login\n"
	s.docs["file:///repo/openspec/extensions/ooux/model.md"] = "# OOUX model\n\n## Objects\n"
	syms := s.workspaceSymbols("login")
	if len(syms) != 1 {
		t.Fatalf("len(workspaceSymbols) = %d, want 1: %+v", len(syms), syms)
	}
	if got, want := syms[0].Name, "Requirement: Login"; got != want {
		t.Fatalf("Name = %q, want %q", got, want)
	}
}

func TestFoldingRanges(t *testing.T) {
	ranges := foldingRanges("# Auth\n\n## Requirements\n\n### Requirement: Login\n\ntext\n")
	if len(ranges) < 2 {
		t.Fatalf("len(foldingRanges) = %d, want at least 2: %+v", len(ranges), ranges)
	}
	if got, want := ranges[0].StartLine, 0; got != want {
		t.Fatalf("StartLine = %d, want %d", got, want)
	}
}

func TestSelectionRanges(t *testing.T) {
	ranges := selectionRanges("# Auth\n\n## Requirements\n\ntext\n", []position{{Line: 3, Character: 0}})
	if len(ranges) != 1 {
		t.Fatalf("len(selectionRanges) = %d, want 1", len(ranges))
	}
	if got, want := ranges[0].Range.Start.Line, 2; got != want {
		t.Fatalf("Range.Start.Line = %d, want %d", got, want)
	}
	if ranges[0].Parent == nil {
		t.Fatal("Parent is nil, want document range")
	}
}

func hasCompletion(items []completionItem, label string) bool {
	_, ok := completionByLabel(items, label)
	return ok
}

func completionByLabel(items []completionItem, label string) (completionItem, bool) {
	for _, item := range items {
		if item.Label == label {
			return item, true
		}
	}
	return completionItem{}, false
}

func hasDiagnostic(diags []diagnostic, msg string) bool {
	for _, diag := range diags {
		if diag.Message == msg {
			return true
		}
	}
	return false
}

func hasCodeAction(actions []codeAction, title string) bool {
	for _, action := range actions {
		if action.Title == title {
			return true
		}
	}
	return false
}
