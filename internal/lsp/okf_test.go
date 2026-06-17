package lsp

import (
	"strings"
	"testing"
)

func TestOKFDiagnosticsMissingType(t *testing.T) {
	text := "---\ntitle: Orders\n---\n\nBody.\n"
	diags := okfDiagnostics("file:///kb/datasets/orders.md", text)
	if !hasDiagnostic(diags, "OKF type: cannot be empty") {
		t.Fatalf("missing type diagnostic: %+v", diags)
	}
	for _, d := range diags {
		if d.Code != "okf" {
			t.Fatalf("Code = %q, want okf", d.Code)
		}
	}
}

func TestOKFDiagnosticsAnchorTitle(t *testing.T) {
	// A present-but-empty optional field should anchor on its own line.
	text := "---\ntype: Dataset\ntitle:\n---\n\nBody.\n"
	diags := okfDiagnostics("file:///kb/datasets/orders.md", text)
	var found bool
	for _, d := range diags {
		if strings.Contains(d.Message, "title") {
			found = true
			if d.Range.Start.Line != 2 {
				t.Fatalf("title diagnostic Line = %d, want 2: %+v", d.Range.Start.Line, d)
			}
		}
	}
	if !found {
		t.Fatalf("no title diagnostic: %+v", diags)
	}
}

func TestOKFDiagnosticsValidConcept(t *testing.T) {
	text := "---\ntype: Dataset\ntitle: Orders\ndescription: Order facts.\n---\n\nBody.\n"
	if diags := okfDiagnostics("file:///kb/datasets/orders.md", text); len(diags) != 0 {
		t.Fatalf("len(diags) = %d, want 0: %+v", len(diags), diags)
	}
}

func TestOKFDiagnosticsSkipsNonConcept(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		text string
	}{
		{"spec", "file:///repo/openspec/specs/auth/spec.md", "---\ntitle: Auth\n---\n\n# Auth\n"},
		{"reserved index", "file:///kb/index.md", "---\nokf_version: \"0.1\"\n---\n"},
		{"reserved log", "file:///kb/log.md", "## 2026-01-01\n"},
		{"no frontmatter", "file:///kb/datasets/orders.md", "# Orders\n\nBody.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diags := okfDiagnostics(tt.uri, tt.text); diags != nil {
				t.Fatalf("diags = %+v, want nil", diags)
			}
		})
	}
}

func TestOKFDiagnosticsTolerantOfFenceWhitespace(t *testing.T) {
	// Editors often leave trailing whitespace on the "---" fences. That must
	// not produce a hard parse error; the concept should validate normally.
	text := "--- \ntype: Dataset\n--- \n\nBody.\n"
	diags := okfDiagnostics("file:///kb/datasets/orders.md", text)
	for _, d := range diags {
		if d.Severity == 1 {
			t.Fatalf("unexpected error diagnostic on whitespace fence: %+v", d)
		}
	}
}

func TestOKFDiagnosticsQuietWhileFrontMatterOpen(t *testing.T) {
	// A front-matter block that has no closing fence yet (still being typed)
	// must not flash a parse error on every keystroke.
	text := "---\ntype: Dataset\ntitle: Orders\n"
	if diags := okfDiagnostics("file:///kb/datasets/orders.md", text); len(diags) != 0 {
		t.Fatalf("diags = %+v, want none while front matter is open", diags)
	}
}

func TestOKFCompletionsWhileFrontMatterOpen(t *testing.T) {
	// Completions must stay available while the closing fence is unwritten.
	text := "---\ntype: Dataset\n"
	items := okfCompletions("file:///kb/datasets/orders.md", text, position{Line: 1, Character: 0})
	if !hasCompletion(items, "title:") {
		t.Fatalf("missing title completion in open front matter: %+v", items)
	}
}

func TestOKFCompletionsInFrontMatter(t *testing.T) {
	text := "---\ntype: Dataset\n\n---\n\nBody.\n"
	items := okfCompletions("file:///kb/datasets/orders.md", text, position{Line: 2, Character: 0})
	if !hasCompletion(items, "title:") {
		t.Fatalf("missing title completion: %+v", items)
	}
	// type is already present and must not be offered again.
	if hasCompletion(items, "type:") {
		t.Fatalf("type offered despite being present: %+v", items)
	}
}

func TestOKFCompletionsOnlyInFrontMatter(t *testing.T) {
	text := "---\ntype: Dataset\n---\n\nBody.\n"
	if items := okfCompletions("file:///kb/datasets/orders.md", text, position{Line: 4, Character: 0}); items != nil {
		t.Fatalf("completions offered in body: %+v", items)
	}
}

func TestOKFHover(t *testing.T) {
	text := "---\ntype: Dataset\ntitle: Orders\ntags: [a, b]\n---\n\nBody.\n"
	got := okfHover("file:///kb/datasets/orders.md", text)
	for _, want := range []string{"OKF concept", "type: Dataset", "title: Orders", "tags: a, b"} {
		if !strings.Contains(got, want) {
			t.Fatalf("hover = %q, want substring %q", got, want)
		}
	}
}

func TestOKFHoverSkipsNonConcept(t *testing.T) {
	if got := okfHover("file:///repo/openspec/specs/auth/spec.md", "---\ntitle: Auth\n---\n"); got != "" {
		t.Fatalf("hover = %q, want empty", got)
	}
}

func TestAnalyzeIncludesOKFDiagnostics(t *testing.T) {
	diags := analyze("file:///kb/datasets/orders.md", "---\ntitle: Orders\n---\n\nBody.\n")
	if !hasDiagnostic(diags, "OKF type: cannot be empty") {
		t.Fatalf("analyze omitted OKF diagnostic: %+v", diags)
	}
}

func TestServerOKFConceptCompletionAndHover(t *testing.T) {
	s := NewServer(nil, nil)
	uri := "file:///kb/datasets/orders.md"
	text := "---\ntype: Dataset\n\n---\n\nBody.\n"
	s.docs[uri] = text
	items := s.completions(uri, text, position{Line: 2, Character: 0})
	if !hasCompletion(items, "description:") {
		t.Fatalf("server completions missing description: %+v", items)
	}
	if hover := s.hoverAt(uri, text, position{Line: 1, Character: 0}); !strings.Contains(hover, "type: Dataset") {
		t.Fatalf("server hover = %q, want OKF summary", hover)
	}
}
