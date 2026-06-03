package lsp

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

const source = "openspec"

type heading struct {
	Level int
	Text  string
	Line  int
}

func analyze(uri, text string) []diagnostic {
	var diags []diagnostic
	heads := headings(text)
	diags = append(diags, requireSections(heads, requiredSections(uri)...)...)
	diags = append(diags, headingWhitespaceDiagnostics(text)...)
	sort.Slice(diags, func(i, j int) bool {
		if diags[i].Range.Start.Line != diags[j].Range.Start.Line {
			return diags[i].Range.Start.Line < diags[j].Range.Start.Line
		}
		return diags[i].Message < diags[j].Message
	})
	return diags
}

func requiredSections(uri string) []string {
	base := path.Base(uri)
	ext := extensionName(uri)
	switch {
	case base == "spec.md" && strings.Contains(uri, "/openspec/specs/"):
		return []string{"Purpose", "Requirements"}
	case base == "proposal.md" && strings.Contains(uri, "/openspec/changes/"):
		return []string{"Why", "What Changes"}
	case ext != "":
		return extensionRequiredSections(ext)
	default:
		return nil
	}
}

func headings(text string) []heading {
	var out []heading
	inFence := false
	for i, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence || !strings.HasPrefix(line, "#") {
			continue
		}
		n := 0
		for n < len(line) && line[n] == '#' {
			n++
		}
		if n == 0 || n > 6 || n >= len(line) || line[n] != ' ' {
			continue
		}
		out = append(out, heading{Level: n, Text: strings.TrimSpace(line[n+1:]), Line: i})
	}
	return out
}

func requireSections(heads []heading, names ...string) []diagnostic {
	var diags []diagnostic
	for _, name := range names {
		found := false
		for _, h := range heads {
			if h.Level == 2 && strings.EqualFold(h.Text, name) {
				found = true
				break
			}
		}
		if !found {
			diags = append(diags, diag(0, 0, 1, fmt.Sprintf("missing ## %s section", name), "section"))
		}
	}
	return diags
}

func extensionDiagnostics(name string, heads []heading) []diagnostic {
	return requireSections(heads, extensionRequiredSections(name)...)
}

func extensionRequiredSections(name string) []string {
	switch name {
	case "ooux":
		return []string{"Objects"}
	case "eventstorm":
		return []string{"Events", "Commands", "Actors"}
	case "contexts":
		return []string{"Contexts", "Relationships"}
	case "domain-story":
		return []string{"Actors", "Story"}
	case "example-mapping":
		return []string{"Story", "Rules", "Examples", "Questions"}
	case "jobs":
		return []string{"Stories"}
	case "journey":
		return []string{"Actor", "Scenario", "Stages"}
	case "opportunity-tree":
		return []string{"Outcome", "Opportunities", "Solutions", "Experiments"}
	case "service-blueprint":
		return []string{"Blueprint"}
	case "stratmd":
		return []string{"Changelog"}
	case "magi":
		return nil
	default:
		return nil
	}
}

func extensionName(uri string) string {
	i := strings.LastIndex(uri, "/extensions/")
	if i < 0 {
		return ""
	}
	rest := strings.TrimPrefix(uri[i+len("/extensions/"):], "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 {
		return ""
	}
	name := strings.TrimSuffix(parts[0], ".md")
	return name
}

func headingWhitespaceDiagnostics(text string) []diagnostic {
	var diags []diagnostic
	for i, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "#\u00a0") || strings.HasPrefix(line, "##\u00a0") || strings.HasPrefix(line, "###\u00a0") {
			diags = append(diags, diag(i, strings.Index(line, "\u00a0"), 2, "heading uses non-breaking space after #", "heading-space"))
		}
	}
	return diags
}

func diag(line, char, severity int, msg, code string) diagnostic {
	if char < 0 {
		char = 0
	}
	return diagnostic{
		Range:    range_{Start: position{Line: line, Character: char}, End: position{Line: line, Character: char + 1}},
		Severity: severity,
		Code:     code,
		Source:   source,
		Message:  msg,
	}
}

func symbols(text string) []documentSymbol {
	heads := headings(text)
	out := make([]documentSymbol, 0, len(heads))
	for _, h := range heads {
		r := range_{Start: position{Line: h.Line, Character: 0}, End: position{Line: h.Line, Character: len(h.Text) + h.Level + 1}}
		out = append(out, documentSymbol{Name: h.Text, Kind: 13, Range: r, SelectionRange: r})
	}
	return out
}

