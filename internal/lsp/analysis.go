package lsp

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	openspec "github.com/tmc/openspec"
)

const source = "openspec"

const (
	completionKindText    = 1
	completionKindSnippet = 15
	insertTextSnippet     = 2
)

type heading struct {
	Level int
	Text  string
	Line  int
	End   int
}

func analyze(uri, text string) []diagnostic {
	var diags []diagnostic
	heads := headings(text)
	diags = append(diags, requireSections(heads, requiredSections(uri)...)...)
	diags = append(diags, headingWhitespaceDiagnostics(text)...)
	diags = append(diags, validationDiagnostics(uri, text, diags)...)
	sort.Slice(diags, func(i, j int) bool {
		if diags[i].Range.Start.Line != diags[j].Range.Start.Line {
			return diags[i].Range.Start.Line < diags[j].Range.Start.Line
		}
		return diags[i].Message < diags[j].Message
	})
	return diags
}

func validationDiagnostics(uri, text string, existing []diagnostic) []diagnostic {
	if hasSectionDiagnostics(existing) {
		return nil
	}
	if !(path.Base(uri) == "spec.md" && strings.Contains(uri, "/openspec/specs/")) {
		return nil
	}
	spec, err := openspec.ParseSpec(specNameFromURI(uri), strings.NewReader(text))
	if err != nil {
		return []diagnostic{diag(0, 0, 1, err.Error(), "parse")}
	}
	report := openspec.ValidateSpecReport(spec)
	var out []diagnostic
	for _, issue := range report.Issues {
		out = append(out, diag(0, 0, validationSeverity(issue.Level), issue.Path+": "+issue.Message, "validation"))
	}
	return out
}

func hasSectionDiagnostics(diags []diagnostic) bool {
	for _, diag := range diags {
		if diag.Code == "section" {
			return true
		}
	}
	return false
}

func validationSeverity(level openspec.ValidationLevel) int {
	switch level {
	case openspec.ValidationLevelError:
		return 1
	case openspec.ValidationLevelWarning:
		return 2
	default:
		return 3
	}
}

func specNameFromURI(uri string) string {
	parts := strings.Split(uri, "/")
	for i := 0; i+2 < len(parts); i++ {
		if parts[i] == "specs" && parts[i+2] == "spec.md" {
			return parts[i+1]
		}
	}
	return strings.TrimSuffix(path.Base(uri), ".md")
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
		end := len(strings.TrimRight(line, " \t\r"))
		out = append(out, heading{Level: n, Text: strings.TrimSpace(line[n+1:]), Line: i, End: utf16Len(line[:end])})
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
		Range:    textRange{Start: position{Line: line, Character: char}, End: position{Line: line, Character: char + 1}},
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
		r := textRange{Start: position{Line: h.Line, Character: 0}, End: position{Line: h.Line, Character: h.End}}
		out = append(out, documentSymbol{Name: h.Text, Kind: 13, Range: r, SelectionRange: r})
	}
	return out
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

func completions(uri, text string) []completionItem {
	var items []completionItem
	labels := make(map[string]bool)
	seen := presentSections(headings(text))
	for _, sec := range requiredSections(uri) {
		if !seen[strings.ToLower(sec)] {
			items = appendCompletion(items, labels, completionItem{Label: "## " + sec, Kind: completionKindSnippet, Detail: "missing required OpenSpec section", InsertText: "## " + sec + "\n"})
		}
	}
	for _, item := range coreCompletions(uri) {
		items = appendCompletion(items, labels, item)
	}
	for _, sec := range extensionSections(extensionName(uri)) {
		items = appendCompletion(items, labels, completionItem{Label: "## " + sec, Kind: completionKindSnippet, Detail: "OpenSpec extension section", InsertText: "## " + sec + "\n"})
	}
	for _, item := range extensionCompletions(extensionName(uri)) {
		items = appendCompletion(items, labels, item)
	}
	return items
}

func (s *Server) completions(uri, text string, pos position) []completionItem {
	items := completions(uri, text)
	labels := make(map[string]bool)
	for _, item := range items {
		labels[item.Label] = true
	}
	for _, item := range s.indexCompletions(uri, text, pos) {
		items = appendCompletion(items, labels, item)
	}
	return items
}

