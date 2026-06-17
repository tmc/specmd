package openspec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"rsc.io/markdown"
)

// ParseSpec reads an OpenSpec spec Markdown document.
func ParseSpec(name string, r io.Reader) (*Spec, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("parse spec: empty name")
	}
	spec, err := parseSpec(name, r, "")
	if err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}
	return spec, nil
}

// ParseSpecFile reads an OpenSpec spec file.
func ParseSpecFile(path string) (*Spec, error) {
	name := specNameFromPath(path)
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("parse spec file: cannot infer spec name from %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("parse spec file: %w", err)
	}
	defer f.Close()
	spec, err := parseSpec(name, f, path)
	if err != nil {
		return nil, fmt.Errorf("parse spec file: %w", err)
	}
	return spec, nil
}

func parseSpec(name string, r io.Reader, sourcePath string) (*Spec, error) {
	doc, err := readMarkdown(r)
	if err != nil {
		return nil, err
	}
	spec := &Spec{
		Name:     strings.TrimSpace(name),
		Metadata: Metadata{Version: "1.0.0", Format: "openspec", SourcePath: sourcePath},
	}
	spec.Overview = sectionText(doc, "Purpose")
	spec.Requirements = parseRequirements(sectionText(doc, "Requirements"))
	return spec, nil
}

// ParseChange reads an OpenSpec change proposal and its delta spec documents.
func ParseChange(name string, proposal io.Reader, deltas map[string]io.Reader) (*Change, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("parse change: empty name")
	}
	change, err := parseChange(name, proposal, "", deltas, nil)
	if err != nil {
		return nil, fmt.Errorf("parse change: %w", err)
	}
	return change, nil
}

// ParseChangeDir reads an OpenSpec change directory.
func ParseChangeDir(path string) (*Change, error) {
	name := filepath.Base(filepath.Clean(path))
	proposalPath := filepath.Join(path, "proposal.md")
	proposal, err := os.Open(proposalPath)
	if err != nil {
		return nil, fmt.Errorf("parse change dir: %w", err)
	}
	defer proposal.Close()

	specs, err := deltaSpecFiles(filepath.Join(path, "specs"))
	if err != nil {
		return nil, fmt.Errorf("parse change dir: %w", err)
	}
	readers := make(map[string]io.Reader, len(specs))
	paths := make(map[string]string, len(specs))
	files := make([]*os.File, 0, len(specs))
	defer func() {
		for _, f := range files {
			f.Close()
		}
	}()
	for _, spec := range specs {
		f, err := os.Open(spec.path)
		if err != nil {
			return nil, fmt.Errorf("parse change dir: %w", err)
		}
		files = append(files, f)
		readers[spec.name] = f
		paths[spec.name] = spec.path
	}
	change, err := parseChange(name, proposal, proposalPath, readers, paths)
	if err != nil {
		return nil, fmt.Errorf("parse change dir: %w", err)
	}
	extensions, err := extensionRefs(filepath.Join(path, "extensions"))
	if err != nil {
		return nil, fmt.Errorf("parse change dir: %w", err)
	}
	change.Extensions = extensions
	return change, nil
}

// ParseProject reads specs and changes from an openspec directory.
func ParseProject(path string) (*Project, error) {
	specFiles, err := specFiles(filepath.Join(path, "specs"))
	if err != nil {
		return nil, fmt.Errorf("parse project: %w", err)
	}
	var project Project
	for _, file := range specFiles {
		spec, err := ParseSpecFile(file.path)
		if err != nil {
			return nil, fmt.Errorf("parse project: %w", err)
		}
		project.Specs = append(project.Specs, *spec)
	}

	changeDirs, err := changeDirs(filepath.Join(path, "changes"))
	if err != nil {
		return nil, fmt.Errorf("parse project: %w", err)
	}
	for _, dir := range changeDirs {
		change, err := ParseChangeDir(dir)
		if err != nil {
			return nil, fmt.Errorf("parse project: %w", err)
		}
		project.Changes = append(project.Changes, *change)
	}
	extensions, err := extensionRefs(filepath.Join(path, "extensions"))
	if err != nil {
		return nil, fmt.Errorf("parse project: %w", err)
	}
	project.Extensions = extensions
	return &project, nil
}

