//go:build cgo && treesitter_c_parity

package cgoharness

import (
	"fmt"
	"strings"
	"testing"
)

// TestCSharpLongArrayInitializerParity is a scoped correctness check for the
// _for_statement_conditions_repeat1 shift/reduce fold added to
// csharpRepeatConflictKind (parser.go). That fold targets exactly the
// {reduce _for_statement_conditions_repeat1, repetition-shift} conflict at
// lookahead ',' -- profiled live (AmbiguityProfile) as the actual GLR
// fork-explosion source for real BouncyCastle.NET files (Sha256Digest.cs,
// AesEngine.cs, ...): tree-sitter-c-sharp's LALR table merges this repeat
// state across _for_statement_conditions' commaSep1($.expression) lists and
// initializer_expression's commaSep($.expression) element list, since both
// reduce a structurally identical 2-child (list, ',', expr) repeat. This
// test proves the folded parse still matches the real upstream C parser
// byte-for-byte on a long initializer list, not just that it's fast.
func TestCSharpLongArrayInitializerParity(t *testing.T) {
	elems := make([]string, 60)
	for i := range elems {
		elems[i] = fmt.Sprintf("%d", i)
	}
	src := []byte(fmt.Sprintf("class C { static byte[] S = { %s }; }\n", strings.Join(elems, ", ")))
	tc := parityCase{name: "c_sharp", source: string(src)}
	runParityCase(t, tc, "long-array-initializer-60-elements", src)
}

func TestCSharpLongObjectInitializerParity(t *testing.T) {
	fields := make([]string, 40)
	for i := range fields {
		fields[i] = fmt.Sprintf("F%d = %d", i, i)
	}
	src := []byte(fmt.Sprintf("class C { static Foo S = new Foo { %s }; }\n", strings.Join(fields, ", ")))
	tc := parityCase{name: "c_sharp", source: string(src)}
	runParityCase(t, tc, "long-object-initializer-40-fields", src)
}

// TestCSharpForStatementMultiConditionParity exercises the grammar rule the
// folded repeat state's name is literally taken from
// (_for_statement_conditions_repeat1), to prove the fold does not regress
// real for-loop initializer/update comma lists even though profiling showed
// short for-loops barely reach this state.
func TestCSharpForStatementMultiConditionParity(t *testing.T) {
	src := []byte("class C { void M() { for (int i = 0, j = 1, k = 2; i < 10 && j < 20; i++, j += 2, k *= 2) { } } }\n")
	tc := parityCase{name: "c_sharp", source: string(src)}
	runParityCase(t, tc, "for-statement-multi-condition", src)
}

// TestCSharpNestedInitializerInsideForLoopParity mixes both grammar origins
// of the shared repeat state in one parse (an object/collection initializer
// literal used as a for-loop's initializer expression), the shape most
// likely to expose any cross-context state confusion the fold could
// introduce.
func TestCSharpNestedInitializerInsideForLoopParity(t *testing.T) {
	src := []byte(`class C {
  void M() {
    for (int[] a = { 1, 2, 3, 4, 5, 6, 7, 8 }, i = a.Length; i > 0; i--) { }
  }
}
`)
	tc := parityCase{name: "c_sharp", source: string(src)}
	runParityCase(t, tc, "nested-initializer-inside-for-loop", src)
}
