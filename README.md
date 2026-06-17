# specmd

specmd is Go tooling for structured-Markdown authoring, validation, and
editor integration. It groups several Markdown families that share one
parse-and-validate shape:

- **OpenSpec** specs and change deltas (package
  `github.com/tmc/specmd/openspec`).
- **OKF** (Open Knowledge Format v0.1) concept bundles (package
  `github.com/tmc/specmd/okf`).
- A stdio **language server** (`cmd/specmd-lsp`) that serves both to editors,
  alongside the design-modeling extension families documented under
  `docs/extension-landscape.md`.

This is a minimal Go implementation. It does not implement the TypeScript CLI,
workspace management, command generation, or archive flows.

## OpenSpec

```go
spec, err := openspec.ParseSpec("auth", strings.NewReader(markdown))
if err != nil {
	log.Fatal(err)
}
if err := openspec.ValidateSpec(spec); err != nil {
	log.Fatal(err)
}
```

Project parsing reads the usual `openspec/specs/...` and
`openspec/changes/...` layout and discovers extension Markdown files without
parsing their contents:

```go
project, err := openspec.ParseProject("openspec")
if err != nil {
	log.Fatal(err)
}
for _, ref := range project.Extensions {
	fmt.Println(ref.Name, ref.SourcePath)
}
```

## OKF

OKF parsing reads Open Knowledge Format v0.1 bundles as Markdown concept files
with YAML front matter:

```go
bundle, err := okf.ParseBundle("testdata/okf")
if err != nil {
	log.Fatal(err)
}
if err := okf.ValidateBundle(bundle); err != nil {
	log.Fatal(err)
}
```

Both families report conformance findings as a
`github.com/tmc/specmd/validation.Report`; the umbrella `specmd` package
re-exports those types as `specmd.ValidationReport`, `specmd.ValidationIssue`,
and so on for callers that want one set of names.

## CLI

The `specmd` command provides small, deterministic operations over the Go
packages:

```sh
go run ./cmd/specmd validate openspec
go run ./cmd/specmd validate -json openspec
go run ./cmd/specmd lsp
```

`validate` accepts an OpenSpec project directory, an OKF bundle directory, a
`spec.md` file, an OKF concept file, or a change directory with `proposal.md`.
Warnings and info are reported but do not fail the command unless `-strict` is
set.

## LSP

The `specmd-lsp` command runs a small stdio Language Server Protocol server
for Markdown editors:

```sh
go run ./cmd/specmd-lsp
```

It supports open/change/close text sync, diagnostics, document symbols,
hover, and completions for OpenSpec specs, changes, OKF concepts, and the
extension fixture families documented under `docs/extension-landscape.md`.

See `docs/editor-lsp.md` for Neovim setup, the VS Code wrapper, the Zed
extension, demo workspace files, and current LSP feature coverage.
