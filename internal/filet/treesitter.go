package filet

import (
	tree_sitter_rust "github.com/FacileStudio/filet/tree_sitter/rust"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// tsGrammar pairs a file extension with the tree-sitter grammar that parses it.
// The grammar's generated C parser is vendored into tree_sitter/ and bound here
// with cgo, so parsing is deterministic and offline: a finding is a pure
// function of the source and the pinned grammar version, with no server and no
// build-time network fetch.
type tsGrammar struct {
	Ext string
	Set func(*tree_sitter.Parser) error
}

// tsGrammars is the registry of languages the treesitter tier understands.
// Add a language by vendoring its grammar under tree_sitter/ and appending a row.
var tsGrammars = []tsGrammar{
	{".rs", func(p *tree_sitter.Parser) error {
		return p.SetLanguage(tree_sitter.NewLanguage(tree_sitter_rust.Language()))
	}},
}

// GrammarFor returns the registered grammar for ext, and whether one was found.
func GrammarFor(ext string) (tsGrammar, bool) {
	for _, g := range tsGrammars {
		if g.Ext == ext {
			return g, true
		}
	}
	return tsGrammar{}, false
}

// HasGrammar reports whether the tier knows a grammar for ext.
func HasGrammar(ext string) bool {
	_, ok := GrammarFor(ext)
	return ok
}

// tsTree is a parsed tree together with the source it was parsed from.
type tsTree struct {
	Tree *tree_sitter.Tree
	Src  []byte
}

// ParseTS parses src with the grammar's language, or returns nil on failure.
func ParseTS(g tsGrammar, src []byte) *tsTree {
	parser := tree_sitter.NewParser()
	defer parser.Close()
	if err := g.Set(parser); err != nil {
		return nil
	}
	return &tsTree{Tree: parser.Parse(src, nil), Src: src}
}

func (t *tsTree) Close()                  { t.Tree.Close() }
func (t *tsTree) Root() *tree_sitter.Node { return t.Tree.RootNode() }
