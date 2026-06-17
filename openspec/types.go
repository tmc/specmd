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
