package lsp

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const maxIndexedFileSize = 1 << 20

type symbolRole int

const (
	symbolDocument symbolRole = iota
	symbolHeading
	symbolSpec
	symbolRequirement
	symbolScenario
	symbolChange
	symbolDelta
	symbolExtension
	symbolObject
	symbolObjectRow
)

type indexedSymbol struct {
	URI       string
	Name      string
	Canon     string
	Norm      string
	Role      symbolRole
	Family    string
	Range     textRange
	Detail    string
	Reference bool
}

type indexedLink struct {
	URI    string
	Range  textRange
	Target linkTarget
}

type indexedDoc struct {
	URI     string
	Path    string
	Text    string
	Symbols []indexedSymbol
	Links   []indexedLink
}

type workspaceIndex struct {
	root  string
	docs  map[string]indexedDoc
	dirty bool
}

func (s *Server) setRoot(p initializeParams) {
	rootURI := p.RootURI
	if rootURI == "" && len(p.WorkspaceFolders) > 0 {
		rootURI = p.WorkspaceFolders[0].URI
	}
	root := ""
	if rootURI != "" {
		root, _ = pathFromURI(rootURI)
	}
	if root == "" {
		root = p.RootPath
	}
	if root == "" {
		return
	}
	s.index.root = filepath.Clean(root)
	s.index.dirty = true
}

func (s *Server) indexDirty(uri string) {
	if s.index.docs != nil {
		s.index.dirty = true
	}
}

func (s *Server) text(uri string) string {
	if text, ok := s.docs[uri]; ok {
		return text
	}
	_ = s.ensureIndex()
	if doc, ok := s.index.docs[uri]; ok {
		return doc.Text
	}
	return ""
}

func (s *Server) ensureIndex() error {
	if s.index.root == "" {
		docs := make(map[string]indexedDoc)
		for uri, text := range s.docs {
			p, _ := pathFromURI(uri)
			docs[uri] = indexDoc(uri, p, text)
		}
		s.index.docs = docs
		s.index.dirty = false
		return nil
	}
	if s.index.docs != nil && !s.index.dirty {
		return nil
	}
	docs := make(map[string]indexedDoc)
	filepath.WalkDir(s.index.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDir(d.Name()) && p != s.index.root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxIndexedFileSize {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil || !utf8.Valid(b) || hasNUL(b) {
			return nil
		}
		uri := uriFromPath(p)
		text := string(b)
		docs[uri] = indexDoc(uri, p, text)
		return nil
	})
	for uri, text := range s.docs {
		p, _ := pathFromURI(uri)
		docs[uri] = indexDoc(uri, p, text)
	}
	s.index.docs = docs
	s.index.dirty = false
	return nil
}

func skipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "target", "out", "dist", "build", ".zed":
		return true
	default:
		return strings.HasPrefix(name, ".cache")
	}
}

func hasNUL(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return true
		}
	}
	return false
}

func pathFromURI(uri string) (string, bool) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return "", false
	}
	return u.Path, true
}

func uriFromPath(p string) string {
	return (&url.URL{Scheme: "file", Path: filepath.Clean(p)}).String()
}

func indexDoc(uri, p, text string) indexedDoc {
	doc := indexedDoc{URI: uri, Path: p, Text: text}
	doc.Symbols = append(doc.Symbols, documentSymbolFor(uri, text))
	for _, h := range headings(text) {
		role := symbolHeadingRole(uri, h.Text)
		r := textRange{Start: position{Line: h.Line, Character: 0}, End: position{Line: h.Line, Character: h.End}}
		name := canonicalName(h.Text)
		doc.Symbols = append(doc.Symbols, indexedSymbol{URI: uri, Name: h.Text, Canon: name, Norm: normName(name), Role: role, Family: artifactFamily(uri), Range: r})
		if objectDetailHeading(uri, h) {
			doc.Symbols = append(doc.Symbols, indexedSymbol{URI: uri, Name: name, Canon: name, Norm: normName(name), Role: symbolObject, Family: "ooux", Range: r})
		}
	}
	doc.Symbols = append(doc.Symbols, objectRows(uri, text)...)
	doc.Symbols = append(doc.Symbols, structuredMentions(uri, text)...)
	for _, link := range wikiLinks(text) {
		doc.Links = append(doc.Links, indexedLink{URI: uri, Range: link.Range, Target: link.Target})
	}
	for _, link := range markdownLinks(text) {
		doc.Links = append(doc.Links, indexedLink{URI: uri, Range: link.Range, Target: link.Target})
	}
	for _, link := range pathLinks(text) {
		doc.Links = append(doc.Links, indexedLink{URI: uri, Range: link.Range, Target: parseLinkTarget(link.Target)})
	}
	return doc
}

