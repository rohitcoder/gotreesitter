//go:build !grammar_subset || grammar_subset_cairo

// Code generated from tree-sitter parser.c; DO NOT EDIT.
// Source: parser.c

package grammars

// cairoExternalLexStates mirrors C tree-sitter ts_external_scanner_states.
var cairoExternalLexStates = [][]bool{
	/* 0 */ {false, false, false},
	/* 1 */ {false, true, true},
	/* 2 */ {true, false, false},
	/* 3 */ {false, true, false},
}

func init() {
	RegisterExternalLexStates("cairo", cairoExternalLexStates)
}