func (s *Server) indexCompletions(uri, text string, pos position) []completionItem {
	line := lineAt(text, pos.Line)
	var items []completionItem
	prefix := line
	if off := byteOffsetForUTF16(line, max(pos.Character, 0)); off >= 0 && off < len(line) {
		prefix = line[:off]
	}
	if strings.Contains(prefix, "[") {
		for _, doc := range s.indexedDocs() {
			label := relativeLabel(uri, doc.URI)
			if label != "" {
				items = append(items, completionItem{Label: label, Kind: completionKindText, Detail: "Markdown file", InsertText: label})
			}
			for _, sym := range doc.Symbols {
				if sym.Role == symbolHeading || sym.Role == symbolObject {
					target := label + "#" + slug(sym.Canon)
					items = append(items, completionItem{Label: target, Kind: completionKindText, Detail: "Markdown heading", InsertText: target})
				}
			}
		}
	}
	if strings.Contains(line, "|") {
		for _, sym := range s.indexedSymbols() {
			if sym.Role == symbolObject && !sym.Reference {
				items = append(items, completionItem{Label: sym.Canon, Kind: completionKindText, Detail: "OOUX object", InsertText: sym.Canon})
			}
		}
	}
	return items
}

func coreCompletions(uri string) []completionItem {
	items := []completionItem{
		{Label: "## Purpose", Kind: completionKindSnippet, Detail: "OpenSpec spec section", InsertText: "## Purpose\n"},
		{Label: "## Requirements", Kind: completionKindSnippet, Detail: "OpenSpec spec section", InsertText: "## Requirements\n"},
		{Label: "### Requirement:", Kind: completionKindSnippet, Detail: "OpenSpec requirement heading", InsertText: "### Requirement: ${1:name}\n\n$0", InsertTextFormat: insertTextSnippet},
		{Label: "#### Scenario:", Kind: completionKindSnippet, Detail: "OpenSpec scenario heading", InsertText: "#### Scenario: ${1:name}\n\n- GIVEN ${2:context}\n- WHEN ${3:action}\n- THEN ${4:outcome}\n", InsertTextFormat: insertTextSnippet},
		{Label: "Requirement block", Kind: completionKindSnippet, Detail: "OpenSpec requirement with scenario", InsertText: "### Requirement: ${1:name}\n\n#### Scenario: ${2:name}\n\n- GIVEN ${3:context}\n- WHEN ${4:action}\n- THEN ${5:outcome}\n", InsertTextFormat: insertTextSnippet},
		{Label: "GIVEN field", Kind: completionKindText, Detail: "OpenSpec scenario field", InsertText: "- GIVEN "},
		{Label: "WHEN field", Kind: completionKindText, Detail: "OpenSpec scenario field", InsertText: "- WHEN "},
		{Label: "THEN field", Kind: completionKindText, Detail: "OpenSpec scenario field", InsertText: "- THEN "},
		{Label: "AND field", Kind: completionKindText, Detail: "OpenSpec scenario field", InsertText: "- AND "},
		{Label: "## ADDED Requirements", Kind: completionKindSnippet, Detail: "OpenSpec delta section", InsertText: "## ADDED Requirements\n"},
		{Label: "## MODIFIED Requirements", Kind: completionKindSnippet, Detail: "OpenSpec delta section", InsertText: "## MODIFIED Requirements\n"},
		{Label: "## REMOVED Requirements", Kind: completionKindSnippet, Detail: "OpenSpec delta section", InsertText: "## REMOVED Requirements\n"},
		{Label: "## RENAMED Requirements", Kind: completionKindSnippet, Detail: "OpenSpec delta section", InsertText: "## RENAMED Requirements\n"},
		{Label: "ADDED requirement block", Kind: completionKindSnippet, Detail: "OpenSpec delta requirement", InsertText: "## ADDED Requirements\n\n### Requirement: ${1:name}\n\n#### Scenario: ${2:name}\n\n- GIVEN ${3:context}\n- WHEN ${4:action}\n- THEN ${5:outcome}\n", InsertTextFormat: insertTextSnippet},
	}
	if strings.HasSuffix(uri, "/proposal.md") {
		items = append([]completionItem{
			{Label: "## Why", Kind: completionKindSnippet, Detail: "OpenSpec change proposal section", InsertText: "## Why\n\n$0", InsertTextFormat: insertTextSnippet},
			{Label: "## What Changes", Kind: completionKindSnippet, Detail: "OpenSpec change proposal section", InsertText: "## What Changes\n\n$0", InsertTextFormat: insertTextSnippet},
		}, items...)
	}
	return items
}

