package lsp

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

type wikiLink struct {
	URI    string
	Range  textRange
	Target linkTarget
}

type markdownLink struct {
	URI    string
	Range  textRange
	Target linkTarget
}

type referenceDefinition struct {
	Label  string
	Range  textRange
	Target linkTarget
}

type linkTarget struct {
	Doc     string
	Heading string
}

func (s *Server) definitions(uri string, pos position) []location {
	text := s.text(uri)
	link, ok := linkAt(text, pos)
	if !ok {
		target, ok := plainNameAt(text, pos)
		if !ok {
			return nil
		}
		loc, ok := s.resolveName(target)
		if !ok {
			return nil
		}
		return []location{loc}
	}
	loc, ok := s.resolveLink(uri, link.Target)
	if !ok {
		return nil
	}
	return []location{loc}
}

func (s *Server) references(uri string, pos position) []location {
	target, ok := s.targetAt(uri, pos)
	if !ok {
		name, ok := plainNameAt(s.text(uri), pos)
		if !ok {
			return nil
		}
		target, ok = s.resolveName(name)
		if !ok {
			return nil
		}
		return s.nameReferences(name, target)
	}
	var locs []location
	for _, link := range s.allLinks() {
		loc, ok := s.resolveLink(link.URI, link.Target)
		if !ok || !sameTarget(loc, target) {
			continue
		}
		locs = append(locs, location{URI: link.URI, Range: link.Range})
	}
	return locs
}

func (s *Server) nameReferences(name string, target location) []location {
	var locs []location
	norm := normName(canonicalName(name))
	for _, sym := range s.indexedSymbols() {
		if sym.Norm == norm && (sym.Role == symbolObject || sym.Role == symbolObjectRow) {
			locs = append(locs, location{URI: sym.URI, Range: sym.Range})
		}
	}
	if len(locs) == 0 {
		locs = append(locs, target)
	}
	return locs
}

func (s *Server) targetAt(uri string, pos position) (location, bool) {
	text := s.text(uri)
	if link, ok := linkAt(text, pos); ok {
		return s.resolveLink(uri, link.Target)
	}
	for _, h := range headings(text) {
		if h.Line == pos.Line {
			r := textRange{Start: position{Line: h.Line, Character: 0}, End: position{Line: h.Line, Character: h.End}}
			return location{URI: uri, Range: r}, true
		}
	}
	return location{}, false
}

func (s *Server) resolveLink(fromURI string, target linkTarget) (location, bool) {
	docURI := fromURI
	if target.Doc != "" {
		if rel, ok := s.resolveRelativeDoc(fromURI, target.Doc); ok && strings.HasSuffix(strings.ToLower(target.Doc), ".md") {
			docURI = rel
		} else {
			var ok bool
			docURI, ok = s.resolveDoc(target.Doc)
			if !ok {
				return location{}, false
			}
		}
	}
	text := s.docs[docURI]
	if text == "" {
		text = s.text(docURI)
	}
	if target.Heading != "" {
		for _, h := range headings(text) {
			if sameName(h.Text, target.Heading) {
				r := textRange{Start: position{Line: h.Line, Character: 0}, End: position{Line: h.Line, Character: h.End}}
				return location{URI: docURI, Range: r}, true
			}
		}
		return location{URI: docURI, Range: textRange{}}, true
	}
	if h, ok := titleHeading(text); ok {
		r := textRange{Start: position{Line: h.Line, Character: 0}, End: position{Line: h.Line, Character: h.End}}
		return location{URI: docURI, Range: r}, true
	}
	return location{URI: docURI, Range: textRange{}}, true
}

func (s *Server) resolveDoc(name string) (string, bool) {
	want := normName(strings.TrimSuffix(name, ".md"))
	for _, doc := range s.indexedDocs() {
		uri, text := doc.URI, doc.Text
		base := strings.TrimSuffix(path.Base(uri), ".md")
		if normName(base) == want {
			return uri, true
		}
		if h, ok := titleHeading(text); ok && normName(h.Text) == want {
			return uri, true
		}
		noExt := strings.TrimSuffix(uri, ".md")
		if strings.HasSuffix(normName(noExt), want) {
			return uri, true
		}
	}
	return "", false
}

func (s *Server) allLinks() []wikiLink {
	var links []wikiLink
	for _, doc := range s.indexedDocs() {
		for _, link := range doc.Links {
			links = append(links, wikiLink{URI: link.URI, Range: link.Range, Target: link.Target})
		}
	}
	return links
}

