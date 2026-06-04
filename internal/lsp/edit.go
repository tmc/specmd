package lsp

import (
	"fmt"
	"sort"
	"strings"
)

func (s *Server) prepareRename(uri string, pos position) any {
	text := s.text(uri)
	if link, ok := linkAt(text, pos); ok {
		return link.Range
	}
	if name, ok := plainNameAt(text, pos); ok {
		if loc, ok := s.resolveName(name); ok {
			return s.editRangeForName(loc.URI, loc.Range, name)
		}
	}
	return nil
}

func (s *Server) rename(uri string, pos position, newName string) workspaceEdit {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return workspaceEdit{}
	}
	text := s.text(uri)
	if link, ok := linkAt(text, pos); ok {
		return s.renameLinkTarget(uri, link, newName)
	}
	name, ok := plainNameAt(text, pos)
	if !ok {
		return workspaceEdit{}
	}
	loc, ok := s.resolveName(name)
	if !ok {
		return workspaceEdit{}
	}
	refs := s.nameReferences(name, loc)
	refs = append(refs, loc)
	return workspaceEdit{Changes: editsForRename(s, refs, name, newName)}
}

func (s *Server) renameLinkTarget(uri string, link wikiLink, newName string) workspaceEdit {
	loc, ok := s.resolveLink(uri, link.Target)
	if !ok {
		return workspaceEdit{}
	}
	refs := s.references(uri, link.Range.Start)
	refs = append(refs, loc)
	return workspaceEdit{Changes: editsForRename(s, refs, canonicalName(link.Target.Heading), newName)}
}

func editsForRename(s *Server, locs []location, oldName, newName string) map[string][]textEdit {
	changes := make(map[string][]textEdit)
	seen := make(map[string]bool)
	for _, loc := range locs {
		r := s.editRangeForName(loc.URI, loc.Range, oldName)
		key := fmt.Sprintf("%s:%d:%d:%d:%d", loc.URI, r.Start.Line, r.Start.Character, r.End.Line, r.End.Character)
		if seen[key] {
			continue
		}
		seen[key] = true
		changes[loc.URI] = append(changes[loc.URI], textEdit{Range: r, NewText: newName})
	}
	for uri := range changes {
		sort.Slice(changes[uri], func(i, j int) bool {
			if changes[uri][i].Range.Start.Line != changes[uri][j].Range.Start.Line {
				return changes[uri][i].Range.Start.Line > changes[uri][j].Range.Start.Line
			}
			return changes[uri][i].Range.Start.Character > changes[uri][j].Range.Start.Character
		})
	}
	return changes
}

func (s *Server) editRangeForName(uri string, r textRange, name string) textRange {
	line := lineAt(s.text(uri), r.Start.Line)
	if line == "" {
		return r
	}
	want := canonicalName(name)
	if want == "" {
		want = name
	}
	if i := strings.Index(line, want); i >= 0 {
		return textRange{
			Start: position{Line: r.Start.Line, Character: utf16Len(line[:i])},
			End:   position{Line: r.Start.Line, Character: utf16Len(line[:i+len(want)])},
		}
	}
	return r
}

func (s *Server) codeLens(uri string) []codeLens {
	var out []codeLens
	for _, sym := range s.indexedSymbols() {
		if sym.URI != uri || sym.Reference {
			continue
		}
		if sym.Role != symbolObject && sym.Role != symbolObjectRow && sym.Role != symbolHeading {
			continue
		}
		n := len(s.references(sym.URI, sym.Range.Start))
		if sym.Role == symbolObject || sym.Role == symbolObjectRow {
			n = len(s.nameReferences(sym.Canon, location{URI: sym.URI, Range: sym.Range}))
		}
		if n == 0 {
			continue
		}
		out = append(out, codeLens{
			Range: sym.Range,
			Command: command{
				Title:   fmt.Sprintf("%d references", n),
				Command: "",
			},
		})
	}
	return out
}

func (s *Server) inlayHints(uri string, r textRange) []inlayHint {
	var out []inlayHint
	for _, lens := range s.codeLens(uri) {
		if lens.Range.Start.Line < r.Start.Line || lens.Range.Start.Line > r.End.Line {
			continue
		}
		out = append(out, inlayHint{Position: lens.Range.End, Label: lens.Command.Title, Kind: 1})
	}
	return out
}

func (s *Server) codeActions(uri, text string, diags []diagnostic) []codeAction {
	actions := codeActions(uri, text, diags)
	actions = append(actions, s.linkCodeActions(uri, text, diags)...)
	actions = append(actions, referenceDefinitionActions(uri, text)...)
	return actions
}

