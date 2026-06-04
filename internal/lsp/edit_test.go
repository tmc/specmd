package lsp

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReferenceStyleLinks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "index.md", "See [catalog][cat].\n\n[cat]: 00-object-catalog.md#objects\n")
	writeFile(t, root, "00-object-catalog.md", "# Catalog\n\n## Objects\n")
	s := NewServer(nil, nil)
	s.setRoot(initializeParams{RootURI: uriFromPath(root)})
	index := uriFromPath(filepath.Join(root, "index.md"))
	catalog := uriFromPath(filepath.Join(root, "00-object-catalog.md"))

	locs := s.definitions(index, position{Line: 0, Character: 6})
	if len(locs) != 1 {
		t.Fatalf("len(definitions) = %d, want 1: %+v", len(locs), locs)
	}
	if got, want := locs[0].URI, catalog; got != want {
		t.Fatalf("URI = %q, want %q", got, want)
	}
	if got, want := locs[0].Range.Start.Line, 2; got != want {
		t.Fatalf("line = %d, want %d", got, want)
	}
	if got := s.documentLinks(index); len(got) != 1 {
		t.Fatalf("len(documentLinks) = %d, want 1: %+v", len(got), got)
	}
}

func TestRenameObjectName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "00-object-catalog.md", "# Catalog\n\n| Object | One-line definition | Domain | Status |\n|---|---|---|---|\n| **Variant** | Alternate. | variants | current |\n")
	writeFile(t, root, "objects/t6.md", "# T6\n\n## Variant   `status: current`\n")
	writeFile(t, root, "matrices/cta-matrix.md", "| Variant | generate |\n")
	s := NewServer(nil, nil)
	s.setRoot(initializeParams{RootURI: uriFromPath(root)})
	catalog := uriFromPath(filepath.Join(root, "00-object-catalog.md"))

	edit := s.rename(catalog, position{Line: 4, Character: 5}, "Experiment Variant")
	if len(edit.Changes) < 3 {
		t.Fatalf("rename changed %d files, want at least 3: %+v", len(edit.Changes), edit.Changes)
	}
	for uri, edits := range edit.Changes {
		if len(edits) == 0 {
			t.Fatalf("no edits for %s", uri)
		}
	}
}

func TestCodeLensAndInlayHints(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "00-object-catalog.md", "# Catalog\n\n| Object | One-line definition | Domain | Status |\n|---|---|---|---|\n| **Variant** | Alternate. | variants | current |\n")
	writeFile(t, root, "objects/t6.md", "# T6\n\n## Variant   `status: current`\n")
	writeFile(t, root, "matrices/cta-matrix.md", "| Variant | generate |\n")
	s := NewServer(nil, nil)
	s.setRoot(initializeParams{RootURI: uriFromPath(root)})
	object := uriFromPath(filepath.Join(root, "objects/t6.md"))

	lens := s.codeLens(object)
	if len(lens) == 0 || !strings.Contains(lens[0].Command.Title, "references") {
		t.Fatalf("codeLens = %+v, want references", lens)
	}
	hints := s.inlayHints(object, textRange{Start: position{}, End: position{Line: 10}})
	if len(hints) == 0 || !strings.Contains(hints[0].Label, "references") {
		t.Fatalf("inlayHints = %+v, want references", hints)
	}
}

func TestLinkCodeActions(t *testing.T) {
	root := t.TempDir()
	text := "See [bad](target.md#missing).\n\n[unused]: target.md\n[dup]: target.md\n[dup]: other.md\n"
	writeFile(t, root, "source.md", text)
	writeFile(t, root, "target.md", "# Target\n")
	s := NewServer(nil, nil)
	s.setRoot(initializeParams{RootURI: uriFromPath(root)})
	source := uriFromPath(filepath.Join(root, "source.md"))
	diags := s.graphDiagnostics(source)
	actions := s.codeActions(source, s.text(source), diags)
	for _, label := range []string{
		"Create missing heading target",
		"Update link to document heading",
		"Extract link to reference definition",
		"Remove unused reference definitions",
		"Remove duplicate reference definitions",
	} {
		if !hasCodeAction(actions, label) {
			t.Fatalf("missing code action %q: %+v", label, actions)
		}
	}
}

func TestFrontMatterTagsCompletionAndHover(t *testing.T) {
	root := t.TempDir()
	text := "---\ntitle: Catalog\nstatus: current\ndomain: objects\n---\n\n# Catalog\n\n#tag\n"
	writeFile(t, root, "index.md", text)
	s := NewServer(nil, nil)
	s.setRoot(initializeParams{RootURI: uriFromPath(root)})
	index := uriFromPath(filepath.Join(root, "index.md"))
	items := s.completions(index, s.text(index), position{Line: 8, Character: 1})
	if !hasCompletion(items, "#tag") {
		t.Fatalf("tag completion missing: %+v", items)
	}
	hover := s.hoverAt(index, s.text(index), position{Line: 1, Character: 1})
	if !strings.Contains(hover, "title: Catalog") || !strings.Contains(hover, "status: current") {
		t.Fatalf("hover = %q, want front matter", hover)
	}
}
