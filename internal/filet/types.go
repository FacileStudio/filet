package filet

import "go/types"

// implementsCloser checks if a type implements io.Closer by checking if
// it has a Close() error method.
func (g *goFile) implementsCloser(t types.Type) bool {
	named := g.getNamedType(t)
	if named == nil {
		return false
	}
	return g.hasCloseMethod(named)
}

// getNamedType gets the named type from a pointer or interface.
func (g *goFile) getNamedType(t types.Type) *types.Named {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return nil
	}
	return named
}

// hasCloseMethod checks if a named type has a Close method that returns error.
func (g *goFile) hasCloseMethod(named *types.Named) bool {
	for m := range named.Methods() {
		if m.Name() != "Close" {
			continue
		}
		if g.returnsError(m.Type()) {
			return true
		}
	}
	return false
}

// returnsError checks if a signature returns exactly one error value.
func (g *goFile) returnsError(typ types.Type) bool {
	sig, ok := typ.(*types.Signature)
	if !ok {
		return false
	}
	results := sig.Results()
	if results == nil || results.Len() != 1 {
		return false
	}
	result := results.At(0)
	return g.isErrorType(result.Type())
}

// isErrorType checks if a type is the built-in error type or an empty interface.
func (g *goFile) isErrorType(t types.Type) bool {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name() == "error"
	}
	if iface, ok := t.(*types.Interface); ok {
		return iface.NumMethods() == 0 || iface.Empty()
	}
	return false
}