func (s *Server) linkCodeActions(uri, text string, diags []diagnostic) []codeAction {
	var actions []codeAction
	for _, d := range diags {
		if d.Code != "link-heading" {
			continue
		}
		link, ok := linkAt(text, d.Range.Start)
		if !ok || link.Target.Heading == "" {
			continue
		}
		targetURI := uri
		if link.Target.Doc != "" {
			targetURI, _ = s.resolveRelativeDoc(uri, link.Target.Doc)
		}
		if targetURI != "" {
			actions = append(actions, editAction("Create missing heading target", targetURI, insertAtEnd(s.text(targetURI), "\n\n## "+titleCase(link.Target.Heading)+"\n"), []diagnostic{d}))
		}
		if h, ok := titleHeading(s.text(targetURI)); ok {
			actions = append(actions, editAction("Update link to document heading", uri, replaceLinkFragmentEdit(text, link.Range, h.Text), []diagnostic{d}))
		}
	}
	for _, link := range markdownLinks(text) {
		if e, ok := extractReferenceLinkEdit(text, link.Range); ok {
			actions = append(actions, editAction("Extract link to reference definition", uri, e, nil))
			break
		}
	}
	return actions
}

func referenceDefinitionActions(uri, text string) []codeAction {
	var actions []codeAction
	if edit, ok := removeUnusedReferenceDefinitionsEdit(text); ok {
		actions = append(actions, editAction("Remove unused reference definitions", uri, edit, nil))
	}
	if edit, ok := removeDuplicateReferenceDefinitionsEdit(text); ok {
		actions = append(actions, editAction("Remove duplicate reference definitions", uri, edit, nil))
	}
	return actions
}

func replaceLinkFragmentEdit(text string, r textRange, heading string) textEdit {
	lines := strings.Split(text, "\n")
	if r.Start.Line < 0 || r.Start.Line >= len(lines) {
		return textEdit{Range: r}
	}
	line := lines[r.Start.Line]
	start := byteOffsetForUTF16(line, r.Start.Character)
	end := byteOffsetForUTF16(line, r.End.Character)
	if start < 0 || end < start {
		return textEdit{Range: r}
	}
	raw := line[start:end]
	i := strings.Index(raw, "#")
	j := strings.LastIndex(raw, ")")
	if i < 0 || j < i {
		return textEdit{Range: r}
	}
	a := start + i + 1
	b := start + j
	return textEdit{
		Range:   textRange{Start: position{Line: r.Start.Line, Character: utf16Len(line[:a])}, End: position{Line: r.Start.Line, Character: utf16Len(line[:b])}},
		NewText: slug(heading),
	}
}

func extractReferenceLinkEdit(text string, r textRange) (textEdit, bool) {
	lines := strings.Split(text, "\n")
	if r.Start.Line < 0 || r.Start.Line >= len(lines) {
		return textEdit{}, false
	}
	line := lines[r.Start.Line]
	start := byteOffsetForUTF16(line, r.Start.Character)
	end := byteOffsetForUTF16(line, r.End.Character)
	if start < 0 || end < start {
		return textEdit{}, false
	}
	raw := line[start:end]
	openTarget := strings.LastIndex(raw, "](")
	closeTarget := strings.LastIndex(raw, ")")
	if openTarget < 0 || closeTarget < openTarget {
		return textEdit{}, false
	}
	label := strings.TrimSpace(strings.Trim(raw[1:openTarget], "[]"))
	target := strings.TrimSpace(raw[openTarget+2 : closeTarget])
	if label == "" || target == "" {
		return textEdit{}, false
	}
	newText := "[" + label + "][" + label + "]"
	appendText := "\n[" + label + "]: " + target + "\n"
	lines[r.Start.Line] = line[:start] + newText + line[end:]
	return textEdit{Range: fileRange(text), NewText: strings.Join(lines, "\n") + appendText}, true
}

func removeUnusedReferenceDefinitionsEdit(text string) (textEdit, bool) {
	defs := referenceDefinitions(text)
	used := make(map[string]bool)
	for _, link := range referenceLinks(text, defs) {
		for key, def := range defs {
			if def.Target == link.Target {
				used[key] = true
			}
		}
	}
	return removeReferenceDefinitionLines(text, func(key string) bool { return !used[key] })
}

func removeDuplicateReferenceDefinitionsEdit(text string) (textEdit, bool) {
	seen := make(map[string]bool)
	return removeReferenceDefinitionLines(text, func(key string) bool {
		if seen[key] {
			return true
		}
		seen[key] = true
		return false
	})
}

func removeReferenceDefinitionLines(text string, remove func(string) bool) (textEdit, bool) {
	lines := strings.Split(text, "\n")
	var out []string
	changed := false
	for _, line := range lines {
		trim := strings.TrimLeft(line, " \t")
		key := ""
		if strings.HasPrefix(trim, "[") {
			if i := strings.Index(trim, "]:"); i > 1 {
				key = normName(strings.TrimSpace(trim[1:i]))
			}
		}
		if key != "" && remove(key) {
			changed = true
			continue
		}
		out = append(out, line)
	}
	if !changed {
		return textEdit{}, false
	}
	return textEdit{Range: fileRange(text), NewText: strings.Join(out, "\n")}, true
}

func titleCase(slug string) string {
	parts := strings.Fields(strings.ReplaceAll(slug, "-", " "))
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
