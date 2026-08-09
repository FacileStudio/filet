package filet

import "strings"

// directivePrefixes are the comment openings that some other tool reads. Removing
// one changes what that tool does, so they are code wearing a comment's clothes.
var directivePrefixes = []string{
	"go:", "nolint", "lint:", "+build", "revive:", "staticcheck",
	"eslint-", "@ts-", "prettier-ignore", "biome-ignore", "deno-lint-ignore",
	"istanbul ignore", "c8 ignore", "v8 ignore", "noinspection",
	"type: ignore", "noqa", "pylint:", "ruff:", "mypy:", "fmt:",
	"clippy::", "rustfmt::", "swiftlint:", "codeql", "sonar",
}

// IsDirective reports whether a comment instructs a tool rather than a reader.
// filet never asks for one of these to be deleted, and never counts it as
// documentation.
func IsDirective(text string) bool {
	body := strings.ToLower(strings.TrimLeft(text, "/#*! \t"))
	for _, prefix := range directivePrefixes {
		if strings.HasPrefix(body, prefix) {
			return true
		}
	}
	return false
}
