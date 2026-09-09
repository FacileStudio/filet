package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/filet/internal/filet"
)

var version = "0.7.1"

const usage = `filet — style checker, code roaster and test runner

COMMANDS

  filet check   [flags] [path]   Run style, quality and architecture rules.
  filet roast   [flags] [path]   Run the same checks, with punchlines.
  filet docker  [flags] [path]   Roast every Dockerfile found under path.
  filet test    [flags] [path]   Detect and run the project's test suites.
  filet init    [flags] [path]   Write a commented filet.yml (-preset relaxed|epitech).
  filet rules                    List every rule id.
  filet version

DROAST

  droast is an alias for "filet docker".

FLAGS

  -format auto|text|line|json|sarif|github   Output format. Default: auto.
  -fail   info|warn|error|never              Severity that makes the command fail.
  -quiet                                     Print the summary only.

EXAMPLES

  filet check .
  filet roast ./apps ./pkg
  filet docker .
  filet test ./apps/api
  filet init -preset relaxed
`

func main() {
	os.Exit(run())
}

func run() int {
	args := os.Args[1:]
	if strings.HasPrefix(filepath.Base(os.Args[0]), "droast") {
		args = append([]string{"docker"}, args...)
	}
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	return dispatch(args[0], args[1:])
}

func dispatch(cmd string, rest []string) int {
	switch cmd {
	case "check":
		return analyze("check", rest, false)
	case "roast":
		return analyze("roast", rest, true)
	case "docker":
		return docker(rest)
	case "test":
		return test(rest)
	case "init":
		return initConfig(rest)
	case "rules":
		return listRules()
	case "version", "--version", "-v":
		fmt.Println("filet " + version)
		return 0
	case "help", "--help", "-h":
		fmt.Print(usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		return 2
	}
}

func listRules() int {
	for _, r := range filet.Rules() {
		fmt.Printf("%-28s %s\n", r.ID, r.Description)
	}
	return 0
}
