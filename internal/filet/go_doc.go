package filet

import (
	"go/ast"
	"go/token"
	"strings"
)

type docTarget struct {
	doc  *ast.CommentGroup
	name string
	kind string
	pos  token.Pos

	// shared marks documentation that covers several declarations at once — the
	// one comment above a `type ( ... )` block. It cannot open with the name of
	// each of them, so go.doc.form has nothing to say about it.
	shared bool
}

// checkDoc enforces both halves of the house convention: exported things carry a
// doc comment, and a Go doc comment opens with the identifier it documents.
func (g *goFile) checkDoc(t docTarget) {
	if !g.cfg.Style.RequireDocComments || g.isTest || !ast.IsExported(t.name) {
		return
	}
	text := prose(t.doc)
	if text == "" {
		g.add("go.doc.missing", t.pos, Info, "exported "+t.kind+" "+t.name+" has no doc comment")
		return
	}
	if t.shared || !g.cfg.Enabled("go.doc.form") {
		return
	}
	if !strings.HasPrefix(text, t.name) && !strings.HasPrefix(text, "Deprecated:") {
		g.add("go.doc.form", t.pos, Info,
			"doc comment for "+t.name+" should start with \""+t.name+"\"")
	}
}

// prose returns the first line of real documentation in a group. A group holding
// nothing but directives documents nothing, whatever godoc's parser thinks.
func prose(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	for _, c := range group.List {
		if IsDirective(c.Text) {
			continue
		}
		if text := strings.TrimSpace(strings.TrimLeft(c.Text, "/*! \t")); text != "" {
			return text
		}
	}
	return ""
}
