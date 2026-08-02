const id_char = /[\pL\p{Mn}\pN_']*/

const varid_start_char = /[_\p{Ll}\p{Lo}]/

const conid_start_char = /[\p{Lu}\p{Lt}]/

module.exports = {

  variable: _ => token(seq(varid_start_char, id_char, /#*/)),

  implicit_variable: _ => token(seq('?', varid_start_char, id_char)),

  name: _ => token(seq(conid_start_char, id_char, /#*/)),

  label: _ => token(seq('#', varid_start_char, id_char)),

  _carrow: _ => choice('=>', '⇒'),
  _arrow: _ => choice('->', '→'),
  _linear_arrow: _ => choice('->.', '⊸'),
  _larrow: _ => choice('<-', '←'),
  // DAML writes type annotations with a single colon, not Haskell's double:
  // `f : Int -> Int`, `party : Party`. Measured against digital-asset/daml,
  // the SDK contains 1847 single-colon signatures and zero double-colon
  // ones, and 2437 single-colon record fields. The upstream grammar is
  // derived from tree-sitter-haskell and kept only the Haskell forms, which
  // is why every file with a type signature failed to parse.
  //
  // Both are accepted rather than only the single colon: `::` remains valid
  // DAML for operator sections and appears in code ported from Haskell, so
  // rejecting it would trade one parse failure for another.
  // DAML writes type annotations with a single colon rather than Haskell's
  // double: `f : Int -> Int`, `party : Party`. Measured against
  // digital-asset/daml, the SDK contains 1847 single-colon signatures and
  // zero double-colon ones, and 2437 single-colon record fields.
  //
  // token(prec(1, ':')) rather than a bare ':' is what makes this work. The
  // external scanner classifies any symbol starting with ':' as a
  // constructor operator — correct for Haskell, where `:` is cons — so a
  // plain string literal here loses the race and `f : Int -> Int` parses as
  // an infix expression with `->` left over. Raising the token's precedence
  // makes the grammar's own lexer win for a bare colon while leaving longer
  // operators such as `:|` and `:+:` to the scanner.
  _colon2: _ => choice(token(prec(1, ':')), '::', '∷'),
  _promote: _ => '\'',

  _qual_dot: $ => seq($._cond_qual_dot, '.'),
  _tight_dot: $ => seq($._cond_tight_dot, '.'),
  _any_tight_dot: $ => choice($._qual_dot, $._tight_dot),
  _prefix_dot: $ => seq($._cond_prefix_dot, '.'),
  _any_prefix_dot: $ => choice($._qual_dot, $._prefix_dot),

  _tight_at: $ => seq($._cond_tight_at, '@'),
  _prefix_at: $ => seq($._cond_prefix_at, '@'),

  _prefix_bang: $ => seq($._cond_prefix_bang, '!'),
  _tight_bang: $ => seq($._cond_tight_bang, '!'),
  _any_prefix_bang: $ => choice($._prefix_bang, $._tight_bang),

  _prefix_tilde: $ => seq($._cond_prefix_tilde, '~'),
  _tight_tilde: $ => seq($._cond_tight_tilde, '~'),
  _any_prefix_tilde: $ => choice($._prefix_tilde, $._tight_tilde),

  _prefix_percent: $ => seq($._cond_prefix_percent, '%'),

  _dotdot: $ => seq($._cond_dotdot, '..'),

  _paren_open: $ => seq(alias(/\(/, '('), $._cmd_texp_start),
  _paren_close: $ => seq(alias(/\)/, ')'), $._cmd_texp_end),
  _bracket_open: $ => seq('[', $._cmd_texp_start),
  _bracket_close: $ => seq(']', $._cmd_texp_end),

  // Sadly, this does not have the effect of creating a single terminal for the bracket :'(
  _unboxed_open: $ => alias(seq($._paren_open, token.immediate('#')), '(#'),
  _unboxed_close: $ => seq('#)', $._cmd_texp_end),
  _unboxed_bar: _ => choice('|', token.immediate('|')),

  _where: $ => seq(optional($._phantom_where), 'where'),

  _bar: $ => seq(optional($._phantom_bar), '|'),

}
