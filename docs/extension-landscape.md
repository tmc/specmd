# Extension Landscape

This note maps the OpenSpec extension surface and adjacent Markdown-spec
patterns for this Go package. It is a design input, not a commitment to a
larger API.

## Upstream Surface

Audited against Fission-AI/OpenSpec at
`bc7ab26650a43384ad525de42a7f58eaa13846f5`.

Core parsed specs do not expose an arbitrary extension field. Upstream
`src/core/schemas/spec.schema.ts:5` defines a spec as `name`, `overview`,
`requirements`, and optional `metadata`; metadata is limited to `version`,
literal `format: openspec`, and optional `sourcePath`. Requirements and
scenarios are also small: `src/core/schemas/base.schema.ts:4` and
`src/core/schemas/base.schema.ts:8` contain only scenario raw text,
requirement text, and scenarios.

Changes are similarly bounded. `src/core/schemas/change.schema.ts:10` allows
the four delta operations `ADDED`, `MODIFIED`, `REMOVED`, and `RENAMED`.
`src/core/schemas/change.schema.ts:12` gives each delta a spec name,
operation, description, optional single/plural requirement payload, and
optional rename pair. Change metadata has the same narrow source-path shape at
`src/core/schemas/change.schema.ts:33`.

Upstream customization is workflow-level, not arbitrary fields inside core
specs:

- `docs/customization.md:3` documents project config, custom schemas, and
  global overrides as the three customization levels.
- `docs/customization.md:83` defines schema resolution as CLI flag, change
  `.openspec.yaml`, project `openspec/config.yaml`, then default.
- `docs/customization.md:94` places project-local schemas under
  `openspec/schemas/<name>/`.
- `src/core/artifact-graph/types.ts:24` defines a custom workflow schema as
  `name`, `version`, optional `description`, `artifacts`, and optional `apply`.
- `src/core/artifact-graph/schema.ts:18` parses YAML schemas and hard-fails
  duplicate artifact IDs, invalid dependency references, and cycles.

Validation has hard errors and softer advisory checks. `src/core/validation/
validator.ts:24` and `:75` parse specs and changes, convert schema failures to
errors, then add rule checks. `validator.ts:290` warns for short Purpose and
reports long requirement text as info. `validator.ts:331` warns for brief delta
descriptions and missing ADDED/MODIFIED requirements. `validator.ts:391`
counts errors, warnings, and info; strict mode fails on warnings, non-strict
mode fails only on errors. `src/core/project-config.ts:47` parses project
config resiliently, returning partial config and warnings for bad fields.

The Go package is already aligned with the artifact-level core that matters
for a small library: Markdown parsing, project layout parsing, source paths,
stable ordering, delta operations, and validation reports. The remaining
upstream features that are extension-relevant are best documented rather than
implemented now: custom workflow schema parsing and project config parsing are
CLI/workflow plumbing unless a caller asks to inspect those files as artifacts.

## Feature Matrix

| Area | Upstream | Current Go package | Gap | Recommendation |
| --- | --- | --- | --- | --- |
| Core spec fields | Closed spec schema with optional metadata | Closed Go structs | No extension field | Keep closed |
| Change deltas | ADDED, MODIFIED, REMOVED, RENAMED | Supported | None for core parsing | Keep tests fixture-backed |
| Source paths | Optional metadata sourcePath | Preserved for specs, changes, deltas | None | Keep stable |
| Validation levels | error, warning, info; strict mode optional | reports distinguish levels; wrappers fail on errors | No strict wrapper | Avoid until needed |
| Project config | `openspec/config.yaml` schema/context/rules | Not parsed | Workflow/CLI concern | Do not add yet |
| Custom schemas | `openspec/schemas/<name>/schema.yaml` | Not parsed | Workflow/CLI concern | Add only as discovery refs if requested |
| Extension artifacts | No core arbitrary fields | `docs/extensions.md` convention | No parsed extension refs | Next minimal slice could discover refs |
| OOUX | Not upstream | Design only | No committed format | Keep typed design, no parser yet |

## Markdown Schema Patterns

External Markdown-spec formats converge on a two-layer shape:

- structured front matter for fields that machines must index or validate;
- ordinary Markdown sections, lists, and tables for human-readable behavior;
- optional sidecar JSON Schema when the structured layer needs machine
  validation;
- explicit schema or type identifiers to choose the right parser.

Examples:

- Factory.md uses YAML front matter plus a free-form Markdown body, validates
  front matter against JSON Schema, and keeps the structured layer minimal:
  <https://www.factoryschema.org/>.
- GitHub Docs uses YAML front matter with a test-suite schema for every docs
  page: <https://docs.github.com/en/enterprise-cloud@latest/contributing/writing-for-github-docs/using-yaml-frontmatter>.
- StratMD defines a Markdown-native strategy format with required YAML
  front matter, standardized headings, tables, optional Mermaid, and links:
  <https://stratmd.org/spec>.
- MAGI uses optional YAML front matter, typed code blocks, and relationship
  footnotes to make Markdown more machine-readable:
  <https://docs.magi-mda.org/mdx/specification>.

