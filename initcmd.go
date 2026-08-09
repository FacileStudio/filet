package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FacileStudio/filet/internal/filet"
)

func initConfig(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	preset := fs.String("preset", "", "seed the config from a preset: relaxed, epitech")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	target := "."
	if fs.NArg() > 0 {
		target = fs.Arg(0)
		if err := fs.Parse(fs.Args()[1:]); err != nil {
			return 2
		}
	}

	path := filepath.Join(target, filet.ConfigNames()[0])
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintln(os.Stderr, path+" already exists")
		return 1
	}
	if err := filet.Scaffold(path, *preset); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	fmt.Println("wrote " + path)
	return 0
}
