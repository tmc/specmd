# specmd LSP Editor Setup

`specmd-lsp` is a small stdio Language Server Protocol server for OpenSpec and
OKF Markdown files. It does not manage workspaces or install editor plugins.

## Run

From this repository:

```sh
go run ./cmd/specmd-lsp
```

For day-to-day editor use:

```sh
go install ./cmd/specmd-lsp
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
      name = "specmd-lsp",
      cmd = { "specmd-lsp" },
      root_dir = root,
    })
  end,
})
```

To run from a checkout without installing:

```lua
cmd = { "go", "run", "./cmd/specmd-lsp" }
```

Use the installed form when opening `docs/editor-demo`, because `go run
./cmd/specmd-lsp` expects the repository root as the current directory.

A reusable setup file is available at `editor-support/nvim/lua/specmd_lsp.lua`.
It attaches only to Markdown buffers whose path or root contains `openspec`.
For a quick local smoke test:

```lua
dofile("/path/to/specmd/editor-support/nvim/specmd-lsp.lua")
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
This repository includes a thin wrapper at `editor-support/vscode`. It starts the
existing `specmd-lsp` binary and leaves parsing, diagnostics, completion, and
navigation in the Go server.

The wrapper exposes:

- `specmd.lsp.path`: path to `specmd-lsp`, defaulting to `specmd-lsp` on
  `PATH`;
- `specmd.lsp.trace.server`: `off`, `messages`, or `verbose`;
- `OpenSpec: Validate Project`;
- `OpenSpec: Insert Requirement`;
- `OpenSpec: Open Extension Model`.

To test locally:

```sh
cd editor-support/vscode
npm install
npm run compile
```

Then open the folder in VS Code and run the extension host. The extension uses
this server shape:

```json
{
  "command": "specmd-lsp",
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
`editor-support/zed`.

Install the server binary:

```sh
go install ./cmd/specmd-lsp
```

Then install the dev extension in Zed:

1. Open the command palette.
2. Run `zed: install dev extension`.
3. Select `editor-support/zed`.
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
      "language_servers": ["specmd-lsp", "..."]
    }
  }
}
```

The extension first honors `lsp.specmd-lsp.binary.path`, then falls back to
`specmd-lsp` on `PATH`. If Zed cannot find the server, add a project-local
override:

```json
{
  "lsp": {
    "specmd-lsp": {
      "binary": {
        "path": "/absolute/path/to/specmd-lsp"
      }
    }
  }
}
```

Project-local tasks live in `.zed/tasks.json`:

- `specmd: validate current file`
- `specmd: validate project`
- `specmd: test lsp`
- `specmd: test all`

These tasks run the Go test suite as the project-level verification gate
rather than shelling out to `specmd validate`, so they exercise the parser,
validator, and LSP together.

## Demo

Open `docs/editor-demo` as the editor workspace after installing
`specmd-lsp`.

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
- If `specmd-lsp` is not found, run `go install ./cmd/specmd-lsp` and make
  sure `$GOBIN` or `$GOPATH/bin` is on `PATH`.
- If `go run ./cmd/specmd-lsp` fails, run it from the repository root.
- If diagnostics do not appear, check that the file path contains an
  `openspec/specs`, `openspec/changes`, or `openspec/extensions` segment.
- Stdio servers do not print logs to the terminal used by the editor; use the
  editor's LSP log when debugging startup.
- In Zed, use `zed: install dev extension` with `editor-support/zed`. If Zed
  cannot compile the extension but shell `cargo build --target wasm32-wasip2`
  succeeds, install from a clean shell or use a locally built dev extension
  copy for a manual smoke test. Do not commit generated `.wasm` files.
- Zed smoke evidence for this branch was captured outside the repo at
  `/tmp/specmd-zed-lsp.png`.

## Feature Coverage

| LSP feature | Status | Notes |
| --- | --- | --- |
| `initialize` | now | advertises the server capabilities below |
| `shutdown`, `exit` | now | stdio lifecycle |
| `textDocument/didOpen` | now | stores open Markdown documents; open buffers override indexed files |
| `textDocument/didChange` | now | full-text sync and index refresh |
| `textDocument/didClose` | now | clears diagnostics |
| `textDocument/publishDiagnostics` | now | section, heading, validation, OKF concept, and graph-quality diagnostics |
| `textDocument/documentSymbol` | now | Markdown headings |
| `textDocument/completion` | now | sections, snippets, fields, extension blocks, OKF front-matter keys, indexed link/object names |
| `textDocument/hover` | now | known sections, extension families, OKF concept summaries, and indexed OOUX object details |
| `textDocument/definition` | now | workspace Markdown/wiki links and known object names |
| `textDocument/references` | now | workspace Markdown/wiki links and known object references |
| `textDocument/rename` | now | OOUX object names and explicit link targets with conservative workspace edits |
| `textDocument/prepareRename` | now | returns the current rename range when supported |
| `textDocument/codeAction` | now | section insertion, heading fixes, link-target fixes, and reference-definition cleanup |
| `textDocument/codeLens` | now | reference counts for indexed headings and OOUX objects |
| `textDocument/inlayHint` | now | reference-count hints for indexed headings and OOUX objects |
| `textDocument/documentLink` | now | Markdown links, wiki links, and local `openspec/...` paths |
| `workspace/symbol` | now | indexed documents, headings, specs, changes, extensions, and OOUX objects |
| `workspace/didChangeWatchedFiles` | now | marks the workspace index dirty for lazy refresh |
| `textDocument/foldingRange` | now | Markdown heading regions |
| `textDocument/selectionRange` | now | heading-to-document expansion |
| `workspace/configuration` | useful later | only if editor-neutral settings become necessary |
| `textDocument/formatting` | useful later | canonical skeleton cleanup |
| semantic tokens | useful later | only if it materially improves readability |
| workspace folders | useful later | single-root indexing is enough for now |

Current behavior:

- diagnostics check required sections for specs, proposals, and documented
  extension families
- heading diagnostics catch non-breaking spaces after Markdown `#`
- graph diagnostics warn about broken local Markdown links, missing heading
  fragments, ambiguous link targets, duplicate OOUX object names, duplicate
  catalog rows without object cards, object cards without catalog rows,
  catalog/detail status mismatches, referenced objects missing from the
  catalog, and current objects missing CTA rows
