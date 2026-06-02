package openspec

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ParseSpec reads an OpenSpec spec Markdown document.
func ParseSpec(name string, r io.Reader) (*Spec, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("parse spec: empty name")
	}
	spec, err := parseSpec(name, r)
	if err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}
	return spec, nil
}

func parseSpec(name string, r io.Reader) (*Spec, error) {
	doc, err := readMarkdown(r)
	if err != nil {
		return nil, err
	}
	spec := &Spec{
		Name:     strings.TrimSpace(name),
		Metadata: Metadata{Version: "1.0.0", Format: "openspec"},
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
	change, err := parseChange(name, proposal, deltas)
	if err != nil {
		return nil, fmt.Errorf("parse change: %w", err)
	}
	return change, nil
}

func parseChange(name string, proposal io.Reader, deltas map[string]io.Reader) (*Change, error) {
	doc, err := readMarkdown(proposal)
	if err != nil {
		return nil, err
	}
	change := &Change{
		Name:        strings.TrimSpace(name),
		Why:         sectionText(doc, "Why"),
		WhatChanges: sectionText(doc, "What Changes"),
		Metadata:    Metadata{Version: "1.0.0", Format: "openspec-change"},
	}
	for spec, r := range deltas {
		deltaDoc, err := readMarkdown(r)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec, err)
		}
		change.Deltas = append(change.Deltas, parseDeltas(spec, deltaDoc)...)
	}
	return change, nil
}

type markdownLine struct {
	level int
	text  string
	raw   string
}

func readMarkdown(r io.Reader) ([]markdownLine, error) {
	var lines []markdownLine
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		raw := scan.Text()
		level, text := heading(raw)
		lines = append(lines, markdownLine{level: level, text: text, raw: raw})
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func heading(s string) (int, string) {
	s = strings.TrimSpace(s)
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n == 0 || n == len(s) || s[n] != ' ' {
		return 0, ""
	}
	return n, strings.TrimSpace(s[n+1:])
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

func parseDeltas(spec string, lines []markdownLine) []Delta {
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
		})
	}
	return deltas
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
