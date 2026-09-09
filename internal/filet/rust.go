package filet

// rustRawClose reports whether line[i] closes an already-open Rust raw string
// with rustHashes trailing `#`s. If so, it returns the index of the last `#` of
// the closing sequence; otherwise it returns i, false.
func rustRawClose(line string, i int, rustHashes int) (int, bool) {
	if line[i] != '"' {
		return i, false
	}
	h := 0
	j := i + 1
	for j < len(line) && line[j] == '#' {
		h++
		j++
	}
	if h != rustHashes {
		return i, false
	}
	return j - 1, true
}

// rustRawOpen reports whether line[i] opens a Rust raw string. If so, it
// returns the index after the opening quote, the number of `#`s, and true;
// otherwise it returns i, 0, false.
func rustRawOpen(line string, i int) (int, int, bool) {
	c := line[i]
	if !(c == 'r' || c == 'R') {
		return i, 0, false
	}
	h := 0
	j := i + 1
	for j < len(line) && line[j] == '#' {
		h++
		j++
	}
	if j >= len(line) || line[j] != '"' {
		return i, 0, false
	}
	return j, h, true
}
