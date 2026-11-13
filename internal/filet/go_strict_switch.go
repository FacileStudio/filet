package filet

import (
	"cmp"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"
)

// enumMember is one constant of an enum-like type: its name, its exact value
// and where it is declared, so a report lists members in declaration order.
type enumMember struct {
	name  string
	value string
	pos   token.Pos
}

// strictSwitches reports a switch over an enum-like type that has no default
// and leaves some of the type's constants out: the value a new constant adds
// falls through the switch without a word.
func (g *goFile) strictSwitches() {
	ast.Inspect(g.file, func(n ast.Node) bool {
		if s, ok := n.(*ast.SwitchStmt); ok && s.Tag != nil && !hasDefault(s.Body) {
			g.exhaustive(s)
		}
		return true
	})
}

func (g *goFile) exhaustive(s *ast.SwitchStmt) {
	named := enumType(g.info.TypeOf(s.Tag))
	if named == nil {
		return
	}
	members := enumMembers(named)
	if len(members) < 2 {
		return
	}
	covered := g.caseValues(s.Body)
	var missing []string
	for _, m := range members {
		if !covered[m.value] {
			missing = append(missing, m.name)
			covered[m.value] = true
		}
	}
	if len(missing) > 0 {
		g.add("go.strict.switch", s.Pos(), Error, fmt.Sprintf("switch on %s has no default and misses %s",
			named.Obj().Name(), listNames(missing)))
	}
}

// enumType returns t when it is enum-like: a defined, non-generic type over an
// integer or string kind. Whether it has constants is enumMembers' question.
func enumType(t types.Type) *types.Named {
	n, ok := types.Unalias(t).(*types.Named)
	if !ok || n.Obj().Pkg() == nil || n.TypeParams().Len() > 0 {
		return nil
	}
	b, ok := n.Underlying().(*types.Basic)
	if !ok || b.Info()&(types.IsInteger|types.IsString) == 0 {
		return nil
	}
	return n
}

// enumMembers lists the constants of exactly type n declared in n's package,
// in declaration order. Another package is read from its export data, which
// carries only the exported constants: the ones a switch there can name.
func enumMembers(n *types.Named) []enumMember {
	pkg := n.Obj().Pkg()
	var out []enumMember
	for _, name := range pkg.Scope().Names() {
		c, ok := pkg.Scope().Lookup(name).(*types.Const)
		if ok && types.Identical(c.Type(), n) {
			out = append(out, enumMember{name: name, value: c.Val().ExactString(), pos: c.Pos()})
		}
	}
	slices.SortFunc(out, func(a, b enumMember) int { return cmp.Compare(a.pos, b.pos) })
	return out
}

func (g *goFile) caseValues(body *ast.BlockStmt) map[string]bool {
	seen := map[string]bool{}
	for _, stmt := range body.List {
		for _, e := range stmt.(*ast.CaseClause).List {
			if tv, ok := g.info.Types[e]; ok && tv.Value != nil {
				seen[tv.Value.ExactString()] = true
			}
		}
	}
	return seen
}

func hasDefault(body *ast.BlockStmt) bool {
	for _, stmt := range body.List {
		if cc, ok := stmt.(*ast.CaseClause); ok && cc.List == nil {
			return true
		}
	}
	return false
}

// listNames joins names for a message, eliding past the fifth so a switch over
// reflect.Kind does not print a paragraph.
func listNames(names []string) string {
	if len(names) <= 5 {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(names[:5], ", "), len(names)-5)
}