For OpenSpec extensions, front matter should identify extension files and hold
small metadata only. Main payloads should stay in Markdown headings/lists/tables
when they are naturally textual, or fenced JSON/YAML blocks when the extension
is mostly structured data. Sidecar JSON Schema is useful for external tooling,
but the Go package should prefer typed parsers over a generic JSON Schema
engine.

## Local md2html Precedent

`~/go/src/github.com/tmc/md2html` has useful patterns but should not be copied
wholesale.

`internal/jsonspec/jsonspec.go` enriches fenced JSON, JSONC, and JSON5 code
blocks that contain a `"type": "<prefix>/<name>"` discriminator. It is tolerant
and presentation-oriented: a regexp detects the discriminator rather than full
JSON validation. `internal/jsonspec/bundle.go` loads `*.schema.json` files into
a compact bundle keyed by filename stem, skips malformed schemas with warnings,
and preserves a small subset of JSON Schema fields. `internal/mdvet/
frontmatter.go` checks a small YAML front matter contract by hand.

Useful ideas for openspec:

- opt-in type/schema discriminator;
- stable sidecar naming;
- tolerant discovery with warnings;
- small projected metadata rather than a full JSON Schema runtime.

Not useful for openspec core:

- Goldmark rendering hooks;
- HTML badge generation;
- presentation-only schema bundles;
- treating fenced JSON as the primary extension payload.

## OOUX

OOUX is a design framework, not a machine-readable API standard. The official
OOUX ORCA article defines the four pillars as objects, relationships,
calls-to-action, and attributes:
<https://ooux.com/resources/introducing-orca-the-third-diamond-in-your-ux-process>.
Targeted search did not find canonical OOUX OpenAPI, JSON Schema, or protobuf
types. Any Go schema here should therefore be described as OOUX-inspired, not
as an official OOUX wire format.

The natural OpenSpec artifact is a domain model:

```text
openspec/extensions/ooux/model.md
openspec/changes/<id>/extensions/ooux.md
```

A protobuf-friendly Go shape can stay dependency-free:

```go
type OOUXModel struct {
	Name       string
	Objects    []OOUXObject
	Metadata   Metadata
}

type OOUXObject struct {
	Name          string
	Description   string
	Attributes    []OOUXAttribute
	Relationships []OOUXRelationship
	Actions       []OOUXAction
}
```

Use `Action` in Go, even when Markdown headings say "Calls to Action".

## Adjacent Markdown Specs

Adjacent frameworks are useful as extension candidates, but most should not
become core OpenSpec types.

| Framework | Fit | Recommendation |
| --- | --- | --- |
| OOUX / ORCA | Domain model before behavior specs | First typed extension candidate |
| EventStorming | Event/process discovery | Generic extension first; typed support only after fixtures |
| Domain Storytelling | Actors and work steps | Generic extension first |
| Bounded Context / Context Map | DDD boundaries and integrations | Generic or typed context extension |
| JTBD / Job Stories | User motivation and outcomes | Generic extension; easy Markdown tables |
| Service Blueprint | Frontstage/backstage process | Generic extension; table-heavy |
| Journey Map | Experience timeline | Generic extension; table-heavy |
| Opportunity Solution Tree | Product discovery tree | Generic extension |
| Example Mapping | Rules, examples, questions | Good fit near requirements; keep separate from core specs |
| StratMD/MAGI-like agent docs | Broader strategy/agent context | Out of scope for first-class support |

## API Direction

Do not add arbitrary `Extensions map[string]any` fields to `Spec`,
`Requirement`, or `Scenario`. That would make the core model less compatible
with upstream and harder to map to protobuf.

The smallest extension discovery API is:

```go
type ExtensionRef struct {
	Name       string
	SourcePath string
}
```

If callers need parsed typed extensions, use an artifact wrapper:

```go
type ExtensionArtifact struct {
	Name     string
	Metadata Metadata
	Payload  ExtensionPayload
}

type ExtensionPayload struct {
	OOUX *OOUXModel
}
```

Only one payload pointer should be non-nil. That invariant belongs in
extension validation, not in the parser.

Parsing rules:

- discover `openspec/extensions/<name>/*.md` in sorted order;
- discover `openspec/changes/<id>/extensions/*.md` in sorted order;
- preserve `SourcePath`;
- ignore unknown extension files in core parse APIs unless an extension
  discovery API is added;
- warn, do not fail, for malformed optional extension artifacts unless the
  caller explicitly validates that extension.

## Next Slice

No public implementation is necessary now. The current package already has the
core upstream-alignment features appropriate for a small Go library. The next
minimal implementation, when needed, should be discovery-only:

```go
type ExtensionRef struct {
	Name       string
	SourcePath string
}

type Project struct {
	Specs      []Spec
	Changes    []Change
	Extensions []ExtensionRef
}

type Change struct {
	// existing fields
	Extensions []ExtensionRef
}
```

That slice would require fixture-backed examples for:

```text
testdata/project/openspec/extensions/ooux/model.md
testdata/project/openspec/changes/add-2fa/extensions/ooux.md
```

Typed OOUX parsing should wait until those fixtures stabilize.

## Non-Goals

Do not implement TypeScript CLI commands, workspace management, command
generation, telemetry, UI/dashboard features, package-manager behavior,
archive/apply flows, or broad custom workflow execution. Do not add a JSON
Schema engine or a plugin registry unless two real extension formats need it.
