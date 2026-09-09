package filet

// charStep tells the strip driver what to do after one character: write
// `write` ("" for a dropped literal), advance to `next` (0 = advance one), and
// stop when `done` (the rest of the line is a comment to keep verbatim).
type charStep struct {
	write string
	next  int
	done  bool
}

// charState is the scanner's live position inside one strip: which literal or
// block is open. It is immutable across characters except through step, which
// mutates it in place, and is what gets folded into lineState on return.
type charState struct {
	quote      byte
	escaped    bool
	rustRaw    bool
	rustHashes int
	block      bool
}

// step dispatches one character to the active-literal or code scanner. A char
// is inside a literal or block comment when any active state is set; otherwise
// it is ordinary code. This mirrors stripLine's old first-match switch exactly.
func (cs *charState) step(line string, i int) charStep {
	if cs.rustRaw || cs.quote != 0 || cs.block {
		return cs.stepActive(line, i)
	}
	return cs.stepCode(line, i)
}

// stepActive consumes one character inside an open literal or block comment.
func (cs *charState) stepActive(line string, i int) charStep {
	switch {
	case cs.rustRaw:
		if end, ok := rustRawClose(line, i, cs.rustHashes); ok {
			cs.rustRaw = false
			return charStep{next: end}
		}
		return charStep{}
	case cs.quote != 0:
		cs.quote, cs.escaped = advance(line[i], cs.quote, cs.escaped)
		return charStep{}
	default:
		if line[i] == '*' && peek(line, i) == '/' {
			cs.block = false
			return charStep{next: i + 2}
		}
		return charStep{}
	}
}

// stepCode consumes one character in ordinary code, opening literals and block
// comments and stopping at a line comment.
func (cs *charState) stepCode(line string, i int) charStep {
	c := line[i]
	switch {
	case c == '/' && peek(line, i) == '/':
		return charStep{write: line[i:], done: true}
	case c == '/' && peek(line, i) == '*':
		cs.block = true
		return charStep{next: i + 2}
	case c == 'r' || c == 'R':
		if j, h, ok := rustRawOpen(line, i); ok {
			cs.rustRaw = true
			cs.rustHashes = h
			return charStep{next: j}
		}
		return charStep{write: singleChar(c)}
	case c == '\'' || c == '"' || c == '`':
		cs.quote = c
		return charStep{}
	default:
		return charStep{write: singleChar(c)}
	}
}

// singleChar makes a one-character string from a byte.
func singleChar(c byte) string {
	return string([]byte{c})
}
