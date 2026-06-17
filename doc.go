// Package specmd is an umbrella for structured-Markdown authoring, validation,
// and language-server tooling.
//
// The module groups several Markdown families that share one parse-and-validate
// shape:
//
//   - [github.com/tmc/specmd/openspec] parses and validates OpenSpec specs and
//     changes.
//   - [github.com/tmc/specmd/okf] parses and validates Open Knowledge Format
//     bundles.
//   - [github.com/tmc/specmd/validation] defines the shared validation
//     vocabulary that both report findings in.
//
// The cmd/specmd command validates these artifacts; cmd/specmd-lsp serves them
// to editors over the Language Server Protocol.
//
// This package re-exports the [github.com/tmc/specmd/validation] types as
// aliases so callers can name a single validation vocabulary
// (specmd.ValidationReport, specmd.ValidationIssue, and so on) regardless of
// which family produced a report.
package specmd
