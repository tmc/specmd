package openspec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ParseOKFConcept reads one Open Knowledge Format concept document.
func ParseOKFConcept(id string, r io.Reader) (*OKFConcept, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("parse okf concept: empty id")
	}
	concept, err := parseOKFConcept(id, r, "")
	if err != nil {
		return nil, fmt.Errorf("parse okf concept: %w", err)
	}
	return concept, nil
}

// ParseOKFConceptFile reads one Open Knowledge Format concept file.
func ParseOKFConceptFile(path string) (*OKFConcept, error) {
	id := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(path)), ".md")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("parse okf concept file: %w", err)
	}
	defer f.Close()
	concept, err := parseOKFConcept(id, f, path)
	if err != nil {
		return nil, fmt.Errorf("parse okf concept file: %w", err)
	}
	return concept, nil
}

// ParseOKFBundle reads an Open Knowledge Format bundle directory.
func ParseOKFBundle(path string) (*OKFBundle, error) {
	var files []string
	if err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".md") {
			files = append(files, p)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("parse okf bundle: %w", err)
	}
	sort.Strings(files)

	bundle := &OKFBundle{
		Root:     path,
		Metadata: Metadata{Version: "0.1", Format: "okf", SourcePath: path},
	}
	for _, file := range files {
		rel, err := filepath.Rel(path, file)
		if err != nil {
			return nil, fmt.Errorf("parse okf bundle: %w", err)
		}
		name := filepath.Base(file)
		switch strings.ToLower(name) {
		case "index.md", "log.md":
			rf, err := parseOKFReservedFile(file, name)
			if err != nil {
				return nil, fmt.Errorf("parse okf bundle: %w", err)
			}
			rf.Root = filepath.Dir(rel) == "."
			if strings.EqualFold(name, "index.md") {
				bundle.Index = append(bundle.Index, *rf)
				if rf.Root {
					bundle.Version = fieldValue(rf.FrontMatter, "okf_version")
				}
			} else {
				bundle.Logs = append(bundle.Logs, *rf)
			}
		default:
			id := conceptID(rel)
			f, err := os.Open(file)
			if err != nil {
				return nil, fmt.Errorf("parse okf bundle: %w", err)
			}
			concept, err := parseOKFConcept(id, f, file)
			if cerr := f.Close(); cerr != nil {
				return nil, fmt.Errorf("parse okf bundle: %w", cerr)
			}
			if err != nil {
				// A malformed concept is a conformance failure, not a
				// reason to reject the whole bundle. Record and continue.
				bundle.Invalid = append(bundle.Invalid, OKFInvalidConcept{ID: id, SourcePath: file, Err: err})
				continue
			}
			bundle.Concepts = append(bundle.Concepts, *concept)
		}
	}
	return bundle, nil
}

func parseOKFConcept(id string, r io.Reader, sourcePath string) (*OKFConcept, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	fields, body, err := parseFrontMatter(string(b))
	if err != nil {
		return nil, err
	}
	c := &OKFConcept{
		ID:          strings.TrimSpace(id),
		FrontMatter: fields,
		Body:        body,
		Metadata:    Metadata{Version: "0.1", Format: "okf-concept", SourcePath: sourcePath},
	}
	for _, f := range fields {
		switch strings.ToLower(f.Key) {
		case "type":
			c.Type = firstValue(f.Values)
		case "title":
			c.Title = firstValue(f.Values)
		case "description":
			c.Description = firstValue(f.Values)
		case "resource":
			c.Resource = firstValue(f.Values)
		case "tags":
			c.Tags = append([]string(nil), f.Values...)
		case "timestamp":
			c.Timestamp = firstValue(f.Values)
		}
	}
	return c, nil
}

func parseOKFReservedFile(path, name string) (*OKFReservedFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	body := normalizeNewlines(string(b))
	var fields []OKFField
	if strings.HasPrefix(body, "---\n") {
		var rest string
		fields, rest, err = parseFrontMatter(body)
		if err != nil {
			return nil, err
		}
		body = rest
	}
	return &OKFReservedFile{Name: name, Body: strings.TrimSpace(body), FrontMatter: fields, Metadata: Metadata{Version: "0.1", Format: "okf-reserved", SourcePath: path}}, nil
}

func parseFrontMatter(text string) ([]OKFField, string, error) {
	text = normalizeNewlines(text)
	if !strings.HasPrefix(text, "---\n") {
		return nil, "", fmt.Errorf("missing frontmatter")
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if strings.HasSuffix(rest, "\n---") {
			end = len(rest) - len("\n---")
		} else {
			return nil, "", fmt.Errorf("unterminated frontmatter")
		}
	}
	raw := rest[:end]
	body := ""
	if end+len("\n---\n") <= len(rest) {
		body = rest[end+len("\n---\n"):]
	}
	fields, err := parseFrontMatterFields(raw)
	if err != nil {
		return nil, "", err
	}
	return fields, strings.TrimSpace(body), nil
}

func parseFrontMatterFields(text string) ([]OKFField, error) {
	var fields []OKFField
	for lineNo, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// A block-style list item ("- value") extends the preceding key.
		if item, ok := blockListItem(line); ok {
			if len(fields) == 0 {
				return nil, fmt.Errorf("frontmatter line %d: list item without key", lineNo+1)
			}
			if item != "" {
				last := &fields[len(fields)-1]
				last.Values = append(last.Values, item)
			}
			continue
		}
		key, raw, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("frontmatter line %d: missing colon", lineNo+1)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("frontmatter line %d: empty key", lineNo+1)
		}
		// A key with no inline value may introduce a block-style list.
		if strings.TrimSpace(raw) == "" {
			fields = append(fields, OKFField{Key: key})
			continue
		}
		fields = append(fields, OKFField{Key: key, Values: parseFrontMatterValue(raw)})
	}
	return fields, nil
}

// blockListItem reports whether line is a YAML block-style list item
// ("- value") and returns the unquoted value.
func blockListItem(line string) (string, bool) {
	if line != "-" && !strings.HasPrefix(line, "- ") {
		return "", false
	}
	return trimQuote(strings.TrimSpace(line[1:])), true
}

func parseFrontMatterValue(raw string) []string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]"))
		if raw == "" {
			return nil
		}
		var values []string
		for _, part := range strings.Split(raw, ",") {
			value := trimQuote(strings.TrimSpace(part))
			if value != "" {
				values = append(values, value)
			}
		}
		return values
	}
	return []string{trimQuote(raw)}
}

func conceptID(path string) string {
	return strings.TrimSuffix(filepath.ToSlash(filepath.Clean(path)), ".md")
}

func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func fieldValue(fields []OKFField, name string) string {
	for _, field := range fields {
		if strings.EqualFold(field.Key, name) {
			return firstValue(field.Values)
		}
	}
	return ""
}

func trimQuote(s string) string {
	return strings.Trim(strings.TrimSpace(s), `"'`)
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}
