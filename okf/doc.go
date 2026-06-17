// Package okf parses and validates Open Knowledge Format v0.1 bundles.
//
// An OKF bundle is a directory of Markdown concept documents with YAML
// frontmatter, plus reserved index.md and log.md files. [ParseBundle] reads a
// bundle directory; [ParseConcept] reads one concept document.
//
// Validation enforces the hard conformance rules while treating optional
// fields, unknown types, unknown keys, and broken links permissively.
// Conformance findings are reported as a
// [github.com/tmc/specmd/validation.Report].
package okf