func documentSymbolFor(uri, text string) indexedSymbol {
	name := strings.TrimSuffix(filepath.Base(uri), ".md")
	if h, ok := titleHeading(text); ok {
		name = h.Text
	}
	return indexedSymbol{URI: uri, Name: name, Canon: canonicalName(name), Norm: normName(name), Role: documentRole(uri), Family: artifactFamily(uri), Range: fileRange(text)}
}

func documentRole(uri string) symbolRole {
	switch {
	case strings.Contains(uri, "/openspec/specs/"):
		return symbolSpec
	case strings.Contains(uri, "/openspec/changes/") && strings.HasSuffix(uri, "/proposal.md"):
		return symbolChange
	case strings.Contains(uri, "/openspec/changes/") && strings.Contains(uri, "/specs/"):
		return symbolDelta
	case extensionName(uri) != "":
		return symbolExtension
	default:
		return symbolDocument
	}
}

func symbolHeadingRole(uri, name string) symbolRole {
	switch {
	case strings.HasPrefix(strings.ToLower(name), "requirement:"):
		return symbolRequirement
	case strings.HasPrefix(strings.ToLower(name), "scenario:"):
		return symbolScenario
	case extensionName(uri) != "":
		return symbolExtension
	default:
		return symbolHeading
	}
}

func artifactFamily(uri string) string {
	switch {
	case strings.Contains(uri, "/openspec/specs/"):
		return "spec"
	case strings.Contains(uri, "/openspec/changes/"):
		return "change"
	case extensionName(uri) != "":
		return extensionName(uri)
	case strings.Contains(uri, "/objects/") || strings.HasSuffix(uri, "/00-object-catalog.md"):
		return "ooux"
	default:
		return ""
	}
}

func objectDetailHeading(uri string, h heading) bool {
	return h.Level == 2 && strings.Contains(uri, "/objects/") && !strings.HasSuffix(uri, "/README.md")
}

func objectRows(uri, text string) []indexedSymbol {
	if !strings.HasSuffix(uri, "/00-object-catalog.md") {
		return nil
	}
	var out []indexedSymbol
	inObjects := false
	for lineNo, line := range strings.Split(text, "\n") {
		cells := tableCells(line)
		if len(cells) < 2 {
			continue
		}
		if strings.EqualFold(canonicalName(cells[0]), "object") {
			inObjects = true
			continue
		}
		if !inObjects {
			continue
		}
		name := canonicalName(cells[0])
		if name == "" || strings.Contains(name, "---") {
			continue
		}
		start := strings.Index(line, cells[0])
		if start < 0 {
			start = strings.Index(line, name)
		}
		if start < 0 {
			start = 0
		}
		detail := strings.TrimSpace(stripMarkdown(cells[1]))
		out = append(out, indexedSymbol{
			URI: uri, Name: name, Canon: name, Norm: normName(name), Role: symbolObjectRow, Family: "ooux", Detail: detail,
			Range: textRange{Start: position{Line: lineNo, Character: utf16Len(line[:start])}, End: position{Line: lineNo, Character: utf16Len(line[:start+len(cells[0])])}},
		})
	}
	return out
}

