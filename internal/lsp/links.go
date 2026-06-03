package lsp

import (
	"path"
	"strings"
	"unicode"
)

type wikiLink struct {
	URI    string
	Range  range_
	Target linkTarget
}

type linkTarget struct {
	Doc     string
	Heading string
}

func (s *Server) definitions(uri string, pos position) []location {
	link, ok := linkAt(s.docs[uri], pos)
	if !ok {
		return nil
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
		return nil
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

func (s *Server) targetAt(uri string, pos position) (location, bool) {
	if link, ok := linkAt(s.docs[uri], pos); ok {
		return s.resolveLink(uri, link.Target)
	}
	for _, h := range headings(s.docs[uri]) {
		if h.Line == pos.Line {
			r := range_{Start: position{Line: h.Line, Character: 0}, End: position{Line: h.Line, Character: h.Level + 1 + len(h.Text)}}
			return location{URI: uri, Range: r}, true
		}
	}
	return location{}, false
}

func (s *Server) resolveLink(fromURI string, target linkTarget) (location, bool) {
	docURI := fromURI
	if target.Doc != "" {
		var ok bool
		docURI, ok = s.resolveDoc(target.Doc)
		if !ok {
			return location{}, false
		}
	}
	text := s.docs[docURI]
	if target.Heading != "" {
		for _, h := range headings(text) {
			if sameName(h.Text, target.Heading) {
				r := range_{Start: position{Line: h.Line, Character: 0}, End: position{Line: h.Line, Character: h.Level + 1 + len(h.Text)}}
				return location{URI: docURI, Range: r}, true
			}
		}
		return location{}, false
	}
	if h, ok := titleHeading(text); ok {
		r := range_{Start: position{Line: h.Line, Character: 0}, End: position{Line: h.Line, Character: h.Level + 1 + len(h.Text)}}
		return location{URI: docURI, Range: r}, true
	}
	return location{URI: docURI, Range: range_{}}, true
}

func (s *Server) resolveDoc(name string) (string, bool) {
	want := normName(strings.TrimSuffix(name, ".md"))
	for uri, text := range s.docs {
		if h, ok := titleHeading(text); ok && normName(h.Text) == want {
			return uri, true
		}
		base := strings.TrimSuffix(path.Base(uri), ".md")
		if normName(base) == want {
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
	for uri, text := range s.docs {
		for _, link := range wikiLinks(text) {
			link.URI = uri
			links = append(links, link)
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
					Range:  range_{Start: position{Line: lineNo, Character: i}, End: position{Line: lineNo, Character: j + 2}},
					Target: target,
				})
			}
			start = j + 2
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

func sameTarget(a, b location) bool {
	if a.URI != b.URI {
		return false
	}
	return a.Range.Start.Line == b.Range.Start.Line && a.Range.Start.Character == b.Range.Start.Character
}

func sameName(a, b string) bool {
	return normName(a) == normName(b)
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
