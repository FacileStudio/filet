package filet

import (
	"bytes"
	"go/format"
	"os"
	"slices"
	"strings"
)

// CleanStats tallies the line edits clean applied to one file, keyed by the
// rule each edit resolves, so the summary names the finding it retired.
type CleanStats struct {
	CommentedCode int
	InlineComment int
	TrailingSpace int
	Formatting    int
}

// CleanFileResult is one file's cleaned lines, whether it changed, and why.
type CleanFileResult struct {
	Path    string
	Lines   []string
	Stats   CleanStats
	Changed bool
}

// CleanReport is the outcome of one clean run: the per-file results plus the
// run-wide totals.
type CleanReport struct {
	PerFile []CleanFileResult
	Stats   CleanStats
	Files   int
	Changed int
}

// CleanFile rewrites f's lines, applying only the auto-fixes the matching
// checks would ask for. Each edit is gated on the same config toggle that gates
// its check, so clean never does more than the project's own rules demand:
//
//   - gen.commented.code: a whole line of commented-out code is dropped.
//   - gen.comment.inline: a comment trailing real code is stripped.
//   - gen.trailing.space: trailing whitespace is removed.
//
// When format is on the file is then passed through the language's own
// formatter (Go: the stdlib go/format engine, the same one gofmt drives), so a
// cleaned file also satisfies the project's formatter. Formatting is best-effort:
// a file go/format cannot parse is left with its line edits applied, never
// failing the run.
//
// A full-line prose comment, a file header, or a directive (//nolint:, //go:)
// is left alone: the inline rule only fires on a comment that follows code, and
// commented-out code is the only whole line clean deletes. Generated files are
// skipped, matching CheckGeneric.
func CleanFile(cfg *Config, f SourceFile) CleanFileResult {
	res := CleanFileResult{Path: f.Path}
	if f.IsGenerated() {
		res.Lines = f.Lines
		return res
	}
	st := lineState{}
	for _, raw := range f.Lines {
		line, drop, next := cleanLine(cfg, f, raw, st, &res.Stats)
		if drop {
			st = next
			continue
		}
		res.Lines = append(res.Lines, line)
		st = next
	}
	res.Changed = !slices.Equal(f.Lines, res.Lines)
	formatLines(cfg, f, &res)
	return res
}

// formatLines runs the reformatted source for a Go file through go/format when
// the config asks for it, replacing res.Lines when the formatter produced
// different output. A parse failure leaves the line edits in place; formatting
// is a best-effort layer, never a reason to fail the whole run.
func formatLines(cfg *Config, f SourceFile, res *CleanFileResult) {
	if !cfg.Style.Format || f.Ext != ".go" {
		return
	}
	before := res.Lines
	if len(before) > 0 && before[len(before)-1] == "" {
		before = before[:len(before)-1]
	}
	formatted, err := format.Source([]byte(strings.Join(before, "\n") + "\n"))
	if err != nil {
		return
	}
	after := strings.Split(string(formatted), "\n")
	if len(after) > 0 && after[len(after)-1] == "" {
		after = after[:len(after)-1]
	}
	if slices.Equal(after, before) {
		return
	}
	res.Lines = after
	res.Changed = true
	res.Stats.Formatting++
}

// cleanLine resolves the auto-fixable findings on one raw line and returns the
// rewritten line, whether it should be dropped (a commented-out-code line), and
// the scanner state to feed the next line. It records each applied fix on s.
func cleanLine(cfg *Config, f SourceFile, raw string, st lineState, s *CleanStats) (string, bool, lineState) {
	stripped, next := stripLine(raw, st)
	if cfg.Enabled("gen.commented.code") && commentedOut(f.Ext, stripped) {
		s.CommentedCode++
		return "", true, next
	}
	line := raw
	if cfg.Style.BanInlineComments && cfg.Enabled("gen.comment.inline") &&
		inlineComment(cfg, f, stripped) {
		if i := trailingCommentIndex(raw, st); i >= 0 {
			line = raw[:i]
			s.InlineComment++
		}
	}
	if cfg.Style.BanTrailingSpace && cfg.Enabled("gen.trailing.space") &&
		line != strings.TrimRight(line, " \t") {
		line = strings.TrimRight(line, " \t")
		s.TrailingSpace++
	}
	return line, false, next
}

// trailingCommentIndex returns the byte index in raw where a trailing `//`
// comment begins, or -1 when there is none. stripLine drops string-literal
// bodies, so a `//` inside a string never survives into the stripped line the
// checks look at; this drives the same scanner to recover that comment's
// position in the original, guarded like trailingComment against an escaped
// marker or a URL separator.
func trailingCommentIndex(raw string, st lineState) int {
	cs := charState{rustRaw: st.rustRaw, rustHashes: st.rustHashes, block: st.block}
	if st.raw {
		cs.quote = '`'
	}
	i := 0
	for i < len(raw) {
		step := cs.step(raw, i)
		if step.done {
			if commentStart(raw, i) {
				return i
			}
			i += 2
			continue
		}
		if step.next > 0 {
			i = step.next
		} else {
			i++
		}
	}
	return -1
}

// commentStart reports whether a line-comment marker at raw index i is a real
// trailing comment: real code precedes it and the marker is neither escaped nor
// a URL separator.
func commentStart(raw string, i int) bool {
	return i > 0 && raw[i-1] != '\\' && raw[i-1] != ':' && strings.TrimSpace(raw[:i]) != ""
}

// Clean walks target and writes back every file whose auto-fixable findings
// clean can resolve. In dryRun mode it reports what would change without
// touching the disk. A file is rewritten only when it actually changed, and its
// original trailing newline and mode are preserved.
func Clean(cfg *Config, target string, dryRun bool) (CleanReport, error) {
	files, err := Scan(cfg, target)
	if err != nil {
		return CleanReport{}, err
	}
	report := CleanReport{Files: len(files)}
	for _, f := range files {
		res := CleanFile(cfg, f)
		if res.Changed {
			if err := applyChanged(&report, res, f.Src, dryRun); err != nil {
				return report, err
			}
		}
		report.PerFile = append(report.PerFile, res)
	}
	return report, nil
}

// applyChanged folds one changed file into the report and writes it back, unless
// dryRun asked for a rehearsal.
func applyChanged(r *CleanReport, res CleanFileResult, src []byte, dryRun bool) error {
	r.Changed++
	r.Stats.CommentedCode += res.Stats.CommentedCode
	r.Stats.InlineComment += res.Stats.InlineComment
	r.Stats.TrailingSpace += res.Stats.TrailingSpace
	r.Stats.Formatting += res.Stats.Formatting
	if dryRun {
		return nil
	}
	return writeCleaned(res.Path, src, res.Lines)
}

// writeCleaned writes lines back over path, preserving the original trailing
// newline and file mode.
func writeCleaned(path string, src []byte, lines []string) error {
	var b bytes.Buffer
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	switch {
	case bytes.HasSuffix(src, []byte("\r\n")):
		b.WriteString("\r\n")
	case bytes.HasSuffix(src, []byte("\n")):
		b.WriteByte('\n')
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), info.Mode()&os.ModePerm)
}
