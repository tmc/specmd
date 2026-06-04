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

A reusable setup file is available at `editors/nvim/lua/openspec_lsp.lua`.
It attaches only to Markdown buffers whose path or root contains `openspec`.
For a quick local smoke test:

```lua
dofile("/path/to/openspec/editors/nvim/openspec-lsp.lua")
```

Useful checks:

```vim
:set filetype?
:checkhealth vim.lsp
:LspInfo
:lua vim.lsp.buf.code_action()
:lua vim.lsp.buf.document_symbol()
```

## VS Code

VS Code needs a small extension wrapper to launch arbitrary stdio LSP servers.
This repository includes a thin wrapper at `editors/vscode`. It starts the
existing `openspec-lsp` binary and leaves parsing, diagnostics, completion, and
navigation in the Go server.

The wrapper exposes:

- `openspec.lsp.path`: path to `openspec-lsp`, defaulting to `openspec-lsp` on
  `PATH`;
- `openspec.lsp.trace.server`: `off`, `messages`, or `verbose`;
- `OpenSpec: Validate Project`;
- `OpenSpec: Insert Requirement`;
- `OpenSpec: Open Extension Model`.

To test locally:

```sh
cd editors/vscode
npm install
npm run compile
```

Then open the folder in VS Code and run the extension host. The extension uses
this server shape:

```json
{
  "command": "openspec-lsp",
  "args": [],
  "transport": "stdio",
  "documentSelector": [{ "language": "markdown", "scheme": "file" }]
}
```

Keep the wrapper thin. It should not reimplement OpenSpec parsing or validation
in TypeScript.

## Zed

Zed requires a language extension to attach a custom language server to
Markdown. This repository includes a local dev extension at
`zed/openspec-lsp`.

Install the server binary:

```sh
go install ./cmd/openspec-lsp
```

Then install the dev extension in Zed:

1. Open the command palette.
2. Run `zed: install dev extension`.
3. Select `zed/openspec-lsp`.
4. Restart the OpenSpec LSP if Zed asks, or reopen the Markdown file.

Zed compiles Rust extensions to WebAssembly. If installation reports a missing
WASI target, run:

```sh
rustup target add wasm32-wasip2
```

The project-local `.zed/settings.json` enables the server for Markdown:

```json
{
  "languages": {
    "Markdown": {
      "language_servers": ["openspec-lsp", "..."]
    }
  }
}
```

The extension first honors `lsp.openspec-lsp.binary.path`, then falls back to
`openspec-lsp` on `PATH`. If Zed cannot find the server, add a project-local
override:

```json
{
  "lsp": {
    "openspec-lsp": {
      "binary": {
        "path": "/absolute/path/to/openspec-lsp"
      }
    }
  }
}
```

Project-local tasks live in `.zed/tasks.json`:

- `openspec: validate current file`
- `openspec: validate project`
- `openspec: test lsp`
- `openspec: test all`

The current repository does not have a separate OpenSpec validation CLI, so the
validation tasks run the Go test suite as the project-level verification gate.

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
- code actions offer section insertion and safe heading fixes
- document links expose `[[wiki links]]` and local `openspec/...` paths
- folding ranges fold Markdown sections
- selection ranges expand from heading to document
- workspace symbols find opened specs, requirements, and extension headings
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
- In Zed, use `zed: install dev extension` with `zed/openspec-lsp`. If Zed
  cannot compile the extension but shell `cargo build --target wasm32-wasip2`
  succeeds, install from a clean shell or use a locally built dev extension
  copy for a manual smoke test. Do not commit generated `.wasm` files.
- Zed smoke evidence for this branch was captured outside the repo at
  `/tmp/openspec-zed-lsp.png`.

## Feature Coverage

| LSP feature | Status | Notes |
| --- | --- | --- |
| `initialize` | now | advertises the server capabilities below |
| `shutdown`, `exit` | now | stdio lifecycle |
| `textDocument/didOpen` | now | stores open Markdown documents |
| `textDocument/didChange` | now | full-text sync |
| `textDocument/didClose` | now | clears diagnostics |
| `textDocument/publishDiagnostics` | now | section and heading diagnostics |
| `textDocument/documentSymbol` | now | Markdown headings |
| `textDocument/completion` | now | sections, snippets, fields, extension blocks |
| `textDocument/hover` | now | known sections and extension families |
| `textDocument/definition` | now | opened wiki-link targets |
| `textDocument/references` | now | opened wiki-link references |
| `textDocument/codeAction` | now | safe section insertion and heading fixes |
| `textDocument/documentLink` | now | wiki links and local `openspec/...` paths |
| `workspace/symbol` | now | opened documents and headings |
| `textDocument/foldingRange` | now | Markdown heading regions |
| `textDocument/selectionRange` | now | heading-to-document expansion |
| `workspace/configuration` | useful later | only if editor-neutral settings become necessary |
| `textDocument/rename` | useful later | requirement names and wiki-link targets |
| `textDocument/formatting` | useful later | canonical skeleton cleanup |
| semantic tokens | useful later | only if it materially improves readability |
| watched files | useful later | needed only for project-level validation |
| workspace folders | useful later | avoid broad workspace management for now |

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
- code actions return workspace edits for missing `## Purpose`,
  missing `## Requirements`, common `## Requiement` typo fixes, and
  requirement/scenario skeleton insertion
- document links return ranges for wiki links and literal local
  `openspec/...` paths
- workspace symbols search currently opened documents, not the whole filesystem

Useful later:

- more cursor-aware completions
- workspace configuration for enabling or disabling extension families
- file watching if project-level validation becomes necessary
- rename support for requirements and wiki-link targets
- formatting support for normalized section skeletons

Out of scope for this package:

- TypeScript CLI plumbing
- telemetry
- UI or dashboard features
- package-manager behavior
- broad workspace management
- archive/apply flows
- command generation
- editor-specific server behavior
