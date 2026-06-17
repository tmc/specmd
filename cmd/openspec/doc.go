// Command openspec parses, validates, and serves OpenSpec Markdown artifacts.
//
// The command intentionally exposes a small surface:
//
//	openspec validate [path]
//	openspec lsp
//
// The validate command accepts an openspec project directory, an OKF bundle, a
// spec.md file, an OKF concept file, or a change directory. It reports warnings
// without failing unless -strict is set. The lsp command runs the same stdio
// language server as openspec-lsp.
package main