func linkAt(text string, pos position) (wikiLink, bool) {
	for _, link := range wikiLinks(text) {
		if pos.Line != link.Range.Start.Line {
			continue
		}
		if pos.Character >= link.Range.Start.Character && pos.Character <= link.Range.End.Character {
			return link, true
		}
	}
	for _, link := range markdownLinks(text) {
		if pos.Line != link.Range.Start.Line {
			continue
		}
		if pos.Character >= link.Range.Start.Character && pos.Character <= link.Range.End.Character {
			return wikiLink{Range: link.Range, Target: link.Target}, true
		}
	}
	defs := referenceDefinitions(text)
	for _, link := range referenceLinks(text, defs) {
		if pos.Line != link.Range.Start.Line {
			continue
		}
		if pos.Character >= link.Range.Start.Character && pos.Character <= link.Range.End.Character {
			return wikiLink{Range: link.Range, Target: link.Target}, true
		}
	}
	return wikiLink{}, false
}

func wikiLinks(text string) []wikiLink {
	var links []wikiLink
	for lineNo, line := range strings.Split(text, "\n") {
		start := 0
		for {
			i := strings.Index(line[start:], "[[")
			if i < 0 {
				break
			}
			i += start
			j := strings.Index(line[i+2:], "]]")
			if j < 0 {
				break
			}
			j += i + 2
			raw := line[i+2 : j]
			target := parseLinkTarget(raw)
			if target.Doc != "" || target.Heading != "" {
				links = append(links, wikiLink{
					Range:  textRange{Start: position{Line: lineNo, Character: utf16Len(line[:i])}, End: position{Line: lineNo, Character: utf16Len(line[:j+2])}},
					Target: target,
				})
			}
			start = j + 2
		}
	}
	return links
}

func markdownLinks(text string) []markdownLink {
	var links []markdownLink
	for lineNo, line := range strings.Split(text, "\n") {
		start := 0
		for {
			openLabel := strings.Index(line[start:], "[")
			if openLabel < 0 {
				break
			}
			openLabel += start
			closeLabel := strings.Index(line[openLabel+1:], "](")
			if closeLabel < 0 {
				break
			}
			closeLabel += openLabel + 1
			openTarget := closeLabel + 2
			closeTarget := strings.Index(line[openTarget:], ")")
			if closeTarget < 0 {
				break
			}
			closeTarget += openTarget
			raw := strings.TrimSpace(line[openTarget:closeTarget])
			if raw != "" && !strings.Contains(raw, "://") && !strings.HasPrefix(raw, "mailto:") {
				target := parseLinkTarget(raw)
				links = append(links, markdownLink{
					Range:  textRange{Start: position{Line: lineNo, Character: utf16Len(line[:openLabel])}, End: position{Line: lineNo, Character: utf16Len(line[:closeTarget+1])}},
					Target: target,
				})
			}
			start = closeTarget + 1
		}
	}
	return links
}

func referenceDefinitions(text string) map[string]referenceDefinition {
	defs := make(map[string]referenceDefinition)
	for lineNo, line := range strings.Split(text, "\n") {
		trim := strings.TrimLeft(line, " \t")
		indent := len(line) - len(trim)
		if !strings.HasPrefix(trim, "[") {
			continue
		}
		closeLabel := strings.Index(trim, "]:")
		if closeLabel <= 1 {
			continue
		}
		label := strings.TrimSpace(trim[1:closeLabel])
		raw := strings.TrimSpace(trim[closeLabel+2:])
		raw = strings.Trim(raw, "<>")
		if label == "" || raw == "" || strings.Contains(raw, "://") || strings.HasPrefix(raw, "mailto:") {
			continue
		}
		defs[normName(label)] = referenceDefinition{
			Label:  label,
			Range:  textRange{Start: position{Line: lineNo, Character: utf16Len(line[:indent])}, End: position{Line: lineNo, Character: utf16Len(line)}},
			Target: parseLinkTarget(raw),
		}
	}
	return defs
}

func referenceLinks(text string, defs map[string]referenceDefinition) []markdownLink {
	var links []markdownLink
	for lineNo, line := range strings.Split(text, "\n") {
		start := 0
		for {
			openText := strings.Index(line[start:], "[")
			if openText < 0 {
				break
			}
			openText += start
			if openText+1 < len(line) && line[openText+1] == '[' {
				start = openText + 2
				continue
			}
			closeText := strings.Index(line[openText+1:], "][")
			if closeText < 0 {
				break
			}
			closeText += openText + 1
			openLabel := closeText + 2
			closeLabel := strings.Index(line[openLabel:], "]")
			if closeLabel < 0 {
				break
			}
			closeLabel += openLabel
			label := strings.TrimSpace(line[openLabel:closeLabel])
			if label == "" {
				label = strings.TrimSpace(line[openText+1 : closeText])
			}
			if def, ok := defs[normName(label)]; ok {
				links = append(links, markdownLink{
					Range:  textRange{Start: position{Line: lineNo, Character: utf16Len(line[:openText])}, End: position{Line: lineNo, Character: utf16Len(line[:closeLabel+1])}},
					Target: def.Target,
				})
			}
			start = closeLabel + 1
		}
	}
	return links
}

