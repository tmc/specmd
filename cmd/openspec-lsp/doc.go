// Command openspec-lsp runs a small stdio LSP server for OpenSpec Markdown.
//
// The server is intended for editor integrations. It reports diagnostics for
// OpenSpec sections and local Markdown graph issues, indexes workspace
// Markdown files for definitions and references, returns document and workspace
// symbols, and offers section-heading completions.
package main
