# Extension Design

Package `openspec` keeps the core OpenSpec artifact model small: specs,
changes, deltas, requirements, scenarios, and validation reports. Extensions
should add typed artifact models around that core, not arbitrary fields inside
requirements.

Upstream OpenSpec extends workflows through project configuration and custom
schemas under `openspec/schemas/<name>/`. Its core parsed spec schema has only
`name`, `overview`, `requirements`, and optional metadata. That makes
extension artifacts the safest compatibility boundary for this Go package.

## Principles

Extensions should be:

- optional: parsing a spec must not require any extension package;
- typed: exported Go structs should describe extension data directly;
- protobuf-friendly: use stable scalar fields, repeated fields, and enum-like
  strings rather than `map[string]any` in primary models;
- markdown-native: extension files should remain readable without generated
  tooling;
- side-effect free: extension parsing should not mutate the core `Project`;
- tolerant: unknown extension artifacts are preserved as paths or ignored by
  default, not treated as core validation failures.

The core package should avoid a generic plugin registry until there are at
least two real extension formats. A registry invites a larger API than the
current package needs and makes protobuf mapping harder. If the package grows
extension support, the first shape should be a small typed wrapper, analogous
to a protobuf `oneof`, not a dynamic registry.

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

That is enough when the package only needs to tell callers that extension
artifacts exist. Once the package parses a concrete extension, use a typed
artifact wrapper:

```go
type ExtensionArtifact struct {
    Name     string
    Metadata Metadata
    Payload  ExtensionPayload
}

type ExtensionPayload struct {
    OOUX *OOUXModel
    // Future concrete extensions go here.
}

type Project struct {
    Specs      []Spec
    Changes    []Change
    Extensions []ExtensionArtifact
}

type Change struct {
    // existing fields...
    Extensions []ExtensionArtifact
}
```

Only one `ExtensionPayload` pointer should be non-nil. That invariant can be
checked by `ValidateExtension` or an extension-level report, not by the parser.

The extension parser should live in the root package only when the format is
stable and small. Otherwise it should be a sibling package, for example
`github.com/tmc/openspec/ooux`, importing the core package.

This keeps the protobuf analogue direct:

```proto
message ExtensionRef {
  string name = 1;
  string source_path = 2;
}

message ExtensionArtifact {
  string name = 1;
  Metadata metadata = 2;
  oneof payload {
    OOUXModel ooux = 10;
  }
}
```

Typed extension payloads then define their own messages instead of packing
everything into `google.protobuf.Struct`.

## OOUX Extension

OOUX fits OpenSpec as a domain model artifact. It describes the objects users
understand before flows, screens, or implementation tasks. The ORCA shape is:
objects, relationships, calls to action, and attributes.

Markdown:

```markdown
# OOUX Model: Authentication

## Objects

### Object: User
The person authenticating with the system.

#### Attributes
- email
- display name
- two-factor status

#### Relationships
- has many Sessions
- may have one Authenticator

#### Calls to Action
- log in
- enroll two-factor authentication
- revoke session

### Object: Session
An authenticated period of access.

#### Attributes
- issued at
- expires at
- last activity at

#### Relationships
- belongs to User

#### Calls to Action
- expire
- revoke
```

Go:

```go
type Model struct {
    Name       string
    Objects    []Object
    Metadata   openspec.Metadata
}

type Object struct {
    Name          string
    Description   string
    Attributes    []Attribute
    Relationships []Relationship
    Actions       []Action
}

type Attribute struct {
    Name        string
    Description string
}

type Relationship struct {
    Target      string
    Description string
}

type Action struct {
    Name        string
    Description string
}
```

Protobuf:

```proto
message OOUXModel {
  string name = 1;
  repeated OOUXObject objects = 2;
  string source_path = 3;
}

message OOUXObject {
  string name = 1;
  string description = 2;
  repeated OOUXAttribute attributes = 3;
  repeated OOUXRelationship relationships = 4;
  repeated OOUXAction actions = 5;
}

message OOUXAttribute {
  string name = 1;
  string description = 2;
}

message OOUXRelationship {
  string target = 1;
  string description = 2;
}

message OOUXAction {
  string name = 1;
  string description = 2;
}
```

Use `Action` in Go rather than `CTA`. The Markdown can say "Calls to Action";
the Go type should use the ordinary word.

A more compact first implementation could use `[]string` for attributes,
relationships, and actions. Prefer the structured `Attribute`,
`Relationship`, and `Action` forms once descriptions or cross-artifact
diagnostics are needed; they map better to protobuf and avoid another API
change.

## Validation

OOUX validation should be advisory unless a file is malformed:

- error: object heading is present but empty;
- warning: model has no objects;
- warning: object has no attributes;
- warning: object has no relationships;
- warning: object has no actions;
- info: relationship target does not match another object in the same model;
- info: object name does not appear in any parsed spec requirement text.

These checks belong in a report type, mirroring `ValidationReport`, so extension
warnings do not make core specs invalid.

## What Not To Do

Do not add `Extensions map[string]any` to `Spec`, `Requirement`, or `Delta`.
It is hard to document, hard to validate, awkward in protobuf, and would imply
that upstream core specs accept arbitrary fields when they do not.

Do not require front matter for OOUX. Front matter is useful for metadata, but
the model itself should be visible in Markdown headings and lists.

Do not make OOUX part of `ValidateSpec`. A spec should remain valid even when no
OOUX artifact exists. Cross-artifact consistency checks can be a separate
project-level report.
