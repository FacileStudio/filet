package filet

import (
	"go/ast"
	"go/types"
)

// defaultClientCalls are the net/http functions that send through
// http.DefaultClient.
var defaultClientCalls = []string{"Get", "Head", "Post", "PostForm"}

const noTimeout = " has no Timeout, so a server that stops answering holds the request forever"

// strictTimeouts reports an HTTP client that can wait forever: an http.Client
// literal, new(http.Client) or zero-value var without a Timeout, and every use
// of http.DefaultClient, through http.Get, Head, Post and PostForm included.
// Writing Timeout: 0 is how a streaming client says it means no limit, and a
// client whose Timeout is assigned elsewhere in the file counts as set.
func (g *goFile) strictTimeouts() {
	set := g.timedClients()
	ast.Inspect(g.file, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.CompositeLit:
			g.clientLiteral(t, set)
		case *ast.CallExpr:
			g.clientCall(t, set)
		case *ast.SelectorExpr:
			g.defaultClient(t)
		case *ast.ValueSpec:
			g.clientVar(t, set)
		}
		return true
	})
}

func (g *goFile) clientLiteral(lit *ast.CompositeLit, set timed) {
	if !namedIs(g.info.TypeOf(lit), "net/http", "Client") || set.exprs[lit] || setsTimeout(lit) {
		return
	}
	g.add("go.strict.timeout", lit.Pos(), Error, "this http.Client"+noTimeout)
}

// setsTimeout reports whether a literal names the Timeout field. A positional
// literal lists every field, Timeout included.
func setsTimeout(lit *ast.CompositeLit) bool {
	for _, e := range lit.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "Timeout" {
			return true
		}
	}
	return false
}

func (g *goFile) clientCall(call *ast.CallExpr, set timed) {
	if fn := g.callee(call); isFunc(fn, "net/http", defaultClientCalls...) {
		g.add("go.strict.timeout", call.Pos(), Error, "http."+fn.Name()+" sends through http.DefaultClient, which"+noTimeout)
		return
	}
	id, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if !ok || len(call.Args) != 1 || set.exprs[call] {
		return
	}
	if _, builtin := g.info.Uses[id].(*types.Builtin); builtin && id.Name == "new" &&
		namedIs(g.info.TypeOf(call.Args[0]), "net/http", "Client") {
		g.add("go.strict.timeout", call.Pos(), Error, "new(http.Client)"+noTimeout)
	}
}

func (g *goFile) defaultClient(sel *ast.SelectorExpr) {
	v, ok := g.info.Uses[sel.Sel].(*types.Var)
	if ok && v.Pkg() != nil && v.Pkg().Path() == "net/http" && v.Name() == "DefaultClient" {
		g.add("go.strict.timeout", sel.Pos(), Error, "http.DefaultClient"+noTimeout)
	}
}

func (g *goFile) clientVar(spec *ast.ValueSpec, set timed) {
	if len(spec.Values) > 0 || spec.Type == nil || !namedIs(g.info.TypeOf(spec.Type), "net/http", "Client") {
		return
	}
	for _, name := range spec.Names {
		if obj := g.info.Defs[name]; obj != nil && !set.objs[obj] {
			g.add("go.strict.timeout", name.Pos(), Error, "the zero http.Client "+name.Name+noTimeout)
		}
	}
}