func parseLinkTarget(raw string) linkTarget {
	raw, _, _ = strings.Cut(raw, "|")
	raw = strings.TrimSpace(raw)
	doc, heading, ok := strings.Cut(raw, "#")
	if !ok {
		return linkTarget{Doc: strings.TrimSpace(raw)}
	}
	return linkTarget{Doc: strings.TrimSpace(doc), Heading: strings.TrimSpace(heading)}
}

func titleHeading(text string) (heading, bool) {
	for _, h := range headings(text) {
		if h.Level == 1 {
			return h, true
		}
	}
	return heading{}, false
}

func (s *Server) resolveName(name string) (location, bool) {
	norm := normName(canonicalName(name))
	for _, role := range []symbolRole{symbolObject, symbolObjectRow, symbolHeading} {
		for _, sym := range s.indexedSymbols() {
			if sym.Reference || sym.Norm != norm || sym.Role != role {
				continue
			}
			return location{URI: sym.URI, Range: sym.Range}, true
		}
	}
	return location{}, false
}

func plainNameAt(text string, pos position) (string, bool) {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return "", false
	}
	line := lines[pos.Line]
	bytePos := byteOffsetForUTF16(line, pos.Character)
	if bytePos < 0 || bytePos > len(line) {
		return "", false
	}
	start := bytePos
	for start > 0 && nameByte(line[start-1]) {
		start--
	}
	end := bytePos
	for end < len(line) && nameByte(line[end]) {
		end++
	}
	name := strings.Trim(line[start:end], " \t|*`")
	if name == "" {
		return "", false
	}
	return name, true
}

func nameRanges(text, name string) []textRange {
	var ranges []textRange
	for lineNo, line := range strings.Split(text, "\n") {
		for start := 0; start < len(line); {
			for start < len(line) && !nameByte(line[start]) {
				start++
			}
			end := start
			for end < len(line) && nameByte(line[end]) {
				end++
			}
			got := strings.Trim(line[start:end], " \t|*`")
			if got != "" && sameName(got, name) {
				ranges = append(ranges, textRange{
					Start: position{Line: lineNo, Character: utf16Len(line[:start])},
					End:   position{Line: lineNo, Character: utf16Len(line[:end])},
				})
			}
			if end == start {
				start++
			} else {
				start = end
			}
		}
	}
	return ranges
}

func nameByte(b byte) bool {
	return b == ' ' || b == '/' || b == '-' || b == '_' || b == '&' || b == '\'' || b == '`' || b == '*' || b == ':' || b == '(' || b == ')' || unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b))
}

func byteOffsetForUTF16(s string, want int) int {
	u16 := 0
	for i, r := range s {
		if u16 >= want {
			return i
		}
		if r > 0xFFFF {
			u16 += 2
		} else {
			u16++
		}
	}
	if u16 == want {
		return len(s)
	}
	return -1
}

func sameTarget(a, b location) bool {
	if a.URI != b.URI {
		return false
	}
	return a.Range.Start.Line == b.Range.Start.Line && a.Range.Start.Character == b.Range.Start.Character
}

func sameName(a, b string) bool {
	return normName(canonicalName(a)) == normName(canonicalName(b))
}

func normName(s string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
		case r == '/' || r == '-' || r == '_' || unicode.IsSpace(r):
			space = true
		}
	}
	return b.String()
}

func canonicalName(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "*("); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, "`status:"); i >= 0 {
		s = s[:i]
	}
	s = strings.Trim(s, " \t`*")
	return s
}

func (s *Server) resolveRelativeDoc(fromURI, doc string) (string, bool) {
	if doc == "" {
		return fromURI, true
	}
	if strings.HasPrefix(doc, "file://") {
		return doc, true
	}
	if strings.HasPrefix(doc, "/") {
		return "file://" + doc, true
	}
	u, err := url.Parse(fromURI)
	if err != nil || u.Scheme != "file" {
		return "", false
	}
	base := filepath.Dir(u.Path)
	target := filepath.Clean(filepath.Join(base, doc))
	return (&url.URL{Scheme: "file", Path: target}).String(), true
}
