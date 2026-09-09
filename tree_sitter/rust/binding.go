package tree_sitter_rust

// #cgo CFLAGS: -std=c11 -fPIC
// #include "src/parser.c"
// #include "src/scanner.c"
import "C"

import "unsafe"

// Language returns the tree-sitter Language pointer for the Rust grammar.
func Language() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_rust())
}
