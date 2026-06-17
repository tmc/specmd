// Package openspec parses and validates OpenSpec Markdown artifacts.
//
// Specs use a Purpose section and a Requirements section. Each requirement is
// introduced by a level-three "Requirement:" heading and has one or more
// level-four "Scenario:" headings. [ParseSpec] reads a single spec; [ValidateSpec]
// checks it.
//
// Changes use Why and What Changes sections plus delta specs. Delta specs group
// requirements under "ADDED", "MODIFIED", "REMOVED", or "RENAMED" requirement
// headings. [ParseChange] reads a change; [ValidateChange] checks it.
//
// [ParseProject] reads the usual specs/ and changes/ layout and also discovers
// extension Markdown artifacts under extensions/ and changes/<name>/extensions/.
// Extensions are returned as [ExtensionRef] values; the package does not parse
// extension payloads.
//
// Conformance findings are reported as a
// [github.com/tmc/specmd/validation.Report].
package openspec
