// Command specmd-lsp runs a small stdio LSP server for structured Markdown.
//
// The server is intended for editor integrations. It reports diagnostics for
// OpenSpec sections, OKF concept conformance, and local Markdown graph issues,
// indexes workspace Markdown files for definitions and references, returns
// document and workspace symbols, supports conservative rename/code-action
// edits, and offers section-heading, link, and object completions.
package main
