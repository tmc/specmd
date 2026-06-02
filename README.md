# openspec

Package openspec parses and validates the small Markdown artifact format used
by OpenSpec specs and change deltas.

This is a minimal Go implementation. It does not implement the TypeScript CLI,
workspace management, command generation, or archive flows.

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

OKF parsing reads Open Knowledge Format v0.1 bundles as Markdown concept files
with YAML front matter:

```go
bundle, err := openspec.ParseOKFBundle("testdata/okf")
if err != nil {
	log.Fatal(err)
}
if err := openspec.ValidateOKFBundle(bundle); err != nil {
	log.Fatal(err)
}
```

## CLI

The `openspec` command provides small, deterministic operations over the Go
package:

```sh
go run ./cmd/openspec validate openspec
go run ./cmd/openspec validate -json openspec
```

`validate` accepts an openspec project directory, an OKF bundle directory, a
`spec.md` file, an OKF concept file, or a change directory with `proposal.md`.
Warnings and info are reported but do not fail the command unless `-strict` is
set.

## LSP

The `openspec-lsp` command runs a small stdio Language Server Protocol server
for Markdown editors:

```sh
go run ./cmd/openspec-lsp
```

It supports open/change/close text sync, diagnostics, document symbols,
hover, and section-heading completions for OpenSpec specs, changes, and the
extension fixture families documented under `docs/extension-landscape.md`.
