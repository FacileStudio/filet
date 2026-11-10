package filet

import "maps"

// punchlinesExtra holds the punchlines for the newer rules, split out of
// roast.go to keep that data-heavy map under the per-file line limit.
var punchlinesExtra = map[string][]string{
	"gen.config.inert": {
		"A rule nobody can break is a rule nobody is following.",
		"The config promises a check the code never gets.",
	},
	"gen.baseline.stale": {
		"You paid off debt and forgot to write it down.",
		"The ratchet only clicks one way. Click it.",
	},
	"lsp.error": {
		"Your toolchain already knows, and now so does the report.",
		"This is the same red squiggly you have been ignoring in the editor.",
	},
	"lsp.warn": {
		"The language server is shouting quietly.",
		"A warning now is an incident later, and the server saw it coming.",
	},
	"lsp.info": {
		"The language server has thoughts on this line.",
		"Not fatal, but the toolchain is not impressed.",
	},
	"lsp.unavailable": {
		"No server, no diagnostics, no excuse on the report.",
		"filet speaks several lint languages, just not this one yet.",
	},
	"go.leak.resource": {
		"An unclosed resource is a leak waiting to happen.",
		"Close() is not optional. The GC will not save you.",
		"Every open handle is a promise to the OS. Keep it.",
	},
	"go.err.nilerr": {
		"Handling an error then returning nil is how silent failures are born.",
		"The if-block is there for a reason. Use it or delete it.",
		"nil, nil compiles but does not compute.",
	},
	"test.fuzz.missing": {
		"A parser nobody fuzzed is a parser the internet will fuzz for you.",
		"Three hand-picked test inputs are not a threat model.",
		"go test -fuzz is free. Production crashes are not.",
	},
}

// mergePunchlines combines two rule-to-punchline maps into one lookup.
func mergePunchlines(a, b map[string][]string) map[string][]string {
	out := make(map[string][]string, len(a)+len(b))
	maps.Copy(out, a)
	maps.Copy(out, b)
	return out
}
