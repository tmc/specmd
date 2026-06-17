package openspec

// A Spec describes current behavior for one OpenSpec capability.
type Spec struct {
	Name         string
	Overview     string
	Requirements []Requirement
	Metadata     Metadata
}

// A Change describes proposed behavior changes.
type Change struct {
	Name        string
	Why         string
	WhatChanges string
	Deltas      []Delta
	Extensions  []ExtensionRef
	Metadata    Metadata
}

// Metadata records OpenSpec format metadata when it is known.
type Metadata struct {
	Version    string
	Format     string
	SourcePath string
}

// A DeltaOperation identifies how a change affects a spec.
type DeltaOperation string

const (
	Added    DeltaOperation = "ADDED"
	Modified DeltaOperation = "MODIFIED"
	Removed  DeltaOperation = "REMOVED"
	Renamed  DeltaOperation = "RENAMED"
)

// A Delta describes one operation against one spec.
type Delta struct {
	Spec         string
	Operation    DeltaOperation
	Description  string
	Requirements []Requirement
	Renames      []Rename
	Metadata     Metadata
}

// A Project is the parsed artifact content under an openspec directory.
type Project struct {
	Specs      []Spec
	Changes    []Change
	Extensions []ExtensionRef
}

// An ExtensionRef records one discovered extension artifact.
type ExtensionRef struct {
	Name       string
	SourcePath string
}

// Rename records the old and new names for a renamed requirement.
type Rename struct {
	From string
	To   string
}

// A Requirement describes one behavior and its verification scenarios.
type Requirement struct {
	Name      string
	Text      string
	Scenarios []Scenario
}

// A Scenario is the raw Markdown body for one scenario.
type Scenario struct {
	Name    string
	RawText string
}

// An OKFBundle is a parsed Open Knowledge Format bundle.
type OKFBundle struct {
	Root     string
	Version  string
	Concepts []OKFConcept
	Invalid  []OKFInvalidConcept
	Index    []OKFReservedFile
	Logs     []OKFReservedFile
	Metadata Metadata
}

// An OKFInvalidConcept records a concept document whose frontmatter could
// not be parsed. ParseOKFBundle keeps these rather than failing the whole
// bundle; ValidateOKFBundle reports each as a conformance error.
type OKFInvalidConcept struct {
	ID         string
	SourcePath string
	Err        error
}

// An OKFConcept is one OKF concept document.
type OKFConcept struct {
	ID          string
	Type        string
	Title       string
	Description string
	Resource    string
	Tags        []string
	Timestamp   string
	FrontMatter []OKFField
	Body        string
	Metadata    Metadata
}

// An OKFField is one frontmatter field.
type OKFField struct {
	Key    string
	Values []string
}

// An OKFReservedFile is a reserved OKF index.md or log.md file.
type OKFReservedFile struct {
	Name        string
	Body        string
	FrontMatter []OKFField
	Root        bool
	Metadata    Metadata
}
