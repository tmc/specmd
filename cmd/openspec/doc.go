// Command openspec validates OpenSpec and OKF Markdown artifacts.
//
// The command intentionally exposes a small surface:
//
//	openspec validate [path]
//
// The validate command accepts an openspec project directory, an OKF bundle, a
// spec.md file, an OKF concept file, or a change directory. It reports warnings
// without failing unless -strict is set.
package main
