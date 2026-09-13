package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/filet/internal/filet"
)

var version = "0.17.0"

func main() {
	os.Exit(run())
}

func run() int {
	args := os.Args[1:]
	if strings.HasPrefix(filepath.Base(os.Args[0]), "droast") {
		args = append([]string{"docker"}, args...)
	}
	if len(args) == 0 {
		filet.RenderUsage(os.Stderr)
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
	case "clean":
		return clean(rest)
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
		filet.RenderUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		filet.RenderUsage(os.Stderr)
		return 2
	}
}

func listRules() int {
	for _, r := range filet.Rules() {
		fmt.Printf("%-28s %s\n", r.ID, r.Description)
	}
	return 0
}
