package lsp

import "testing"

func TestDefinitionsForWikiLinks(t *testing.T) {
	s := NewServer(nil, nil)
	s.docs = map[string]string{
		"file:///repo/openspec/extensions/ooux/model.md":   "# OOUX model\n\n## Objects\n\n### Account\n\nSee [[Context map#Relationships|context relationships]].\n",
		"file:///repo/openspec/extensions/contexts/map.md": "# Context map\n\n## Contexts\n\n## Relationships\n",
	}
	locs := s.definitions("file:///repo/openspec/extensions/ooux/model.md", position{Line: 6, Character: 8})
	if len(locs) != 1 {
		t.Fatalf("len(definitions) = %d, want 1: %+v", len(locs), locs)
	}
	if got, want := locs[0].URI, "file:///repo/openspec/extensions/contexts/map.md"; got != want {
		t.Fatalf("URI = %q, want %q", got, want)
	}
	if got, want := locs[0].Range.Start.Line, 4; got != want {
		t.Fatalf("line = %d, want %d", got, want)
	}
}

func TestReferencesForWikiLinks(t *testing.T) {
	s := NewServer(nil, nil)
	s.docs = map[string]string{
		"file:///repo/openspec/extensions/ooux/model.md":       "# OOUX model\n\nSee [[Context map]].\n",
		"file:///repo/openspec/extensions/contexts/map.md":     "# Context map\n\n## Relationships\n",
		"file:///repo/openspec/extensions/eventstorm/model.md": "# Event storm\n\nSee [[Context map#Relationships]].\n",
	}
	locs := s.references("file:///repo/openspec/extensions/contexts/map.md", position{Line: 0, Character: 2})
	if len(locs) != 1 {
		t.Fatalf("len(references title) = %d, want 1: %+v", len(locs), locs)
	}
	if got, want := locs[0].URI, "file:///repo/openspec/extensions/ooux/model.md"; got != want {
		t.Fatalf("URI = %q, want %q", got, want)
	}
	locs = s.references("file:///repo/openspec/extensions/contexts/map.md", position{Line: 2, Character: 3})
	if len(locs) != 1 {
		t.Fatalf("len(references heading) = %d, want 1: %+v", len(locs), locs)
	}
	if got, want := locs[0].URI, "file:///repo/openspec/extensions/eventstorm/model.md"; got != want {
		t.Fatalf("URI = %q, want %q", got, want)
	}
}

func TestWikiLinksParseAliasesAndLocalHeadings(t *testing.T) {
	links := wikiLinks("See [[#Objects]] and [[OOUX model|model]].\n")
	if len(links) != 2 {
		t.Fatalf("len(wikiLinks) = %d, want 2: %+v", len(links), links)
	}
	if links[0].Target.Heading != "Objects" || links[0].Target.Doc != "" {
		t.Fatalf("first target = %+v, want local Objects", links[0].Target)
	}
	if links[1].Target.Doc != "OOUX model" {
		t.Fatalf("second doc = %q, want OOUX model", links[1].Target.Doc)
	}
}

func TestWikiLinkRangesUseUTF16(t *testing.T) {
	links := wikiLinks("🔐 see [[Context map]].\n")
	if len(links) != 1 {
		t.Fatalf("len(wikiLinks) = %d, want 1", len(links))
	}
	if got, want := links[0].Range.Start.Character, 7; got != want {
		t.Fatalf("Start.Character = %d, want %d", got, want)
	}
}

func TestMarkdownLinksParseRelativeTargets(t *testing.T) {
	links := markdownLinks("[catalog](../00-object-catalog.md#cross-tier-object-map)\n")
	if len(links) != 1 {
		t.Fatalf("len(markdownLinks) = %d, want 1", len(links))
	}
	if got, want := links[0].Target.Doc, "../00-object-catalog.md"; got != want {
		t.Fatalf("Doc = %q, want %q", got, want)
	}
	if got, want := links[0].Target.Heading, "cross-tier-object-map"; got != want {
		t.Fatalf("Heading = %q, want %q", got, want)
	}
}

func TestDefinitionsResolveMarkdownLinksAndObjectNames(t *testing.T) {
	s := NewServer(nil, nil)
	indexURI := "file:///repo/matrices/relationship-map.md"
	catalogURI := "file:///repo/00-object-catalog.md"
	objectURI := "file:///repo/objects/t1-conversation-and-reasoning.md"
	s.docs[indexURI] = "See [map](../00-object-catalog.md#cross-tier-object-map).\n\n| Thread | Message |\n"
	s.docs[catalogURI] = "# Catalog\n\n## Cross-tier Object Map\n"
	s.docs[objectURI] = "# T1\n\n## Thread   *(status: current)*\n"

	locs := s.definitions(indexURI, position{Line: 0, Character: 6})
	if len(locs) != 1 {
		t.Fatalf("len(markdown link definitions) = %d, want 1: %+v", len(locs), locs)
	}
	if got, want := locs[0].URI, catalogURI; got != want {
		t.Fatalf("URI = %q, want %q", got, want)
	}
	if got, want := locs[0].Range.Start.Line, 2; got != want {
		t.Fatalf("Range.Start.Line = %d, want %d", got, want)
	}

	locs = s.definitions(indexURI, position{Line: 2, Character: 3})
	if len(locs) != 1 {
		t.Fatalf("len(object name definitions) = %d, want 1: %+v", len(locs), locs)
	}
	if got, want := locs[0].URI, objectURI; got != want {
		t.Fatalf("URI = %q, want %q", got, want)
	}
}

func TestReferencesResolveObjectNames(t *testing.T) {
	s := NewServer(nil, nil)
	indexURI := "file:///repo/matrices/relationship-map.md"
	objectURI := "file:///repo/objects/t1-conversation-and-reasoning.md"
	s.docs[indexURI] = "| Thread | Message |\n| Thread | Run |\n"
	s.docs[objectURI] = "# T1\n\n## Thread   *(status: current)*\n"

	locs := s.references(indexURI, position{Line: 0, Character: 3})
	if len(locs) != 3 {
		t.Fatalf("len(references) = %d, want 3: %+v", len(locs), locs)
	}
	var sawDecl bool
	for _, loc := range locs {
		if loc.URI == objectURI && loc.Range.Start.Line == 2 {
			sawDecl = true
		}
	}
	if !sawDecl {
		t.Fatalf("references did not include object declaration: %+v", locs)
	}
}
