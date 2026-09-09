package filet

import "strings"

// trailingComment reports whether a line carries a comment after real code.
// An escaped marker (a backslash immediately before it) is inside a string or
// regex literal, not a comment: `\/\/` in a JS regex, `\#` in a shell word,
// so the scan skips those to the first marker that actually opens a comment.
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
	quote := byte(0)
	if st.raw {
		quote = '`'
	}
	escaped := false
	rustRaw := st.rustRaw
	rustHashes := st.rustHashes

	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case rustRaw:
			if end, ok := rustRawClose(line, i, rustHashes); ok {
				rustRaw = false
				i = end
				continue
			}
		case quote != 0:
			quote, escaped = advance(c, quote, escaped)
		case st.block:
			if c == '*' && peek(line, i) == '/' {
				st.block = false
				i++
			}
		case c == '/' && peek(line, i) == '/':
			b.WriteString(line[i:])
			return b.String(), lineState{raw: quote == '`', block: st.block, rustRaw: rustRaw, rustHashes: rustHashes}
		case c == '/' && peek(line, i) == '*':
			st.block = true
			i++
		case (c == 'r' || c == 'R'):
			if j, h, ok := rustRawOpen(line, i); ok {
				rustRaw = true
				rustHashes = h
				i = j
				continue
			}
			b.WriteByte(c)
		case c == '\'' || c == '"' || c == '`':
			quote = c
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), lineState{raw: quote == '`', block: st.block, rustRaw: rustRaw, rustHashes: rustHashes}
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
	if i := strings.Index(stripped, "//"); i >= 0 {
		return stripped[:i]
	}
	return stripped
}

// commentText returns the prose of a trailing line comment, without its marker.
func commentText(ext, stripped string) string {
	marker := "//"
	if braceless[ext] {
		marker = "#"
	}
	i := strings.Index(stripped, marker)
	if i < 0 {
		return ""
	}
	return strings.TrimLeft(stripped[i+len(marker):], " \t*")
}
