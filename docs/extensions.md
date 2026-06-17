# Extension Design

Package `openspec` keeps the core OpenSpec artifact model small: specs,
changes, deltas, requirements, and scenarios, with validation findings
reported through the shared `specmd/validation` package. Extensions should add
typed artifact models around that core, not arbitrary fields inside
requirements.

Upstream OpenSpec extends workflows through project configuration and custom
schemas under `openspec/schemas/<name>/`. Its core parsed spec schema has only
`name`, `overview`, `requirements`, and optional metadata. That makes
extension artifacts the safest compatibility boundary for this Go package.

## Principles

Extensions should be:

- optional: parsing a spec must not require any extension package;
- typed when parsed: exported Go structs should describe extension data
  directly once a concrete extension format has stable fixtures;
- protobuf-friendly: prefer stable scalar fields, repeated fields, and
  enum-like strings over `map[string]any` in primary models;
- markdown-native: extension files should remain readable without generated
  tooling;
- side-effect free: extension parsing should not mutate the core `Project`;
- tolerant: unknown extension artifacts are preserved as paths or ignored by
  default, not treated as core validation failures.

The core package avoids a generic plugin registry. A registry invites a larger
API than the current package needs and makes protobuf mapping harder. Extension
discovery exists; typed extension parsing should come later, outside the core
parser, after formats stabilize.

## Artifact Convention

A project-level extension belongs under a named directory:

```text
openspec/
  extensions/
    ooux/
      model.md
```

A change-local extension belongs beside the change artifacts:

```text
openspec/
  changes/
    add-2fa/
      proposal.md
      specs/auth/spec.md
      extensions/ooux.md
```

This mirrors upstream's custom-schema separation: workflow extensions live
beside, not inside, core spec documents.

## Go Shape

The smallest discovery-only API is:

```go
type ExtensionRef struct {
    Name       string
    SourcePath string
}
```

That is enough to tell callers that extension artifacts exist. `ParseProject`
populates project-level refs from `openspec/extensions/**/*.md`; `ParseChangeDir`
populates change-local refs from `openspec/changes/<id>/extensions/**/*.md`.

If the package later parses a concrete extension, use a typed artifact wrapper
only after the file format is backed by fixtures:

```go
type ExtensionArtifact struct {
    Name       string
    SourcePath string
}
```

The current shape intentionally remains discovery-only. A future typed parser
can define its own payload type in a sibling package without forcing all
callers to depend on it.

## OOUX

OOUX is a good example of an extension artifact, but it is not an OpenSpec
core type and this package should not define an unofficial OOUX standard.
Treat OOUX files as discoverable extension artifacts until there are stable
fixtures and callers asking for typed parsing.

The external survey and possible future shapes live in
`docs/extension-landscape.md`.

## What Not To Do

Do not add `Extensions map[string]any` to `Spec`, `Requirement`, or `Delta`.
It is hard to document, hard to validate, awkward in protobuf, and would imply
that upstream core specs accept arbitrary fields when they do not.

Do not require front matter for OOUX. Front matter is useful for metadata, but
the model itself should remain visible in Markdown headings and lists if a
future OOUX parser is added.

Do not make OOUX part of `ValidateSpec`. A spec should remain valid even when no
OOUX artifact exists. Cross-artifact consistency checks can be a separate
project-level report.
