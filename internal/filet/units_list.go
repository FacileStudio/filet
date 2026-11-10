package filet

import (
	"fmt"
	"io"
	"strings"
)

// AllSuites flattens the units' suites in unit order.
func AllSuites(units []Unit) []Suite {
	var out []Suite
	for _, u := range units {
		out = append(out, u.Suites...)
	}
	return out
}

func (u Unit) suite(name, cmd string, args ...string) Suite {
	return Suite{Name: name, Dir: u.Dir, Unit: u.Path, Cmd: cmd, Args: args}
}

// UnitsLine is the one line the gate prints above its table so a reader sees
// what it covered: `units: apps/api (go), apps/client (bun)`.
func UnitsLine(units []Unit) string {
	if len(units) == 0 {
		return "units: none found"
	}
	parts := make([]string, 0, len(units))
	for _, u := range units {
		parts = append(parts, u.Path+" ("+strings.Join(u.Kinds, ", ")+")")
	}
	return "units: " + strings.Join(parts, ", ")
}

// WriteUnits prints UnitsLine, in bold on a terminal.
func WriteUnits(w io.Writer, units []Unit) {
	p := painter(Colorize(w))
	fmt.Fprintf(w, "\n  %s\n", p(bold, UnitsLine(units)))
}
