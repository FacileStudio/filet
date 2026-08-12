package filet

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// commentedOut reports a commented-out statement. A line ending in a full stop
// is a sentence, whatever code-shaped punctuation it happens to contain. The
// marker depends on the language: `#` counts only where `#` actually starts a
// comment, so a TypeScript/Svelte private field (`#anchor = -1`) is never read
// as commented-out code.
func commentedOut(ext, raw string) bool {
	if strings.HasSuffix(strings.TrimRight(raw, " \t"), ".") {
		return false
	}
	re := commentedCodeCC
	if braceless[ext] {
		re = commentedCodeSharp
	}
	return re.MatchString(raw)
}

func checkLine(cfg *Config, f SourceFile, n int, raw, line string) []Finding {
	var out []Finding
	if w := utf8.RuneCountInString(raw); w > cfg.Limits.LineLength && cfg.Enabled("gen.line.long") {
		out = append(out, newFinding("gen.line.long", f.Display, n, Info,
			fmt.Sprintf("line is %d characters (limit %d)", w, cfg.Limits.LineLength)))
	}
	if cfg.Style.BanTrailingSpace && cfg.Enabled("gen.trailing.space") && raw != strings.TrimRight(raw, " \t") {
		out = append(out, newFinding("gen.trailing.space", f.Display, n, Info, "trailing whitespace"))
	}
	if m := leftoverMarker(cfg, f, line); m != "" {
		out = append(out, newFinding("gen.todo", f.Display, n, Warn, "leftover "+m+" marker"))
	}
	if inlineComment(cfg, f, line) {
		out = append(out, newFinding("gen.comment.inline", f.Display, n, Info, "inline comment trailing code"))
	}
	if cfg.Enabled("gen.commented.code") && commentedOut(f.Ext, raw) {
		out = append(out, newFinding("gen.commented.code", f.Display, n, Info, "commented-out code"))
	}
	return out
}