func completions(uri, text string) []completionItem {
	var items []completionItem
	labels := make(map[string]bool)
	seen := presentSections(headings(text))
	for _, sec := range requiredSections(uri) {
		if !seen[strings.ToLower(sec)] {
			items = appendCompletion(items, labels, completionItem{Label: "## " + sec, Kind: 15, Detail: "missing required OpenSpec section", InsertText: "## " + sec + "\n"})
		}
	}
	for _, item := range []completionItem{
		{Label: "## Purpose", Kind: 15, Detail: "OpenSpec spec section", InsertText: "## Purpose\n"},
		{Label: "## Requirements", Kind: 15, Detail: "OpenSpec spec section", InsertText: "## Requirements\n"},
		{Label: "### Requirement:", Kind: 15, Detail: "OpenSpec requirement", InsertText: "### Requirement: "},
		{Label: "#### Scenario:", Kind: 15, Detail: "OpenSpec scenario", InsertText: "#### Scenario: "},
		{Label: "## ADDED Requirements", Kind: 15, Detail: "OpenSpec delta section", InsertText: "## ADDED Requirements\n"},
		{Label: "## MODIFIED Requirements", Kind: 15, Detail: "OpenSpec delta section", InsertText: "## MODIFIED Requirements\n"},
		{Label: "## REMOVED Requirements", Kind: 15, Detail: "OpenSpec delta section", InsertText: "## REMOVED Requirements\n"},
		{Label: "## RENAMED Requirements", Kind: 15, Detail: "OpenSpec delta section", InsertText: "## RENAMED Requirements\n"},
	} {
		items = appendCompletion(items, labels, item)
	}
	for _, sec := range extensionSections(extensionName(uri)) {
		items = appendCompletion(items, labels, completionItem{Label: "## " + sec, Kind: 15, Detail: "OpenSpec extension section", InsertText: "## " + sec + "\n"})
	}
	return items
}

func appendCompletion(items []completionItem, labels map[string]bool, item completionItem) []completionItem {
	if labels[item.Label] {
		return items
	}
	labels[item.Label] = true
	return append(items, item)
}

func presentSections(heads []heading) map[string]bool {
	seen := make(map[string]bool)
	for _, h := range heads {
		if h.Level == 2 {
			seen[strings.ToLower(h.Text)] = true
		}
	}
	return seen
}

func extensionSections(name string) []string {
	switch name {
	case "ooux":
		return []string{"Objects", "Attributes", "Relationships", "Calls to Action"}
	case "eventstorm":
		return []string{"Events", "Commands", "Actors", "External Systems", "Policies", "Read Models"}
	case "contexts":
		return []string{"Contexts", "Relationships"}
	case "domain-story":
		return []string{"Actors", "Story"}
	case "example-mapping":
		return []string{"Story", "Rules", "Examples", "Questions"}
	case "jobs":
		return []string{"Stories"}
	case "journey":
		return []string{"Actor", "Scenario", "Stages"}
	case "opportunity-tree":
		return []string{"Outcome", "Opportunities", "Solutions", "Experiments"}
	case "service-blueprint":
		return []string{"Blueprint"}
	case "stratmd":
		return []string{"Strategic Intent", "Objective", "Goals", "Risks", "Actions", "Changelog"}
	default:
		return nil
	}
}

func hoverFor(uri string) string {
	switch extensionName(uri) {
	case "ooux":
		return "OOUX / ORCA extension: objects, relationships, calls to action, and attributes."
	case "eventstorm":
		return "EventStorming extension: events, commands, actors, systems, policies, and read models."
	case "contexts":
		return "Context Map extension: bounded contexts and their upstream/downstream relationships."
	case "example-mapping":
		return "Example Mapping extension: story, rules, examples, and questions."
	case "service-blueprint":
		return "Service Blueprint extension: evidence, customer actions, frontstage, backstage, and support processes."
	case "journey":
		return "Journey Map extension: actor, scenario, stages, actions, mindset, emotions, and opportunities."
	case "jobs":
		return "Job Stories extension: When situation, I want motivation, so I can outcome."
	case "opportunity-tree":
		return "Opportunity Solution Tree extension: outcome, opportunities, solutions, and experiments."
	default:
		return "OpenSpec Markdown document."
	}
}

func hoverAt(uri, text string, pos position) string {
	for _, h := range headings(text) {
		if h.Line == pos.Line {
			if s := sectionHover(uri, h.Text); s != "" {
				return s
			}
			return fmt.Sprintf("Markdown heading: %s.", h.Text)
		}
	}
	return hoverFor(uri)
}

func sectionHover(uri, name string) string {
	switch strings.ToLower(name) {
	case "purpose":
		return "Purpose describes why the OpenSpec capability exists."
	case "requirements":
		return "Requirements contain user-visible behavior and scenarios."
	case "requirement:":
		return "Requirement headings name one behavior contract."
	case "scenario:":
		return "Scenario headings describe concrete GIVEN/WHEN/THEN behavior."
	}
	for _, sec := range extensionSections(extensionName(uri)) {
		if strings.EqualFold(name, sec) {
			return "OpenSpec extension section: " + sec + "."
		}
	}
	if strings.HasPrefix(strings.ToLower(name), "requirement:") {
		return "Requirement headings name one behavior contract."
	}
	if strings.HasPrefix(strings.ToLower(name), "scenario:") {
		return "Scenario headings describe concrete GIVEN/WHEN/THEN behavior."
	}
	return ""
}
