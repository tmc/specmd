# OpenSpec LSP Editor Setup

`openspec-lsp` is a small stdio Language Server Protocol server for OpenSpec
Markdown files. It does not manage workspaces or install editor plugins.

## Run

From this repository:

```sh
go run ./cmd/openspec-lsp
```

For day-to-day editor use:

```sh
go install ./cmd/openspec-lsp
```

The server reads open Markdown documents from the editor. Paths are used to
detect artifact families, so files should live under paths such as:

```text
openspec/specs/<name>/spec.md
openspec/changes/<id>/proposal.md
openspec/extensions/<family>/<name>.md
openspec/changes/<id>/extensions/<family>/<name>.md
```

## Neovim

Neovim can start a stdio LSP server directly. Put this in `init.lua` or a
project-local Lua file you source while testing:

```lua
vim.api.nvim_create_autocmd("FileType", {
  pattern = "markdown",
  callback = function(args)
    local root = vim.fs.root(args.buf, { "openspec", ".git" }) or vim.fn.getcwd()
    vim.lsp.start({
      name = "openspec-lsp",
      cmd = { "openspec-lsp" },
      root_dir = root,
    })
  end,
})
```

To run from a checkout without installing:

```lua
cmd = { "go", "run", "./cmd/openspec-lsp" }
```

Use the installed form when opening `docs/editor-demo`, because `go run
./cmd/openspec-lsp` expects the repository root as the current directory.

## VS Code

VS Code needs a small extension wrapper to launch arbitrary stdio LSP servers.
The wrapper should start:

```json
{
  "command": "openspec-lsp",
  "args": [],
  "transport": "stdio",
  "documentSelector": [{ "language": "markdown", "scheme": "file" }]
}
```

Keep the wrapper thin. The server already handles initialization, text sync,
diagnostics, symbols, completion, and hover.

## Demo

Open `docs/editor-demo` as the editor workspace after installing
`openspec-lsp`.

Files to try:

- `openspec/specs/auth/spec.md` reports a missing `## Requirements` section.
- `openspec/specs/billing/spec.md` is clean and shows Markdown document
  symbols.
- `openspec/extensions/example-mapping/auth.md` reports missing `## Story` and
  `## Questions`.
- `openspec/extensions/opportunity-tree/login.md` is clean; completion offers
  opportunity-tree sections and hover describes extension headings.
- `openspec/extensions/ooux/model.md` is clean; completion offers OOUX object
  blocks and object subheadings.
- `openspec/extensions/contexts/map.md`, `domain-story/model.md`,
  `eventstorm/model.md`, `jobs/stories.md`, `journey/login.md`,
  `service-blueprint/login.md`, `stratmd/strategy.md`, and
  `magi/context.md` exercise the other extension completion families.

Useful editor checks:

- diagnostics appear after opening a file
- diagnostics clear after adding the missing section
- completion after `## ` includes missing required sections first
- completion offers requirement/scenario blocks and scenario fields
- extension files offer family-specific fields, blocks, and subheadings
- Obsidian-style links such as `[[OOUX model#Objects]]` jump to opened
  extension documents and headings
- references from a linked heading show other opened documents that link to it
- document symbols show headings
- hover on known headings returns OpenSpec-specific text

## Troubleshooting

- If no LSP starts, check that the filetype is `markdown`.
- If `openspec-lsp` is not found, run `go install ./cmd/openspec-lsp` and make
  sure `$GOBIN` or `$GOPATH/bin` is on `PATH`.
- If `go run ./cmd/openspec-lsp` fails, run it from the repository root.
- If diagnostics do not appear, check that the file path contains an
  `openspec/specs`, `openspec/changes`, or `openspec/extensions` segment.
- Stdio servers do not print logs to the terminal used by the editor; use the
  editor's LSP log when debugging startup.

## Feature Coverage

Supported now:

- `initialize`
- `shutdown` and `exit`
- `textDocument/didOpen`
- `textDocument/didChange`
- `textDocument/didClose`
- `textDocument/publishDiagnostics`
- `textDocument/documentSymbol`
- `textDocument/completion`
- `textDocument/hover`
- `textDocument/definition`
- `textDocument/references`

Current behavior:

- diagnostics check required sections for specs, proposals, and documented
  extension families
- heading diagnostics catch non-breaking spaces after Markdown `#`
- completion suggests missing required sections before general OpenSpec
  headings, plus requirement blocks, scenario fields, delta blocks, and
  extension-specific fields or subheadings
- document symbols are Markdown headings
- hover describes known sections or the document extension family
- definitions and references resolve Obsidian-style `[[target]]`,
  `[[target#heading]]`, `[[#heading]]`, and aliased `[[target|label]]` links
  among currently opened documents

Useful later:

- parser-backed diagnostics that reuse more of the public validation rules
- more cursor-aware completions
- workspace configuration for enabling or disabling extension families
- file watching if project-level validation becomes necessary

Out of scope for this package:

- TypeScript CLI plumbing
- telemetry
- UI or dashboard features
- package-manager behavior
- broad workspace management
- archive/apply flows
- command generation

Not implemented:

- definition
- references
- rename
- formatting
- code actions
- semantic tokens
- workspace folders
- watched files
- editor-specific server behavior
