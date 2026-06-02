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