func parseChange(name string, proposal io.Reader, sourcePath string, deltas map[string]io.Reader, deltaPaths map[string]string) (*Change, error) {
	doc, err := readMarkdown(proposal)
	if err != nil {
		return nil, err
	}
	change := &Change{
		Name:        strings.TrimSpace(name),
		Why:         sectionText(doc, "Why"),
		WhatChanges: sectionText(doc, "What Changes"),
		Metadata:    Metadata{Version: "1.0.0", Format: "openspec-change", SourcePath: sourcePath},
	}
	names := sortedKeys(deltas)
	for _, spec := range names {
		r := deltas[spec]
		deltaDoc, err := readMarkdown(r)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec, err)
		}
		change.Deltas = append(change.Deltas, parseDeltas(spec, deltaDoc, deltaPaths[spec])...)
	}
	return change, nil
}

type markdownLine struct {
	level int
	text  string
	raw   string
}

func readMarkdown(r io.Reader) ([]markdownLine, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	raw := strings.Split(text, "\n")
	if len(raw) > 0 && raw[len(raw)-1] == "" {
		raw = raw[:len(raw)-1]
	}

	lines := make([]markdownLine, len(raw))
	for i, line := range raw {
		lines[i].raw = line
	}
	doc := new(markdown.Parser).Parse(text)
	for _, h := range headings(doc.Blocks) {
		line := h.StartLine - 1
		if line < 0 || line >= len(lines) {
			continue
		}
		lines[line].level = h.Level
		lines[line].text = strings.TrimSpace(markdown.Format(h.Text))
	}

	return lines, nil
}

func headings(blocks []markdown.Block) []*markdown.Heading {
	var out []*markdown.Heading
	for _, block := range blocks {
		switch block := block.(type) {
		case *markdown.Heading:
			out = append(out, block)
		case *markdown.Document:
			out = append(out, headings(block.Blocks)...)
		case *markdown.Item:
			out = append(out, headings(block.Blocks)...)
		case *markdown.List:
			for _, item := range block.Items {
				if item, ok := item.(*markdown.Item); ok {
					out = append(out, headings(item.Blocks)...)
				}
			}
		case *markdown.Quote:
			out = append(out, headings(block.Blocks)...)
		}
	}
	return out
}

func sectionText(lines []markdownLine, name string) string {
	for i, line := range lines {
		if line.level == 2 && strings.EqualFold(line.text, name) {
			return collectUntil(lines[i+1:], 2)
		}
	}
	return ""
}

func collectUntil(lines []markdownLine, maxLevel int) string {
	var out []string
	for _, line := range lines {
		if line.level > 0 && line.level <= maxLevel {
			break
		}
		out = append(out, line.raw)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func parseRequirements(markdown string) []Requirement {
	lines, _ := readMarkdown(strings.NewReader(markdown))
	var reqs []Requirement
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		name, ok := requirementName(line.text)
		if line.level != 3 || !ok {
			continue
		}
		req := Requirement{Name: name}
		req.Text = collectRequirementText(lines[i+1:])
		req.Scenarios = parseScenarios(lines[i+1:])
		reqs = append(reqs, req)
	}
	return reqs
}

func collectRequirementText(lines []markdownLine) string {
	var out []string
	for _, line := range lines {
		if line.level > 0 && line.level <= 3 {
			break
		}
		if line.level == 4 && hasPrefixFold(line.text, "Scenario:") {
			break
		}
		out = append(out, line.raw)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func parseScenarios(lines []markdownLine) []Scenario {
	var scenarios []Scenario
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line.level > 0 && line.level <= 3 {
			break
		}
		if line.level != 4 || !hasPrefixFold(line.text, "Scenario:") {
			continue
		}
		scenario := Scenario{Name: strings.TrimSpace(line.text[len("Scenario:"):])}
		scenario.RawText = collectUntil(lines[i+1:], 4)
		scenarios = append(scenarios, scenario)
	}
	return scenarios
}

func parseDeltas(spec string, lines []markdownLine, sourcePath string) []Delta {
	var deltas []Delta
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line.level != 2 {
			continue
		}
		op, ok := parseDeltaHeading(line.text)
		if !ok {
			continue
		}
		body := collectUntil(lines[i+1:], 2)
		reqs := parseRequirements(body)
		renames := parseRenames(body)
		description := firstParagraph(body)
		if description == "" {
			description = deltaDescription(op, reqs, renames)
		}
		deltas = append(deltas, Delta{
			Spec:         spec,
			Operation:    op,
			Description:  description,
			Requirements: reqs,
			Renames:      renames,
			Metadata:     Metadata{SourcePath: sourcePath},
		})
	}
	return deltas
}

type namedPath struct {
	name string
	path string
}

func specFiles(root string) ([]namedPath, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []namedPath
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), "spec.md")
		if _, err := os.Stat(path); err == nil {
			files = append(files, namedPath{entry.Name(), path})
		} else if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files, nil
}

