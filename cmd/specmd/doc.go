// Command specmd parses, validates, and serves structured-Markdown artifacts.
//
// The command intentionally exposes a small surface:
//
//	specmd validate [path]
//	specmd lsp
//
// The validate command accepts an OpenSpec project directory, an OKF bundle, a
// spec.md file, an OKF concept file, or a change directory. It reports warnings
// without failing unless -strict is set. The lsp command runs the same stdio
// language server as specmd-lsp.
package main
