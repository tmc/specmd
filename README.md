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
