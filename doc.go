// Package openspec parses and validates OpenSpec Markdown artifacts.
//
// Specs use a Purpose section and Requirements section. Each requirement is
// introduced by a level-three "Requirement:" heading and has one or more
// level-four "Scenario:" headings.
//
// Changes use Why and What Changes sections plus delta specs. Delta specs group
// requirements under "ADDED", "MODIFIED", "REMOVED", or "RENAMED" requirement
// headings.
//
// Project parsing also discovers extension Markdown artifacts under
// extensions/ and changes/<name>/extensions/. Extensions are returned as
// ExtensionRef values; the package does not parse extension payloads.
//
// OKF support parses Open Knowledge Format v0.1 bundles as Markdown concept
// files with YAML frontmatter. Validation enforces the hard conformance rules
// while treating optional fields, unknown types, unknown keys, and broken links
// permissively.
package openspec
