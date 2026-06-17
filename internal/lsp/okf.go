package lsp

import (
	"path"
	"strings"

	openspec "github.com/tmc/openspec"
)

// okfConceptID derives a stable concept id from a document URI, mirroring the
// library's path-based id without requiring the file on disk.
func okfConceptID(uri string) string {
	if p, ok := pathFromURI(uri); ok {
		return strings.TrimSuffix(path.Base(p), ".md")
	}
	return strings.TrimSuffix(path.Base(uri), ".md")
}

// okfParse parses an OKF concept from editor text. It canonicalizes the YAML
// fence lines first because the library parser requires bare "---" fences while
// editors commonly leave trailing whitespace on them; the LSP's lenient
// front-matter detection accepts those, so the parser must too.
func okfParse(uri, text string) (*openspec.OKFConcept, error) {
	return openspec.ParseOKFConcept(okfConceptID(uri), strings.NewReader(canonicalFences(text)))
}

// canonicalFences trims trailing whitespace from leading "---" fence lines so a
// front matter block with whitespace-padded fences parses like a bare one.
func canonicalFences(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return text
	}
	lines[0] = "---"
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			lines[i] = "---"
			break
		}
	}
	return strings.Join(lines, "\n")
}

// isOKFReserved reports whether uri names a reserved OKF file (index.md or
// log.md), which carry no concept frontmatter.
func isOKFReserved(uri string) bool {
	switch strings.ToLower(path.Base(uri)) {
	case "index.md", "log.md":
		return true
	default:
		return false
	}
}

// isOKFConcept reports whether the document looks like an OKF concept: a
// Markdown file with YAML front matter that is not an OpenSpec spec, change,
// or extension artifact and is not a reserved OKF file. Any other front-matter
// Markdown file is treated as an OKF concept per OKF v0.1.
func isOKFConcept(uri, text string) bool {
	if !strings.HasSuffix(strings.ToLower(uri), ".md") {
		return false
	}
	if frontMatter(text) == nil {
		return false
	}
	if isOKFReserved(uri) {
		return false
	}
	if documentRole(uri) != symbolDocument {
		return false
	}
	return true
}

// okfDiagnostics validates an OKF concept document and reports the library's
// conformance issues at their front-matter locations.
func okfDiagnostics(uri, text string) []diagnostic {
	if !isOKFConcept(uri, text) {
		return nil
	}
	concept, err := okfParse(uri, text)
	if err != nil {
		if !frontMatterClosed(text) {
			// The block is still being written; reporting a hard parse error
			// on every keystroke is noise, not a conformance finding.
			return nil
		}
		return []diagnostic{diag(0, 0, 1, "OKF front matter: "+err.Error(), "okf-parse")}
	}
	report := openspec.ValidateOKFConceptReport(concept)
	var out []diagnostic
	for _, issue := range report.Issues {
		line, char := okfFieldLocation(text, issue.Path)
		out = append(out, diag(line, char, validationSeverity(issue.Level), "OKF "+issue.Path+": "+issue.Message, "okf"))
	}
	return out
}

// okfFieldLocation finds the front-matter line for a validation field path. It
// returns the opening "---" line when the field is absent so a missing-type or
// missing-title diagnostic still anchors inside the front matter.
func okfFieldLocation(text, field string) (int, int) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return 0, 0
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			break
		}
		key, _, ok := strings.Cut(lines[i], ":")
		if ok && strings.EqualFold(strings.TrimSpace(key), field) {
			return i, 0
		}
	}
	return 0, 0
}

// okfCompletions offers OKF concept front-matter keys when the cursor is inside
// the front matter of an OKF concept document.
func okfCompletions(uri, text string, pos position) []completionItem {
	if !isOKFConcept(uri, text) {
		return nil
	}
	if !inFrontMatter(text, pos.Line) {
		return nil
	}
	present := frontMatter(text)
	var items []completionItem
	for _, key := range []struct{ name, detail string }{
		{"type", "OKF concept type (required)"},
		{"title", "OKF concept title (recommended)"},
		{"description", "OKF concept description (recommended)"},
		{"resource", "OKF concept resource link"},
		{"tags", "OKF concept tags"},
		{"timestamp", "OKF concept ISO 8601 timestamp"},
	} {
		if _, ok := present[key.name]; ok {
			continue
		}
		items = append(items, completionItem{Label: key.name + ":", Kind: completionKindText, Detail: key.detail, InsertText: key.name + ": "})
	}
	return items
}

// frontMatterClosed reports whether a leading "---" front-matter block has a
// matching closing fence.
func frontMatterClosed(text string) bool {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return true
		}
	}
	return false
}

// inFrontMatter reports whether line falls inside a leading YAML front-matter
// block delimited by "---" fences. An unterminated block (still being written)
// extends to the end of the document so completions stay available.
func inFrontMatter(text string, line int) bool {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return line >= 0 && line <= i
		}
	}
	return line >= 0 && line < len(lines)
}

// okfHover returns hover text for an OKF concept document, summarizing the
// concept's type and recognized fields. It returns "" when uri is not an OKF
// concept so the caller can fall back to generic hover.
func okfHover(uri, text string) string {
	if !isOKFConcept(uri, text) {
		return ""
	}
	concept, err := okfParse(uri, text)
	if err != nil {
		return "OKF concept (unparseable front matter): " + err.Error()
	}
	var parts []string
	if concept.Type != "" {
		parts = append(parts, "type: "+concept.Type)
	} else {
		parts = append(parts, "type: (missing, required)")
	}
	if concept.Title != "" {
		parts = append(parts, "title: "+concept.Title)
	}
	if concept.Description != "" {
		parts = append(parts, "description: "+concept.Description)
	}
	if len(concept.Tags) > 0 {
		parts = append(parts, "tags: "+strings.Join(concept.Tags, ", "))
	}
	return "OKF concept\n\n" + strings.Join(parts, "\n")
}
