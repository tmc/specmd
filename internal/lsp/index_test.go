package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceIndexDefinitionsAndReferences(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "00-object-catalog.md", strings.Join([]string{
		"# Catalog",
		"",
		"| Object | One-line definition | Domain | Status |",
		"|---|---|---|---|",
		"| **Variant** | Alternate generated thread. | variants | current |",
		"",
		"## Cross-tier Object Map",
		"  Variant --> Thread",
		"",
	}, "\n"))
	writeFile(t, root, "objects/t6-quality-and-self-improvement.md", "# T6\n\n## Variant   `status: current`\n")
	writeFile(t, root, "matrices/relationship-map.md", "See [map](../00-object-catalog.md#cross-tier-object-map).\n\n| Variant | Thread |\n")

	s := NewServer(nil, nil)
	s.setRoot(initializeParams{RootURI: uriFromPath(root)})
	rel := uriFromPath(filepath.Join(root, "matrices/relationship-map.md"))
	catalog := uriFromPath(filepath.Join(root, "00-object-catalog.md"))
	object := uriFromPath(filepath.Join(root, "objects/t6-quality-and-self-improvement.md"))

	locs := s.definitions(catalog, position{Line: 4, Character: 5})
	if len(locs) != 1 {
		t.Fatalf("len(catalog definitions) = %d, want 1: %+v", len(locs), locs)
	}
	if got, want := locs[0].URI, object; got != want {
		t.Fatalf("catalog definition URI = %q, want %q", got, want)
	}

	locs = s.definitions(rel, position{Line: 0, Character: 6})
	if len(locs) != 1 {
		t.Fatalf("len(markdown link definitions) = %d, want 1: %+v", len(locs), locs)
	}
	if got, want := locs[0].URI, catalog; got != want {
		t.Fatalf("markdown definition URI = %q, want %q", got, want)
	}
	if got, want := locs[0].Range.Start.Line, 6; got != want {
		t.Fatalf("markdown definition line = %d, want %d", got, want)
	}

	refs := s.references(catalog, position{Line: 4, Character: 5})
	if len(refs) < 3 {
		t.Fatalf("len(references) = %d, want at least 3: %+v", len(refs), refs)
	}
}

func TestWorkspaceIndexGraphDiagnostics(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "00-object-catalog.md", strings.Join([]string{
		"# Catalog",
		"| Object | One-line definition | Domain | Status |",
		"|---|---|---|---|",
		"| **Missing Detail** | No card yet. | none | planned |",
		"",
		"See [missing](missing.md), [heading](objects/card.md#missing-heading), [[Duplicate]], and openspec/specs/auth/spec.md.",
	}, "\n"))
	writeFile(t, root, "objects/card.md", "# Cards\n\n## Orphan Object\n")
	writeFile(t, root, "a/duplicate.md", "# Duplicate\n")
	writeFile(t, root, "b/duplicate.md", "# Duplicate\n")
	writeFile(t, root, "openspec/specs/auth/spec.md", "# Auth\n\n## Purpose\n\n## Requirements\n")

	s := NewServer(nil, nil)
	s.setRoot(initializeParams{RootURI: uriFromPath(root)})
	catalog := uriFromPath(filepath.Join(root, "00-object-catalog.md"))
	card := uriFromPath(filepath.Join(root, "objects/card.md"))

	diags := s.graphDiagnostics(catalog)
	for _, msg := range []string{
		"catalog object has no matching object detail heading",
		"broken local markdown link",
		"missing markdown heading target",
		"ambiguous markdown link target",
	} {
		if !hasDiagnostic(diags, msg) {
			t.Fatalf("catalog diagnostics missing %q: %+v", msg, diags)
		}
	}
	if len(s.resolveDocMatches("openspec/specs/auth/spec.md")) != 0 {
		t.Fatalf("resolveDocMatches should not treat literal openspec paths as ambiguous document names")
	}
	if !hasDiagnostic(s.graphDiagnostics(card), "object detail heading has no catalog row") {
		t.Fatalf("card diagnostics missing orphan object warning")
	}
}

func TestWorkspaceIndexCompletionAndHover(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "00-object-catalog.md", "# Catalog\n\n| Object | One-line definition |\n|---|---|\n| **Variant** | Alternate generated thread. |\n")
	writeFile(t, root, "objects/t6.md", "# T6\n\n## Variant   `status: current`\n")
	writeFile(t, root, "matrices/cta.md", "|  |\n")

	s := NewServer(nil, nil)
	s.setRoot(initializeParams{RootURI: uriFromPath(root)})
	cta := uriFromPath(filepath.Join(root, "matrices/cta.md"))
	items := s.completions(cta, s.text(cta), position{Line: 0, Character: 2})
	if !hasCompletion(items, "Variant") {
		t.Fatalf("completion missing Variant: %+v", items)
	}
	hover := s.hoverAt(cta, "| Variant |\n", position{Line: 0, Character: 3})
	if !strings.Contains(hover, "Alternate generated thread.") {
		t.Fatalf("hover = %q, want catalog detail", hover)
	}
}

func TestOOUXIntegrityDiagnostics(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "00-object-catalog.md", strings.Join([]string{
		"# Catalog",
		"",
		"| Object | One-line definition | Domain | Status |",
		"|---|---|---|---|",
		"| **Variant** | Alternate generated thread. | variants | current |",
	}, "\n"))
	writeFile(t, root, "objects/t6.md", "# T6\n\n## Variant   `status: planned`\n")
	writeFile(t, root, "matrices/relationship-map.md", "| Missing Object | Variant |\n")
	writeFile(t, root, "matrices/cta-matrix.md", "| Other Variant | generate |\n")

	s := NewServer(nil, nil)
	s.setRoot(initializeParams{RootURI: uriFromPath(root)})
	catalog := uriFromPath(filepath.Join(root, "00-object-catalog.md"))
	rel := uriFromPath(filepath.Join(root, "matrices/relationship-map.md"))

	for _, msg := range []string{
		"object status differs from catalog row",
		"current ooux object has no cta row",
	} {
		if !hasDiagnostic(s.graphDiagnostics(catalog), msg) {
			t.Fatalf("catalog diagnostics missing %q: %+v", msg, s.graphDiagnostics(catalog))
		}
	}
	if !hasDiagnostic(s.graphDiagnostics(rel), "referenced ooux object is missing from catalog") {
		t.Fatalf("relationship diagnostics missing unknown object warning: %+v", s.graphDiagnostics(rel))
	}
}

func writeFile(t *testing.T, root, name, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0666); err != nil {
		t.Fatal(err)
	}
}
