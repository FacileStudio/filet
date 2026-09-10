package filet

import "strings"

// trailingComment reports whether a line carries a comment after real code.
// Two lookalikes are deliberately not comments: an escaped marker (a regex
// like /^https?:\/\//) is a literal, and a protocol separator (https://) is
// a URL — in a Svelte template or a help string it is text, not a trailing
// comment, and stripping it would corrupt the very thing the page tells the
// user to type.
func trailingComment(ext, stripped string) bool {
	marker := "//"
	if braceless[ext] {
		marker = "#"
	}
	search := 0
	for {
		i := strings.Index(stripped[search:], marker)
		if i < 0 {
			return false
		}
		i += search
		if i == 0 {
			return false
		}
		switch stripped[i-1] {
		case '\\':
			search = i + 2
			continue
		case ':':
			search = i + 2
			continue
		}
		return strings.TrimSpace(stripped[:i]) != ""
	}
}

// lineState carries the scanner's position across a line boundary: only raw
// strings and block comments can stay open at the end of a line.
type lineState struct {
	raw        bool
	block      bool
	rustRaw    bool
	rustHashes int
}

// stripLine removes string literal contents and block comments from one line so
// brace scanning cannot be fooled by punctuation inside them. A line comment and
// its text are kept, so TODO and inline-comment rules still see them. It returns
// the state to feed into the next line.
func stripLine(line string, st lineState) (string, lineState) {
	var b strings.Builder
	b.Grow(len(line))
	cs := charState{rustRaw: st.rustRaw, rustHashes: st.rustHashes, block: st.block}
	if st.raw {
		cs.quote = '`'
	}
	i := 0
	for i < len(line) {
		step := cs.step(line, i)
		b.WriteString(step.write)
		if step.done {
			break
		}
		i++
		if step.next > 0 {
			i = step.next
		}
	}
	return b.String(), lineState{raw: cs.quote == '`', block: cs.block, rustRaw: cs.rustRaw, rustHashes: cs.rustHashes}
}

func peek(line string, i int) byte {
	if i+1 < len(line) {
		return line[i+1]
	}
	return 0
}

func advance(c, quote byte, escaped bool) (byte, bool) {
	switch {
	case escaped:
		return quote, false
	case quote != '`' && c == '\\':
		return quote, true
	case c == quote:
		return 0, false
	}
	return quote, false
}

// codeOnly drops a trailing line comment so only real code is brace-counted.
func codeOnly(stripped string) string {
	before, _, ok := strings.Cut(stripped, "//")
	if !ok {
		return stripped
	}
	return before
}

// commentText returns the prose of a trailing line comment, without its marker.
func commentText(ext, stripped string) string {
	marker := "//"
	if braceless[ext] {
		marker = "#"
	}
	_, rest, ok := strings.Cut(stripped, marker)
	if !ok {
		return ""
	}
	return strings.TrimLeft(rest, " \t*")
}
