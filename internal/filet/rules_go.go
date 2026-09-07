package filet

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
)

type addFn func(rule string, pos token.Pos, sev Severity, msg string)

type goFile struct {
	cfg    *Config
	file   *ast.File
	isTest bool
	line   func(token.Pos) int
	noise  map[int]bool
	info   *types.Info
	add    addFn
}

// CheckGo parses a Go file and runs the AST-aware rules over it.
func CheckGo(cfg *Config, f SourceFile) []Finding {
	if f.IsGenerated() {
		return nil
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, f.Path, f.Src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return []Finding{newFinding("go.parse", f.Display, 1, Error, "cannot parse: "+err.Error())}
	}

	// Type checking for semantic rules
	var info *types.Info
	if cfg.Enabled("go.leak.resource") {
		info = &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
			Defs:  make(map[*ast.Ident]types.Object),
			Uses:  make(map[*ast.Ident]types.Object),
		}
		conf := &types.Config{Importer: importer.Default(), Error: func(err error) {}}
		if _, err := conf.Check(f.Path, fset, []*ast.File{parsed}, info); err != nil {
			info = nil
		}
	}

	var out []Finding
	g := &goFile{
		cfg:    cfg,
		file:   parsed,
		isTest: strings.HasSuffix(f.Rel, "_test.go"),
		line:   func(p token.Pos) int { return fset.Position(p).Line },
		noise:  noiseLines(f, parsed, fset),
		info:   info,
		add: func(rule string, pos token.Pos, sev Severity, msg string) {
			if !cfg.Enabled(rule) {
				return
			}
			at := fset.Position(pos)
			finding := newFinding(rule, f.Display, at.Line, sev, msg)
			finding.Column = at.Column
			out = append(out, finding)
		},
	}

	g.decls()
	g.inBodyComments()
	g.calls()
	if info != nil {
		g.resourceLeaks()
	}
	return out
}

func (g *goFile) decls() {
	receivers := map[string]map[string]token.Pos{}
	funcs := 0

	for _, decl := range g.file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			funcs++
			g.funcBudget(d, funcs)
			g.function(d)
			collectReceiver(receivers, d)
			g.funcStyle(d)
		case *ast.GenDecl:
			g.genDecl(d)
		}
	}

	for typeName, names := range receivers {
		if len(names) < 2 {
			continue
		}
		for name, pos := range names {
			g.add("go.receiver.inconsistent", pos, Info,
				fmt.Sprintf("receiver for %s is named %q here but differs elsewhere", typeName, name))
		}
	}
}

func (g *goFile) funcBudget(d *ast.FuncDecl, nth int) {
	limit := g.cfg.Limits.FuncsPerFile
	if limit > 0 && nth == limit+1 && !g.isTest {
		g.add("go.file.funcs", d.Pos(), Warn,
			fmt.Sprintf("%s is function %d in this file (limit %d)", d.Name.Name, nth, limit))
	}
}

func (g *goFile) funcStyle(d *ast.FuncDecl) {
	if d.Recv != nil {
		return
	}
	if d.Name.Name == "init" && g.cfg.Style.BanInit {
		g.add("go.init", d.Pos(), Warn, "init() runs magic before main; prefer an explicit constructor")
	}
	g.checkDoc(docTarget{doc: d.Doc, name: d.Name.Name, kind: "func", pos: d.Pos()})
}

func (g *goFile) function(d *ast.FuncDecl) {
	if d.Body == nil {
		return
	}
	lim, name := g.cfg.Limits, d.Name.Name

	if n := g.effectiveLines(d.Body); n > lim.FuncLines {
		g.add("go.func.long", d.Pos(), Warn,
			fmt.Sprintf("%s is %d lines of code (limit %d)", name, n, lim.FuncLines))
	}
	if n := statements(d.Body); lim.FuncStatements > 0 && n > lim.FuncStatements {
		g.add("go.func.statements", d.Pos(), Warn,
			fmt.Sprintf("%s has %d statements (limit %d)", name, n, lim.FuncStatements))
	}
	if n := countFields(d.Type.Params); n > lim.Params {
		g.add("go.func.params", d.Pos(), Warn,
			fmt.Sprintf("%s takes %d parameters (limit %d)", name, n, lim.Params))
	}
	if n := countFields(d.Type.Results); n > lim.Returns {
		g.add("go.func.returns", d.Pos(), Warn,
			fmt.Sprintf("%s returns %d values (limit %d)", name, n, lim.Returns))
	}
	if c := Cognitive(d); c > lim.Complexity && !g.isTest {
		g.add("go.func.complexity", d.Pos(), Warn,
			fmt.Sprintf("%s has cognitive complexity %d (limit %d)", name, c, lim.Complexity))
	}
	if named(d.Type.Results) {
		g.nakedReturns(d)
	}
}

func (g *goFile) effectiveLines(body *ast.BlockStmt) int {
	first, last := g.line(body.Pos()), g.line(body.End())
	n := 0
	for l := first + 1; l < last; l++ {
		if !g.noise[l] {
			n++
		}
	}
	return n
}

func (g *goFile) nakedReturns(d *ast.FuncDecl) {
	ast.Inspect(d.Body, func(n ast.Node) bool {
		if r, ok := n.(*ast.ReturnStmt); ok && len(r.Results) == 0 {
			g.add("go.return.naked", r.Pos(), Info, "naked return hides what "+d.Name.Name+" actually gives back")
		}
		return true
	})
}

func collectReceiver(receivers map[string]map[string]token.Pos, d *ast.FuncDecl) {
	if d.Recv == nil || len(d.Recv.List) == 0 || len(d.Recv.List[0].Names) == 0 {
		return
	}
	typeName := receiverType(d.Recv.List[0].Type)
	name := d.Recv.List[0].Names[0].Name
	if receivers[typeName] == nil {
		receivers[typeName] = map[string]token.Pos{}
	}
	if _, seen := receivers[typeName][name]; !seen {
		receivers[typeName][name] = d.Pos()
	}
}