func deltaSpecFiles(root string) ([]namedPath, error) {
	return specFiles(root)
}

func changeDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archive" {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(path, "proposal.md")); err == nil {
			dirs = append(dirs, path)
		} else if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

func extensionRefs(root string) ([]ExtensionRef, error) {
	var refs []ExtensionRef
	if err := walkExtensionRefs(root, "", &refs); err != nil {
		return nil, err
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Name < refs[j].Name })
	return refs, nil
}

func walkExtensionRefs(root, prefix string, refs *[]ExtensionRef) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		name := entry.Name()
		if prefix != "" {
			name = prefix + "/" + name
		}
		if entry.IsDir() {
			if err := walkExtensionRefs(path, name, refs); err != nil {
				return err
			}
			continue
		}
		if filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		name = strings.TrimSuffix(name, ".md")
		*refs = append(*refs, ExtensionRef{Name: filepath.ToSlash(name), SourcePath: path})
	}
	return nil
}

func specNameFromPath(path string) string {
	path = filepath.Clean(path)
	if filepath.Base(path) == "spec.md" {
		return filepath.Base(filepath.Dir(path))
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func parseDeltaHeading(s string) (DeltaOperation, bool) {
	fields := strings.Fields(s)
	if len(fields) < 2 || !strings.EqualFold(fields[1], "Requirements") {
		return "", false
	}
	op := DeltaOperation(strings.ToUpper(fields[0]))
	switch op {
	case Added, Modified, Removed, Renamed:
		return op, true
	default:
		return "", false
	}
}

func firstParagraph(s string) string {
	for _, part := range strings.Split(s, "\n\n") {
		part = strings.TrimSpace(part)
		if part != "" && !strings.HasPrefix(part, "### ") {
			return part
		}
	}
	return ""
}

func requirementName(s string) (string, bool) {
	if !hasPrefixFold(s, "Requirement:") {
		return "", false
	}
	return strings.TrimSpace(s[len("Requirement:"):]), true
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

func parseRenames(markdown string) []Rename {
	lines := strings.Split(markdown, "\n")
	var renames []Rename
	var from string
	for _, line := range lines {
		key, name, ok := renameLine(line)
		if !ok {
			continue
		}
		switch key {
		case "FROM":
			from = name
		case "TO":
			if from != "" {
				renames = append(renames, Rename{From: from, To: name})
				from = ""
			}
		}
	}
	return renames
}

func renameLine(line string) (key, name string, ok bool) {
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key = strings.ToUpper(strings.TrimSpace(parts[0]))
	if key != "FROM" && key != "TO" {
		return "", "", false
	}
	name = strings.Trim(strings.TrimSpace(parts[1]), "`")
	name = strings.TrimSpace(strings.TrimPrefix(name, "###"))
	req, ok := requirementName(strings.TrimSpace(name))
	return key, req, ok
}

func deltaDescription(op DeltaOperation, reqs []Requirement, renames []Rename) string {
	switch {
	case len(reqs) > 0:
		return fmt.Sprintf("%s requirement: %s", deltaVerb(op), reqs[0].Text)
	case len(renames) > 0:
		return fmt.Sprintf("Rename requirement from %q to %q", renames[0].From, renames[0].To)
	default:
		return string(op) + " requirements"
	}
}

func deltaVerb(op DeltaOperation) string {
	switch op {
	case Added:
		return "Add"
	case Modified:
		return "Modify"
	case Removed:
		return "Remove"
	case Renamed:
		return "Rename"
	default:
		return string(op)
	}
}