func structuredMentions(uri, text string) []indexedSymbol {
	var out []indexedSymbol
	for lineNo, line := range strings.Split(text, "\n") {
		cells := tableCells(line)
		if len(cells) > 0 && !strings.HasSuffix(uri, "/00-object-catalog.md") && !strings.Contains(cells[0], "---") && !strings.EqualFold(canonicalName(cells[0]), "object") {
			name := canonicalName(cells[0])
			if name != "" {
				start := strings.Index(line, cells[0])
				if start < 0 {
					start = 0
				}
				out = append(out, indexedSymbol{URI: uri, Name: name, Canon: name, Norm: normName(name), Role: symbolObject, Family: "ooux", Reference: true, Range: textRange{Start: position{Line: lineNo, Character: utf16Len(line[:start])}, End: position{Line: lineNo, Character: utf16Len(line[:start+len(cells[0])])}}})
			}
			continue
		}
		if name, start, ok := mermaidName(line); ok {
			out = append(out, indexedSymbol{URI: uri, Name: name, Canon: name, Norm: normName(name), Role: symbolObject, Family: "ooux", Reference: true, Range: textRange{Start: position{Line: lineNo, Character: utf16Len(line[:start])}, End: position{Line: lineNo, Character: utf16Len(line[:start+len(name)])}}})
		}
	}
	return out
}

func tableCells(line string) []string {
	trim := strings.TrimSpace(line)
	if !strings.HasPrefix(trim, "|") || !strings.HasSuffix(trim, "|") {
		return nil
	}
	raw := strings.Split(strings.Trim(trim, "|"), "|")
	cells := make([]string, 0, len(raw))
	for _, cell := range raw {
		cells = append(cells, strings.TrimSpace(cell))
	}
	return cells
}

func mermaidName(line string) (string, int, bool) {
	trim := strings.TrimLeft(line, " \t")
	start := len(line) - len(trim)
	if trim == "" || strings.HasPrefix(trim, "flowchart ") || strings.HasPrefix(trim, "```") {
		return "", 0, false
	}
	i := strings.Index(trim, "-->")
	if i < 0 {
		return "", 0, false
	}
	left := strings.TrimSpace(trim[:i])
	if j := strings.IndexAny(left, "[("); j >= 0 {
		left = left[:j]
	}
	left = canonicalName(left)
	return left, start, left != ""
}

func stripMarkdown(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "`", "")
	if i := strings.Index(s, "*("); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func (s *Server) indexedDocs() []indexedDoc {
	_ = s.ensureIndex()
	docs := make([]indexedDoc, 0, len(s.index.docs))
	for _, doc := range s.index.docs {
		docs = append(docs, doc)
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].URI < docs[j].URI })
	return docs
}

func (s *Server) indexedSymbols() []indexedSymbol {
	var syms []indexedSymbol
	for _, doc := range s.indexedDocs() {
		syms = append(syms, doc.Symbols...)
	}
	sort.SliceStable(syms, func(i, j int) bool {
		if syms[i].URI != syms[j].URI {
			return syms[i].URI < syms[j].URI
		}
		if syms[i].Range.Start.Line != syms[j].Range.Start.Line {
			return syms[i].Range.Start.Line < syms[j].Range.Start.Line
		}
		if syms[i].Range.Start.Character != syms[j].Range.Start.Character {
			return syms[i].Range.Start.Character < syms[j].Range.Start.Character
		}
		return syms[i].Role < syms[j].Role
	})
	return syms
}

