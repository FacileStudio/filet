package filet

import (
	"fmt"
	"os"
	"strings"
)

const template = `# filet configuration — https://github.com/FacileStudio/filet
# Every value below is the default; delete what you do not want to pin.
# Run "filet rules" for the full list of rule ids.

# preset: epitech   # 5 funcs/file, 20-line funcs, 4 params, depth 3, 80 cols
# preset: relaxed   # published tool defaults: 60-line funcs, complexity 15, no function budget

ignore: [.git, node_modules, vendor, dist, build, target, testdata, .venv, __pycache__, .next, .output, _app, coverage]

extensions: [.go, .ts, .tsx, .js, .jsx, .svelte, .rs, .py, .c, .h, .cpp, .java, .rb, .sh]

limits:
  fileLines: %d
  funcsPerFile: %d      # 0 turns the function budget off; _test.go files are exempt
  funcLines: %d         # Go only, blank and comment lines excluded
  funcStatements: %d    # Go only; 0 disables
  lineLength: %d
  params: %d
  returns: %d
  nesting: %d
  complexity: %d        # cognitive, not cyclomatic: a switch costs 1, nesting costs more
  structFields: %d
  interfaceMethods: %d

architecture:
  # requiredDirs: [internal, cmd]
  # forbiddenDirs: [internal/utils, pkg/common]
  # maxDepth: 6           # 0 (the default) disables the depth rule
  # fileNamePattern: '^[a-z0-9_]+\.go$'
  # requiredFiles:         # every directory matching a glob must hold these
  #   "apps/api/modules/*": [router.go]
  # forbiddenImports:      # keep the layers honest: prefix -> imports it must never pull in
  #   internal/domain: [net/http, database/sql]

style:
  banInlineComments: %t   # a comment trailing code, or sitting inside a function body
  banTODO: %t
  requireDocComments: %t
  banGlobalMutable: %t    # Must…/Once… initialised vars are treated as immutable singletons
  banInit: %t
  banTrailingSpace: %t

disabled: []              # e.g. [gen.line.long, go.doc.missing]

failOn: %s                # info | warn | error | never
`

// Scaffold writes a commented starter config at path, seeded from the named preset.
func Scaffold(path, preset string) error {
	cfg := DefaultConfig()
	if preset != "" {
		apply, ok := Preset(preset)
		if !ok {
			return fmt.Errorf("unknown preset %q (known: %s)", preset, strings.Join(PresetNames(), ", "))
		}
		apply(cfg)
	}

	l, s := cfg.Limits, cfg.Style
	body := fmt.Sprintf(template,
		l.FileLines, l.FuncsPerFile, l.FuncLines, l.FuncStatements, l.LineLength, l.Params, l.Returns,
		l.Nesting, l.Complexity, l.StructFields, l.InterfaceMethods,
		s.BanInlineComments, s.BanTODO, s.RequireDocComments, s.BanGlobalMutable, s.BanInit, s.BanTrailingSpace,
		cfg.FailOn)

	if preset != "" && preset != "default" {
		body = strings.Replace(body, "# preset: "+preset, "preset: "+preset, 1)
	}
	return os.WriteFile(path, []byte(body), 0o644)
}
