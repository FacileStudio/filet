package filet

import "maps"

// punchlinesExtra holds the punchlines for the newer rules, split out of
// roast.go to keep that data-heavy map under the per-file line limit.
var punchlinesExtra = map[string][]string{
	"gen.config.inert": {
		"A rule nobody can break is a rule nobody is following.",
		"The config promises a check the code never gets.",
	},
	"gen.license.required": {
		"No licence means all rights reserved, whatever the README hopes.",
		"The code is open. The permission to use it is not written down.",
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
	"go.strict.switch": {
		"The constant you add next month already has a silent branch waiting.",
		"A switch with no default is a promise to list everything. Keep it.",
	},
	"go.strict.close": {
		"A response body nobody closes keeps its connection hostage.",
		"Discarding the handle does not close it. It just loses it.",
	},
	"go.strict.import": {
		"unsafe is a door out of the type system, and it locks behind you.",
		"Reflection: deferring compile errors to three in the morning.",
	},
	"go.strict.timeout": {
		"No timeout means the slowest server on the internet sets your latency.",
		"http.DefaultClient will wait for a reply until the heat death of the universe.",
	},
	"go.strict.sql": {
		"Concatenated SQL is a form field away from DROP TABLE.",
		"Placeholders exist so Bobby Tables stays a joke.",
	},
	"go.strict.goroutine": {
		"A goroutine with no context is a goroutine with no off switch.",
		"The request ended. Nobody told the goroutine.",
	},
	"go.strict.assert": {
		"x.(T) is a panic with extra steps.",
		"The comma-ok form costs one identifier. The panic costs the process.",
	},
	"go.strict.nilflow": {
		"The error path is where this pointer is nil, and you read it first.",
		"Check the error, then touch the result. In that order.",
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