- completion suggests missing required sections before general OpenSpec
  headings, plus requirement blocks, scenario fields, delta blocks, and
  extension-specific fields or subheadings
- completion also suggests indexed Markdown files/headings while typing links
  and known OOUX objects, domains, statuses, and Markdown tags in appropriate
  contexts
- OKF concept documents (front-matter Markdown files outside the OpenSpec spec,
  change, and extension trees, excluding reserved `index.md`/`log.md`) report
  the Open Knowledge Format v0.1 conformance issues: a hard error for a missing
  `type` and advisories for missing `title`/`description`, anchored on the
  relevant front-matter line
- completion offers OKF concept front-matter keys (`type`, `title`,
  `description`, `resource`, `tags`, `timestamp`) while editing front matter,
  and hover summarizes the concept's type and recognized fields
- document symbols are Markdown headings
- hover describes known sections, the document extension family, or an indexed
  OOUX object with its catalog definition/status/domain when available
- hover on a local link previews the target document heading and source file
- hover near YAML front matter shows indexed metadata such as title, status,
  owner, domain, and last reviewed date
- definitions and references resolve Obsidian-style `[[target]]`,
  `[[target#heading]]`, `[[#heading]]`, and aliased `[[target|label]]` links
  across the indexed workspace
- definitions and references resolve standard relative Markdown links such as
  `[map](../00-object-catalog.md#cross-tier-object-map)`
- definitions and references resolve reference-style Markdown links such as
  `[catalog][cat]` with `[cat]: 00-object-catalog.md#objects`
- definitions on known OOUX object names such as `Variant` prefer the object
  detail heading, then the catalog row, then other heading matches
- rename edits update known OOUX object names across object cards, catalog
  rows, matrices, and other structured mentions
- code actions return workspace edits for missing `## Purpose`,
  missing `## Requirements`, common `## Requiement` typo fixes, and
  requirement/scenario skeleton insertion
