//go:build !grammar_subset || grammar_subset_move

package grammars

import gotreesitter "github.com/odvcencio/gotreesitter"

// External token indexes for the move grammar
// (aptos-labs/tree-sitter-move-on-aptos src/scanner.c). The order must match
// the `externals` list in the grammar.
const (
	moveTokBlockDocCommentMarker = 0
	moveTokBlockCommentContent   = 1
	moveTokDocLineComment        = 2
	moveTokErrorSentinel         = 3
)

// Fallback symbol IDs, correct for the checked-in Move blob. They are only
// used when the scanner has not been bound to a Language — binding resolves
// the real IDs from the grammar, which is what keeps this correct when the
// grammar changes.
//
// Hardcoding these alone is a latent version-coupling bug: adding any symbol
// to the grammar shifts every later ID, and the scanner then returns
// _block_comment_content while the parser reads that ID as
// _block_doc_comment_marker. The parser cannot shift the token it was handed,
// retries the same states forever, and every non-empty block comment in the
// language fails to parse — while `/**/` and `//` keep working, because they
// never reach the external scanner.
const (
	moveSymBlockDocCommentMarker gotreesitter.Symbol = 151
	moveSymBlockCommentContent   gotreesitter.Symbol = 152
	moveSymDocLineComment        gotreesitter.Symbol = 153
)

// moveExternalSymbolNames is the grammar's `externals` list, in order. Used to
// resolve symbol IDs from the Language at attach time.
var moveExternalSymbolNames = []string{
	"_block_doc_comment_marker",
	"_block_comment_content",
	"_doc_line_comment",
	"_error_sentinel",
}

var moveDefaultSymTable = [4]gotreesitter.Symbol{
	moveSymBlockDocCommentMarker,
	moveSymBlockCommentContent,
	moveSymDocLineComment,
	0,
}

// MoveExternalScanner ports the stateless upstream scanner: block doc-comment
// markers (`/**` but not `/***` or `/**/`), nestable block comment content,
// and doc line comment bodies (`/// ...` up to and including EOL).
type MoveExternalScanner struct {
	symbols [4]gotreesitter.Symbol
	bound   bool
}

// ExternalScannerForLanguage binds this scanner to lang, resolving each
// external token's symbol ID from the grammar rather than trusting the
// compiled-in constants. Implementing this interface is what lets the same
// scanner serve any revision of the Move grammar.
func (MoveExternalScanner) ExternalScannerForLanguage(lang *gotreesitter.Language) gotreesitter.ExternalScanner {
	s := MoveExternalScanner{symbols: moveDefaultSymTable, bound: true}
	bindExternalScannerSymbolNames(lang, moveExternalSymbolNames, func(tokenIdx int, sym gotreesitter.Symbol) {
		if tokenIdx >= 0 && tokenIdx < len(s.symbols) {
			s.symbols[tokenIdx] = sym
		}
	})
	return s
}

// symbolTable returns the bound symbol IDs, or the compiled-in defaults when
// the scanner was used without being bound to a Language.
func (s MoveExternalScanner) symbolTable() [4]gotreesitter.Symbol {
	if s.bound {
		return s.symbols
	}
	return moveDefaultSymTable
}

func (MoveExternalScanner) Create() any                           { return nil }
func (MoveExternalScanner) Destroy(payload any)                   {}
func (MoveExternalScanner) Serialize(payload any, buf []byte) int { return 0 }
func (MoveExternalScanner) Deserialize(payload any, buf []byte)   {}

// The scanner carries no payload and derives every result from local
// lookahead plus validSymbols, so every incremental boundary is quiescent and
// failed scans cannot mutate persistent state.
func (MoveExternalScanner) SupportsIncrementalReuse() bool    { return true }
func (MoveExternalScanner) ExternalScannerIsStateless() bool  { return true }
func (MoveExternalScanner) PreservesStateOnScanFailure() bool { return true }

func (s MoveExternalScanner) Scan(payload any, lexer *gotreesitter.ExternalLexer, validSymbols []bool) bool {
	// Error recovery state: bail out, exactly like the C scanner.
	if moveValid(validSymbols, moveTokErrorSentinel) {
		return false
	}

	if moveValid(validSymbols, moveTokDocLineComment) {
		return moveScanLineDocContent(lexer, s.symbolTable())
	}

	matched := false
	if moveValid(validSymbols, moveTokBlockDocCommentMarker) {
		matched = moveScanBlockDocCommentMarker(lexer, s.symbolTable())
	}
	if !matched && moveValid(validSymbols, moveTokBlockCommentContent) {
		matched = moveScanBlockCommentContent(lexer, s.symbolTable())
	}
	return matched
}

// moveScanBlockDocCommentMarker matches the `*` of `/**` provided it is not
// followed by `/` (empty comment `/**/`) or another `*`.
func moveScanBlockDocCommentMarker(lexer *gotreesitter.ExternalLexer, syms [4]gotreesitter.Symbol) bool {
	if lexer.Lookahead() != '*' {
		return false
	}
	lexer.Advance(false)
	lexer.MarkEnd()
	if lexer.Lookahead() == '/' || lexer.Lookahead() == '*' {
		return false
	}
	lexer.SetResultSymbol(syms[moveTokBlockDocCommentMarker])
	return true
}

// moveScanBlockCommentContent munches nestable block comment content. The
// outermost closing `*/` is excluded (MarkEnd before consuming it) so
// tree-sitter can recognise it as its own token.
func moveScanBlockCommentContent(lexer *gotreesitter.ExternalLexer, syms [4]gotreesitter.Symbol) bool {
	depth := 1
	for lexer.Lookahead() != 0 && depth > 0 {
		switch lexer.Lookahead() {
		case '*':
			if depth == 1 {
				lexer.MarkEnd()
			}
			lexer.Advance(false)
			if lexer.Lookahead() == '/' {
				depth--
				lexer.Advance(false)
			}
		case '/':
			lexer.Advance(false)
			if lexer.Lookahead() == '*' {
				lexer.Advance(false)
				depth++
			}
		default:
			lexer.Advance(false)
		}
	}
	if depth > 0 {
		// Unterminated comment: everything scanned is content.
		lexer.MarkEnd()
		return false
	}
	lexer.SetResultSymbol(syms[moveTokBlockCommentContent])
	return true
}

// moveScanLineDocContent consumes a doc line comment body up to and including
// the EOL character (always matches).
func moveScanLineDocContent(lexer *gotreesitter.ExternalLexer, syms [4]gotreesitter.Symbol) bool {
	lexer.SetResultSymbol(syms[moveTokDocLineComment])
	for lexer.Lookahead() != 0 {
		if moveIsEOL(lexer.Lookahead()) {
			lexer.Advance(false)
			break
		}
		lexer.Advance(false)
	}
	return true
}

// moveIsEOL reports end-of-line characters; EOF is not EOL.
func moveIsEOL(ch rune) bool { return ch == '\n' || ch == 0x2028 || ch == 0x2029 }

func moveValid(vs []bool, i int) bool { return i < len(vs) && vs[i] }
