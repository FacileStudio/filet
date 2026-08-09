package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saravenpi/filet/internal/filet"
)

const version = "0.1.0"

const usage = `filet — style checker, code roaster and test runner

usage:
  filet check   [flags] [path]   run style, quality and architecture rules
  filet roast   [flags] [path]   same rules, with commentary
  filet docker  [flags] [path]   roast every Dockerfile it finds
  filet test    [flags] [path]   detect and run the project's test suites
  filet init    [flags] [path]   write a commented .filet.yml (-preset relaxed|epitech)
  filet rules                    list every rule id
  filet version

droast is an alias for "filet docker".

flags:
  -format text|json    output format (default text)
  -fail info|warn|error|never   severity that makes the command exit 1
  -quiet               only print the summary
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
