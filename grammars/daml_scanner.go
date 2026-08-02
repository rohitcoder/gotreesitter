//go:build !grammar_subset || grammar_subset_daml

package grammars

import gotreesitter "github.com/odvcencio/gotreesitter"

// DAML reuses the Haskell layout scanner, rebased onto DAML's symbol IDs.
//
// DAML is a Haskell dialect and inherits its layout rules wholesale:
// significant indentation, implicit block starts for do/case/if/let, layouts
// terminated by parse errors, and phantom where/in/arrow/bar/deriving
// tokens. Its grammar's externals list is Haskell's, verified index by index
// against tree-sitter-daml's generated parser.c — all 49 tokens agree in
// both position and meaning, from _cond_layout_semicolon at 1 through
// _token1 at 48.
//
// What differs is the symbol IDs those indices map to. HaskellExternalScanner
// carries a hardcoded table starting at 108; DAML's equivalents start at 120,
// a constant offset of 12 across every one of the 49 tokens. Attaching the
// scanner unmodified therefore produced a parser that read `module Main
// where` but not `x = 1`: the layout tokens were being emitted under
// Haskell's IDs, which mean something else in DAML's table.
//
// This is the same class of bug that made every non-empty block comment in
// Move unparseable — a scanner returning symbol IDs that its grammar does not
// use — so the fix is the same one: resolve the IDs from the Language at
// attach time rather than trusting a compiled-in constant. Writing a second
// 2300-line port of Haskell's layout algorithm would instead leave two
// implementations of a notoriously subtle rule to drift apart.
type DamlExternalScanner struct {
	inner  HaskellExternalScanner
	remap  map[gotreesitter.Symbol]gotreesitter.Symbol
	remapd bool
}

// ExternalScannerForLanguage binds this scanner to lang, building a map from
// the symbol IDs the Haskell scanner emits to the IDs DAML's tables expect.
//
// Both grammars list their externals in the same order, so index i in one is
// index i in the other; the map is built from that correspondence rather than
// from the offset, so it stays correct if a future grammar revision changes
// the spacing rather than merely the base.
func (DamlExternalScanner) ExternalScannerForLanguage(lang *gotreesitter.Language) gotreesitter.ExternalScanner {
	s := DamlExternalScanner{}
	if lang == nil || len(lang.ExternalSymbols) == 0 {
		return s
	}
	hs := haskellExternalSymbolIDs()
	if len(hs) != len(lang.ExternalSymbols) {
		// A length mismatch means the two externals lists have diverged, and
		// index-for-index reuse is no longer sound. Return unbound rather
		// than silently emitting wrong symbols.
		return s
	}
	s.remap = make(map[gotreesitter.Symbol]gotreesitter.Symbol, len(hs))
	for i, from := range hs {
		s.remap[from] = lang.ExternalSymbols[i]
	}
	s.remapd = true
	return s
}

func (DamlExternalScanner) Create() any         { return HaskellExternalScanner{}.Create() }
func (DamlExternalScanner) Destroy(payload any) { HaskellExternalScanner{}.Destroy(payload) }
func (DamlExternalScanner) Serialize(payload any, buf []byte) int {
	return HaskellExternalScanner{}.Serialize(payload, buf)
}
func (DamlExternalScanner) Deserialize(payload any, buf []byte) {
	HaskellExternalScanner{}.Deserialize(payload, buf)
}

// Scan delegates to the Haskell scanner and rewrites the symbol it produced
// into DAML's numbering.
func (s DamlExternalScanner) Scan(payload any, lexer *gotreesitter.ExternalLexer, validSymbols []bool) bool {
	if !s.inner.Scan(payload, lexer, validSymbols) {
		return false
	}
	if !s.remapd {
		return true
	}
	if mapped, ok := s.remap[lexer.ResultSymbol()]; ok {
		lexer.SetResultSymbol(mapped)
	}
	return true
}
