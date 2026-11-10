package filet

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Suite is one detected test command for a project or workspace member. Unit
// is the member's path relative to the discovery target, "." for the target.
type Suite struct {
	Name string
	Dir  string
	Unit string
	Cmd  string
	Args []string
}

// Result is the outcome of running a Suite.
type Result struct {
	Suite    string        `json:"suite"`
	Unit     string        `json:"unit,omitempty"`
	Command  string        `json:"command"`
	ExitCode int           `json:"exitCode"`
	Millis   int64         `json:"durationMs"`
	Duration time.Duration `json:"-"`
	Err      string        `json:"error,omitempty"`
}

// DetectSuites inspects dir and returns every test command that applies to it.
func DetectSuites(dir string) []Suite {
	var out []Suite
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}

	if exists("go.mod") {
		out = append(out, Suite{Name: "go", Dir: dir, Cmd: "go", Args: []string{"test", "./..."}})
	}
	if exists("Cargo.toml") {
		out = append(out, Suite{Name: "cargo", Dir: dir, Cmd: "cargo", Args: []string{"test"}})
	}
	if exists("deno.json") || exists("deno.jsonc") {
		out = append(out, Suite{Name: "deno", Dir: dir, Cmd: "deno", Args: []string{"test", "-A"}})
	}
	if exists("package.json") {
		out = append(out, nodeSuite(dir, exists))
	}
	if exists("pyproject.toml") || exists("pytest.ini") || exists("setup.cfg") || exists("tox.ini") {
		out = append(out, Suite{Name: "pytest", Dir: dir, Cmd: "pytest", Args: []string{"-q"}})
	}
	return out
}

func packageScripts(dir string) map[string]string {
	raw, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(raw, &pkg) != nil {
		return nil
	}
	return pkg.Scripts
}

func nodeSuite(dir string, exists func(string) bool) Suite {
	_, hasScript := packageScripts(dir)["test"]
	switch {
	case exists("bun.lock") || exists("bun.lockb"):
		if hasScript {
			return Suite{Name: "bun", Dir: dir, Cmd: "bun", Args: []string{"run", "test"}}
		}
		return Suite{Name: "bun", Dir: dir, Cmd: "bun", Args: []string{"test"}}
	case exists("pnpm-lock.yaml"):
		return Suite{Name: "pnpm", Dir: dir, Cmd: "pnpm", Args: []string{"test"}}
	case exists("yarn.lock"):
		return Suite{Name: "yarn", Dir: dir, Cmd: "yarn", Args: []string{"test"}}
	default:
		return Suite{Name: "npm", Dir: dir, Cmd: "npm", Args: []string{"test"}}
	}
}

// Command renders the command line this suite will run.
func (s Suite) Command() string {
	return s.Cmd + " " + strings.Join(s.Args, " ")
}

// Label is the name a heading or summary row shows: the suite name, then its
// unit when the suite does not sit at the target itself.
func (s Suite) Label() string {
	return withUnit(s.Name, s.Unit)
}

func withUnit(name, unit string) string {
	if unit == "" || unit == "." {
		return name
	}
	return name + " " + unit
}

// RunSuite executes one suite, streaming its output to w.
func RunSuite(s Suite, extra []string, w io.Writer) Result {
	args := append(append([]string{}, s.Args...), extra...)
	res := Result{Suite: s.Name, Unit: s.Unit, Command: s.Cmd + " " + strings.Join(args, " ")}

	if _, err := exec.LookPath(s.Cmd); err != nil {
		res.ExitCode = 127
		res.Err = s.Cmd + " not found in PATH"
		return res
	}

	cmd := exec.Command(s.Cmd, args...)
	cmd.Dir = s.Dir
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.Env = os.Environ()

	start := time.Now()
	err := cmd.Run()
	res.Duration = time.Since(start)
	res.Millis = res.Duration.Milliseconds()

	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		res.ExitCode = exitErr.ExitCode()
	default:
		res.ExitCode = 1
		res.Err = err.Error()
	}
	return res
}