func extensionCompletions(name string) []completionItem {
	switch name {
	case "ooux":
		return []completionItem{
			{Label: "OOUX object block", Kind: completionKindSnippet, Detail: "object, attributes, relationships, and CTAs", InsertText: "### ${1:Object}\n\n#### Attributes\n\n- ${2:attribute}\n\n#### Relationships\n\n- ${3:relationship}\n\n#### Calls to Action\n\n- ${4:action}\n", InsertTextFormat: insertTextSnippet},
			{Label: "#### Attributes", Kind: completionKindSnippet, Detail: "OOUX object subheading", InsertText: "#### Attributes\n\n- ${1:attribute}\n", InsertTextFormat: insertTextSnippet},
			{Label: "#### Relationships", Kind: completionKindSnippet, Detail: "OOUX object subheading", InsertText: "#### Relationships\n\n- ${1:relationship}\n", InsertTextFormat: insertTextSnippet},
			{Label: "#### Calls to Action", Kind: completionKindSnippet, Detail: "OOUX object subheading", InsertText: "#### Calls to Action\n\n- ${1:action}\n", InsertTextFormat: insertTextSnippet},
		}
	case "eventstorm":
		return []completionItem{
			{Label: "eventstorm slice", Kind: completionKindSnippet, Detail: "event, command, actor, policy, and read model", InsertText: "### ${1:Slice}\n\n- event: ${2:event}\n- command: ${3:command}\n- actor: ${4:actor}\n- policy: ${5:policy}\n- read model: ${6:model}\n", InsertTextFormat: insertTextSnippet},
			{Label: "event field", Kind: completionKindText, Detail: "EventStorming event bullet", InsertText: "- event: "},
			{Label: "command field", Kind: completionKindText, Detail: "EventStorming command bullet", InsertText: "- command: "},
			{Label: "actor field", Kind: completionKindText, Detail: "EventStorming actor bullet", InsertText: "- actor: "},
			{Label: "policy field", Kind: completionKindText, Detail: "EventStorming policy bullet", InsertText: "- policy: "},
			{Label: "read model field", Kind: completionKindText, Detail: "EventStorming read model bullet", InsertText: "- read model: "},
		}
	case "contexts":
		return []completionItem{
			{Label: "context relationship", Kind: completionKindSnippet, Detail: "bounded context relationship", InsertText: "### ${1:Context} -> ${2:Context}\n\n- upstream: ${3:upstream}\n- downstream: ${4:downstream}\n- relationship: ${5:relationship}\n", InsertTextFormat: insertTextSnippet},
			{Label: "upstream field", Kind: completionKindText, Detail: "Context Map field", InsertText: "- upstream: "},
			{Label: "downstream field", Kind: completionKindText, Detail: "Context Map field", InsertText: "- downstream: "},
			{Label: "relationship field", Kind: completionKindText, Detail: "Context Map field", InsertText: "- relationship: "},
		}
	case "domain-story":
		return []completionItem{
			{Label: "domain story step", Kind: completionKindSnippet, Detail: "actor action work object", InsertText: "- ${1:actor} ${2:does} ${3:work object}\n", InsertTextFormat: insertTextSnippet},
			{Label: "actor field", Kind: completionKindText, Detail: "Domain Storytelling actor", InsertText: "- actor: "},
			{Label: "work object field", Kind: completionKindText, Detail: "Domain Storytelling work object", InsertText: "- work object: "},
			{Label: "activity field", Kind: completionKindText, Detail: "Domain Storytelling activity", InsertText: "- activity: "},
		}
	case "example-mapping":
		return []completionItem{
			{Label: "rule field", Kind: completionKindText, Detail: "Example Mapping rule bullet", InsertText: "- rule: "},
			{Label: "example field", Kind: completionKindText, Detail: "Example Mapping example bullet", InsertText: "- example: "},
			{Label: "question field", Kind: completionKindText, Detail: "Example Mapping question bullet", InsertText: "- question: "},
		}
	case "jobs":
		return []completionItem{
			{Label: "job story", Kind: completionKindSnippet, Detail: "When/I want/so I can story", InsertText: "- WHEN ${1:situation}\n  I WANT ${2:motivation}\n  SO I CAN ${3:outcome}\n", InsertTextFormat: insertTextSnippet},
			{Label: "WHEN job field", Kind: completionKindText, Detail: "Job Story situation", InsertText: "- WHEN "},
			{Label: "I WANT field", Kind: completionKindText, Detail: "Job Story motivation", InsertText: "  I WANT "},
			{Label: "SO I CAN field", Kind: completionKindText, Detail: "Job Story outcome", InsertText: "  SO I CAN "},
		}
	case "opportunity-tree":
		return []completionItem{
			{Label: "opportunity field", Kind: completionKindText, Detail: "Opportunity Solution Tree bullet", InsertText: "- opportunity: "},
			{Label: "solution field", Kind: completionKindText, Detail: "Opportunity Solution Tree bullet", InsertText: "- solution: "},
			{Label: "experiment field", Kind: completionKindText, Detail: "Opportunity Solution Tree bullet", InsertText: "- experiment: "},
		}
	case "journey":
		return []completionItem{
			{Label: "stage block", Kind: completionKindSnippet, Detail: "Journey stage with common fields", InsertText: "### ${1:Stage}\n\n- action: ${2:action}\n- mindset: ${3:mindset}\n- emotion: ${4:emotion}\n- opportunity: ${5:opportunity}\n", InsertTextFormat: insertTextSnippet},
			{Label: "action field", Kind: completionKindText, Detail: "Journey stage field", InsertText: "- action: "},
			{Label: "mindset field", Kind: completionKindText, Detail: "Journey stage field", InsertText: "- mindset: "},
			{Label: "emotion field", Kind: completionKindText, Detail: "Journey stage field", InsertText: "- emotion: "},
			{Label: "opportunity field", Kind: completionKindText, Detail: "Journey stage field", InsertText: "- opportunity: "},
		}
	case "service-blueprint":
		return []completionItem{
			{Label: "blueprint step", Kind: completionKindSnippet, Detail: "Service Blueprint step", InsertText: "### ${1:Step}\n\n- evidence: ${2:evidence}\n- customer action: ${3:action}\n- frontstage: ${4:frontstage}\n- backstage: ${5:backstage}\n- support: ${6:support}\n", InsertTextFormat: insertTextSnippet},
			{Label: "blueprint lane", Kind: completionKindText, Detail: "Service Blueprint lane", InsertText: "- lane: "},
			{Label: "evidence field", Kind: completionKindText, Detail: "Service Blueprint field", InsertText: "- evidence: "},
			{Label: "customer action field", Kind: completionKindText, Detail: "Service Blueprint field", InsertText: "- customer action: "},
			{Label: "frontstage field", Kind: completionKindText, Detail: "Service Blueprint field", InsertText: "- frontstage: "},
			{Label: "backstage field", Kind: completionKindText, Detail: "Service Blueprint field", InsertText: "- backstage: "},
			{Label: "support field", Kind: completionKindText, Detail: "Service Blueprint field", InsertText: "- support: "},
		}
	case "stratmd":
		return []completionItem{
			{Label: "strategy objective", Kind: completionKindSnippet, Detail: "StratMD objective with goals and actions", InsertText: "## Objective\n\n${1:objective}\n\n## Goals\n\n- ${2:goal}\n\n## Actions\n\n- ${3:action}\n\n## Changelog\n\n- ${4:change}\n", InsertTextFormat: insertTextSnippet},
			{Label: "goal field", Kind: completionKindText, Detail: "StratMD goal", InsertText: "- goal: "},
			{Label: "risk field", Kind: completionKindText, Detail: "StratMD risk", InsertText: "- risk: "},
			{Label: "action field", Kind: completionKindText, Detail: "StratMD action", InsertText: "- action: "},
		}
	case "magi":
		return []completionItem{
			{Label: "magi typed block", Kind: completionKindSnippet, Detail: "MAGI typed fenced block", InsertText: "```yaml ${1:type}\n${2:key}: ${3:value}\n```\n", InsertTextFormat: insertTextSnippet},
			{Label: "relationship field", Kind: completionKindText, Detail: "MAGI relationship", InsertText: "- relationship: "},
			{Label: "artifact field", Kind: completionKindText, Detail: "MAGI artifact", InsertText: "- artifact: "},
		}
	default:
		return nil
	}
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
	case "domain-story":
		return "Domain Storytelling extension: actors, activities, work objects, and story steps."
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
	case "stratmd":
		return "StratMD extension: strategic intent, objectives, goals, risks, actions, and changelog."
	case "magi":
		return "MAGI extension: typed Markdown blocks, artifacts, and relationships."
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

func (s *Server) hoverAt(uri, text string, pos position) string {
	if name, ok := plainNameAt(text, pos); ok {
		if sym, ok := s.objectSymbol(name); ok {
			if row, ok := s.objectRow(sym.Canon); ok && row.Detail != "" {
				return row.Canon + "\n\n" + row.Detail + "\n\nSource: " + path.Base(row.URI)
			}
			return sym.Canon + "\n\nSource: " + path.Base(sym.URI)
		}
	}
	return hoverAt(uri, text, pos)
}

func (s *Server) objectSymbol(name string) (indexedSymbol, bool) {
	norm := normName(canonicalName(name))
	for _, sym := range s.indexedSymbols() {
		if sym.Norm == norm && sym.Role == symbolObject && !sym.Reference {
			return sym, true
		}
	}
	return indexedSymbol{}, false
}

func (s *Server) objectRow(name string) (indexedSymbol, bool) {
	norm := normName(canonicalName(name))
	for _, sym := range s.indexedSymbols() {
		if sym.Norm == norm && sym.Role == symbolObjectRow {
			return sym, true
		}
	}
	return indexedSymbol{}, false
}

func lineAt(text string, line int) string {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	return lines[line]
}

func relativeLabel(fromURI, toURI string) string {
	from, ok1 := pathFromURI(fromURI)
	to, ok2 := pathFromURI(toURI)
	if !ok1 || !ok2 || from == "" || to == "" {
		return ""
	}
	rel, err := filepath.Rel(filepath.Dir(from), to)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

func slug(s string) string {
	return strings.ReplaceAll(normName(s), " ", "-")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
