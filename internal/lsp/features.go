package lsp

import (
	"sort"
	"strings"
)

const (
	symbolKindFile      = 1
	symbolKindNamespace = 3
	symbolKindClass     = 5
	symbolKindMethod    = 6
	symbolKindString    = 15
)

func codeActions(uri, text string, diags []diagnostic) []codeAction {
	var actions []codeAction
	seen := presentSections(headings(text))
	for _, section := range requiredSections(uri) {
		if !seen[strings.ToLower(section)] {
			title := "Insert ## " + section
			actions = append(actions, editAction(title, uri, appendSectionEdit(text, section), matchingDiagnostics(diags, "missing ## "+section+" section")))
		}
	}
	for _, h := range headings(text) {
		if strings.EqualFold(h.Text, "Requiement") {
			actions = append(actions, editAction("Fix heading to ## Requirements", uri, textEdit{
				Range:   textRange{Start: position{Line: h.Line, Character: 0}, End: position{Line: h.Line, Character: h.End}},
				NewText: "## Requirements",
			}, nil))
		}
	}
	if strings.Contains(uri, "/openspec/specs/") || strings.Contains(uri, "/openspec/changes/") {
		actions = append(actions, editAction("Insert requirement scenario skeleton", uri, insertAtEnd(text, "\n### Requirement: name\n\nThe system SHALL ...\n\n#### Scenario: name\n\n- GIVEN context\n- WHEN action\n- THEN outcome\n"), nil))
	}
	return actions
}

func editAction(title, uri string, edit textEdit, diags []diagnostic) codeAction {
	return codeAction{
		Title:       title,
		Kind:        "quickfix",
		Diagnostics: diags,
		Edit:        workspaceEdit{Changes: map[string][]textEdit{uri: []textEdit{edit}}},
	}
}

func matchingDiagnostics(diags []diagnostic, msg string) []diagnostic {
	var out []diagnostic
	for _, diag := range diags {
		if diag.Message == msg {
			out = append(out, diag)
		}
	}
	return out
}

func appendSectionEdit(text, section string) textEdit {
	insert := "\n\n## " + section + "\n"
	if strings.TrimSpace(text) == "" {
		insert = "## " + section + "\n"
	}
	return insertAtEnd(text, insert)
}

func insertAtEnd(text, insert string) textEdit {
	lines := strings.Split(text, "\n")
	line := len(lines) - 1
	char := utf16Len(lines[line])
	return textEdit{Range: textRange{Start: position{Line: line, Character: char}, End: position{Line: line, Character: char}}, NewText: insert}
}

func (s *Server) documentLinks(uri string) []documentLink {
	var links []documentLink
	text := s.text(uri)
	for _, link := range wikiLinks(text) {
		docURI := ""
		if loc, ok := s.resolveLink(uri, link.Target); ok {
			docURI = loc.URI
		}
		links = append(links, documentLink{Range: link.Range, Target: docURI})
	}
	for _, link := range markdownLinks(text) {
		docURI := ""
		if loc, ok := s.resolveLink(uri, link.Target); ok {
			docURI = loc.URI
		}
		links = append(links, documentLink{Range: link.Range, Target: docURI})
	}
	links = append(links, pathLinks(text)...)
	return links
}

func pathLinks(text string) []documentLink {
	var links []documentLink
	for lineNo, line := range strings.Split(text, "\n") {
		start := 0
		for {
			i := strings.Index(line[start:], "openspec/")
			if i < 0 {
				break
			}
			i += start
			j := i
			for j < len(line) && !strings.ContainsRune(" \t\r\n)]}\"'", rune(line[j])) {
				j++
			}
			if j > i {
				links = append(links, documentLink{
					Range:  textRange{Start: position{Line: lineNo, Character: utf16Len(line[:i])}, End: position{Line: lineNo, Character: utf16Len(line[:j])}},
					Target: line[i:j],
				})
			}
			start = j
		}
	}
	return links
}

func (s *Server) workspaceSymbols(query string) []workspaceSymbol {
	query = strings.ToLower(strings.TrimSpace(query))
	var out []workspaceSymbol
	for _, sym := range s.indexedSymbols() {
		if sym.Reference || !matchSymbol(query, sym.Name) {
			continue
		}
		out = append(out, workspaceSymbol{Name: sym.Name, Kind: symbolKindForRole(sym.Role), Location: location{URI: sym.URI, Range: sym.Range}})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Location.URI != out[j].Location.URI {
			return out[i].Location.URI < out[j].Location.URI
		}
		return out[i].Location.Range.Start.Line < out[j].Location.Range.Start.Line
	})
	return out
}

func symbolKindForRole(role symbolRole) int {
	switch role {
	case symbolDocument, symbolSpec, symbolChange, symbolDelta:
		return symbolKindFile
	case symbolRequirement:
		return symbolKindMethod
	case symbolScenario:
		return symbolKindString
	case symbolObject, symbolObjectRow:
		return symbolKindClass
	case symbolExtension:
		return symbolKindClass
	default:
		return symbolKindNamespace
	}
}

func matchSymbol(query, name string) bool {
	return query == "" || strings.Contains(strings.ToLower(name), query)
}

func headingSymbolKind(uri, name string) int {
	switch {
	case strings.HasPrefix(strings.ToLower(name), "requirement:"):
		return symbolKindMethod
	case strings.HasPrefix(strings.ToLower(name), "scenario:"):
		return symbolKindString
	case extensionName(uri) != "":
		return symbolKindClass
	default:
		return symbolKindNamespace
	}
}

func fileRange(text string) textRange {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return textRange{}
	}
	last := len(lines) - 1
	return textRange{Start: position{}, End: position{Line: last, Character: utf16Len(lines[last])}}
}

func foldingRanges(text string) []foldingRange {
	heads := headings(text)
	var out []foldingRange
	lines := strings.Split(text, "\n")
	for i, h := range heads {
		end := len(lines) - 1
		for j := i + 1; j < len(heads); j++ {
			if heads[j].Level <= h.Level {
				end = heads[j].Line - 1
				break
			}
		}
		if end > h.Line {
			out = append(out, foldingRange{StartLine: h.Line, StartCharacter: h.End, EndLine: end, EndCharacter: utf16Len(lines[end]), Kind: "region"})
		}
	}
	return out
}

func selectionRanges(text string, positions []position) []selectionRange {
	heads := headings(text)
	out := make([]selectionRange, 0, len(positions))
	docRange := fileRange(text)
	for _, pos := range positions {
		var chosen *heading
		for i := range heads {
			h := &heads[i]
			if h.Line <= pos.Line && (chosen == nil || h.Line >= chosen.Line) {
				chosen = h
			}
		}
		if chosen == nil {
			out = append(out, selectionRange{Range: docRange})
			continue
		}
		hr := textRange{Start: position{Line: chosen.Line, Character: 0}, End: position{Line: chosen.Line, Character: chosen.End}}
		out = append(out, selectionRange{Range: hr, Parent: &selectionRange{Range: docRange}})
	}
	return out
}