func (s *Server) graphDiagnostics(uri string) []diagnostic {
	_ = s.ensureIndex()
	doc, ok := s.index.docs[uri]
	if !ok {
		return nil
	}
	var out []diagnostic
	for _, link := range doc.Links {
		if link.Target.Doc != "" && !strings.HasSuffix(strings.ToLower(link.Target.Doc), ".md") {
			if len(s.resolveDocMatches(link.Target.Doc)) > 1 {
				out = append(out, diag(link.Range.Start.Line, link.Range.Start.Character, 2, "ambiguous markdown link target", "link-ambiguous"))
			}
			continue
		}
		loc, ok := s.resolveLink(uri, link.Target)
		if !ok || loc.URI == "" || s.text(loc.URI) == "" {
			out = append(out, diag(link.Range.Start.Line, link.Range.Start.Character, 2, "broken local markdown link", "link"))
			continue
		}
		if link.Target.Heading != "" && loc.Range == (textRange{}) {
			out = append(out, diag(link.Range.Start.Line, link.Range.Start.Character, 2, "missing markdown heading target", "link-heading"))
		}
	}
	if !strings.HasSuffix(uri, "/00-object-catalog.md") && !strings.Contains(uri, "/objects/") {
		return sortDiagnostics(out)
	}
	for _, d := range s.duplicateObjectDiagnostics(uri) {
		out = append(out, d)
	}
	for _, d := range s.catalogObjectDiagnostics(uri) {
		out = append(out, d)
	}
	return sortDiagnostics(out)
}

func (s *Server) resolveDocMatches(name string) []string {
	want := normName(strings.TrimSuffix(name, ".md"))
	var matches []string
	for _, doc := range s.indexedDocs() {
		uri, text := doc.URI, doc.Text
		base := strings.TrimSuffix(filepath.Base(uri), ".md")
		switch {
		case normName(base) == want:
			matches = append(matches, uri)
		default:
			if h, ok := titleHeading(text); ok && normName(h.Text) == want {
				matches = append(matches, uri)
			}
		}
	}
	sort.Strings(matches)
	return matches
}

func (s *Server) duplicateObjectDiagnostics(uri string) []diagnostic {
	var out []diagnostic
	seen := make(map[string]indexedSymbol)
	for _, sym := range s.indexedSymbols() {
		if sym.Role != symbolObject || sym.Reference {
			continue
		}
		if prev, ok := seen[sym.Norm]; ok {
			if prev.URI == uri {
				out = append(out, diag(prev.Range.Start.Line, prev.Range.Start.Character, 2, "duplicate canonical ooux object name", "object-duplicate"))
			}
			if sym.URI == uri {
				out = append(out, diag(sym.Range.Start.Line, sym.Range.Start.Character, 2, "duplicate canonical ooux object name", "object-duplicate"))
			}
			continue
		}
		seen[sym.Norm] = sym
	}
	return out
}

func (s *Server) catalogObjectDiagnostics(uri string) []diagnostic {
	var out []diagnostic
	rows := make(map[string]indexedSymbol)
	objects := make(map[string]indexedSymbol)
	for _, sym := range s.indexedSymbols() {
		switch {
		case sym.Role == symbolObjectRow:
			rows[sym.Norm] = sym
		case sym.Role == symbolObject && !sym.Reference:
			objects[sym.Norm] = sym
		}
	}
	for norm, row := range rows {
		if _, ok := objects[norm]; !ok && row.URI == uri {
			out = append(out, diag(row.Range.Start.Line, row.Range.Start.Character, 2, "catalog object has no matching object detail heading", "object-missing-detail"))
		}
	}
	for norm, obj := range objects {
		if _, ok := rows[norm]; !ok && obj.URI == uri {
			out = append(out, diag(obj.Range.Start.Line, obj.Range.Start.Character, 2, "object detail heading has no catalog row", "object-missing-catalog"))
		}
	}
	return out
}

func sortDiagnostics(diags []diagnostic) []diagnostic {
	sort.Slice(diags, func(i, j int) bool {
		if diags[i].Range.Start.Line != diags[j].Range.Start.Line {
			return diags[i].Range.Start.Line < diags[j].Range.Start.Line
		}
		if diags[i].Range.Start.Character != diags[j].Range.Start.Character {
			return diags[i].Range.Start.Character < diags[j].Range.Start.Character
		}
		return diags[i].Message < diags[j].Message
	})
	return diags
}
