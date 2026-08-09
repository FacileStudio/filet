package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/saravenpi/filet/internal/filet"
)

func test(args []string) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	target, extra := splitTarget(fs.Args())

	suites := filet.DetectSuites(target)
	if len(suites) == 0 {
		fmt.Fprintln(os.Stderr, "no test suite detected in "+target)
		return 2
	}

	out := os.Stdout
	if *format == "json" {
		out = os.Stderr
	}
	results := runSuites(suites, extra, out)

	if *format == "json" {
		return emitResults(results)
	}
	if worst := filet.WriteResults(out, results); worst != 0 {
		return 1
	}
	return 0
}

func splitTarget(args []string) (string, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return ".", args
	}
	if info, err := os.Stat(args[0]); err != nil || !info.IsDir() {
		return ".", args
	}
	return args[0], args[1:]
}

func runSuites(suites []filet.Suite, extra []string, out *os.File) []filet.Result {
	results := make([]filet.Result, 0, len(suites))
	for _, s := range suites {
		fmt.Fprintf(out, "\n── %s ──\n", s.Name)
		results = append(results, filet.RunSuite(s, extra, out))
	}
	return results
}

func emitResults(results []filet.Result) int {
	payload := struct {
		Results []filet.Result `json:"results"`
	}{results}
	if err := filet.WriteJSONValue(os.Stdout, payload); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	for _, r := range results {
		if r.ExitCode != 0 {
			return 1
		}
	}
	return 0
}