- code actions can create a missing heading target, update a stale heading
  fragment to the document heading, extract an inline Markdown link into a
  reference definition, and remove duplicate or unused reference definitions
- document links return ranges for Markdown links, wiki links, and literal local
  `openspec/...` paths
- code lens and inlay hints expose reference counts without adding a graph UI
- workspace symbols search indexed Markdown files under the LSP root

## Workspace Index

`specmd-lsp` builds a small internal index of Markdown files under the LSP
root supplied during `initialize`. It skips `.git`, `node_modules`, build
outputs, generated editor outputs, binary files, invalid UTF-8, and files over
1 MiB. The index stores only text-derived facts:

- document symbols and artifact family
- Markdown headings and normalized heading slugs
- standard Markdown links and Obsidian-style wiki links
- reference-style Markdown links and reference definitions
- OpenSpec specs, changes, deltas, and extension files by path
- OOUX object detail headings under `objects/`
- OOUX object catalog rows in `00-object-catalog.md`
- structured object mentions in Markdown tables and simple Mermaid edge lines
- YAML-style front matter fields and Markdown tags

Open buffers are authoritative. When an editor sends `didOpen` or `didChange`,
the in-memory document replaces the on-disk snapshot for navigation,
completion, hover, and diagnostics.

Definition ranking is intentionally small:

1. explicit Markdown or wiki link target
2. exact OOUX object detail heading
3. OOUX object catalog row
4. other matching heading

References prefer explicit links, then structured object mentions. Plain prose
matches are not treated as references unless they are represented as known
object symbols in tables, catalog rows, object headings, or simple Mermaid
edges.

## Design References

The implementation borrows only small, portable ideas from nearby Markdown LSP
projects:

- Marksman shows that Markdown LSP value comes from workspace-wide links,
  heading definitions, references, completions, and structural diagnostics:
  <https://github.com/artempyanykh/marksman>.
- IWE treats a Markdown folder as a graph across editors including VS Code,
  Neovim, Zed, and Helix; `specmd-lsp` keeps the graph internal and avoids
  query languages, graph UI, MCP, or rewrite tooling:
  <https://iwe.md/docs/concepts/comparison/>.
- Foam's link-reference behavior reinforces standard Markdown portability; the
  server supports standard Markdown links first and wiki links as an additional
  syntax: <https://docs.swo.moe/foam-1/link-reference-definitions.html>.
- The VS Code Markdown language service is the generic baseline; OpenSpec and
  OOUX semantics stay in this server rather than editor wrappers:
  <https://github.com/microsoft/vscode-markdown-languageservice>.
- Zed's native Markdown page lists Tree-sitter support and no Markdown language
  server, so the Zed extension attaches this Go server for semantic features:
  <https://zed.dev/docs/languages/markdown>.

## Axion OOUX Demo

Open `/Users/tmc/go/src/github.com/EternisAI/tmc-personal-notes/axion-ooux-spec`
as the workspace. Expected checks:

- go-to-definition on `Variant` in `00-object-catalog.md` jumps to
  `objects/t6-quality-and-self-improvement.md`
- go-to-definition on
  `[cross-tier object map](../00-object-catalog.md#cross-tier-object-map)` in
  `matrices/relationship-map.md` jumps to the catalog heading
- references for `Variant` include the catalog, matrices, object detail file,
  and other structured mentions
- workspace symbols find `Variant`, `Forecast`, and `Thread`
- document links in `INDEX.md` target local Markdown files
- rename on `Variant` produces edits in the catalog, object card, and matrices
- reference counts appear for object headings in clients that show code lens or
  inlay hints
- graph diagnostics avoid irrelevant external URL checks and focus on local
  Markdown/spec navigation issues

Useful later:

- workspace configuration for enabling or disabling extension families
- formatting support for normalized section skeletons
- semantic tokens if they materially improve readability

Out of scope for this package:

- TypeScript CLI plumbing
- telemetry
- UI or dashboard features
- package-manager behavior
- broad workspace management
- archive/apply flows
- command generation
- editor-specific server behavior
- graph UI
- AI tooling
- full Markdown formatter
- broad personal-knowledge-management product features
