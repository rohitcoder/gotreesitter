# Grammar patches for Sui Move and Cairo

CBOMkit parses contract source with `gotreesitter`, a cgo-free Go
reimplementation of tree-sitter. Its grammars ship as precompiled binary
blobs (`grammars/grammar_blobs/*.bin`), generated from tree-sitter grammar
definitions — they are not editable Go.

Two of those blobs cannot parse the language dialects our customers
actually write. This directory holds the source-level patches that fix
them, so the change is reviewable as a diff rather than as an opaque
binary.

## What was broken, measured

| Corpus | Before | After |
|---|---|---|
| MystenLabs/sui — files calling crypto | 0/21 | 5/21 |
| OpenZeppelin/cairo-contracts | 35/319 (11.0%) | 184/319 (57.7%) |
| aptos-core framework (regression check) | 248/253 | 248/253 — unchanged |

The Sui number matters more than it looks. The bundled Move grammar is
`aptos-labs/tree-sitter-move-on-aptos`, an **Aptos-dialect** grammar. Sui
uses Move 2024's label-style `module a::b;` declaration, which it cannot
parse at all. Aggregate parse rates over the whole Sui repo look
reasonable (~50%) because most files are tests and fixtures; the subset
that actually calls `sui::ed25519` and friends — the only files a CBOM
cares about — was at **zero**.

## The patches

**Move** (6 changed lines) — `module_body` accepts a `;` label form in
addition to a brace block:

```js
module_body: $ =>
    choice(
        seq('{', repeat($._module_member), '}'),
        prec.right(seq(';', repeat($._module_member)))
    ),
```

**Cairo** (71 changed lines) — the upstream grammar
(`amaanq/tree-sitter-cairo`, last commit April 2024) predates Cairo 2.x.
Its author scaffolded `optional($.visibility_modifier)` at ten item sites
and commented every one out, never defining the rule. The patch defines it
and uncomments those sites, then adds:

- `impl_bound_parameter` — `+Drop<T>` / `-Drop<T>` anonymous impl bounds
- `macro_invocation` / `token_tree` — `array![]`, `selector!(...)`
- `macro_statement` — `component!(path: X, ...)` at item level
- visibility on function definitions, signatures and `impl` items

## Regenerating the blobs

Requires Node (for the tree-sitter CLI). `ts2go` ships inside gotreesitter.

```sh
# 1. tree-sitter CLI (the native `tree-sitter` npm binding fails to build
#    on recent Node; only the CLI is needed)
npm install --no-save tree-sitter-cli@0.26.5

# 2. build ts2go from the pinned gotreesitter
go build -o ts2go github.com/odvcencio/gotreesitter/cmd/ts2go

# 3. apply the patch to the upstream grammar, then
tree-sitter generate                       # -> src/parser.c
./ts2go -input src/parser.c -output move_patched.go \
        -package grammars -name move       # -> grammar_blobs/move.bin
```

`ts2go` also emits a `*_external_lex_states_gen.go` sidecar. Copy it
alongside the blob — block comments are lexed by an external scanner and
the sidecar must match the regenerated state table.

## Known limits — deliberately not shipped

Going past this cost more than it returned:

- **Sui at 81%** is reachable by also adding `mut` bindings and
  `macro fun`, but that build broke inline `/* */` comments inside
  expressions and cost 4 Aptos files (248 → 244). Traced to the
  external-scanner lex-state mapping; regenerating the sidecar did not
  fix it. Not shipped — a net-negative trade.
- **Cairo's remaining 42%** needs `if let`, `for ... in`, const generic
  arrays and several macro forms.

Both are additive work on top of these patches, not rewrites.

## Why vendor rather than upstream

Upstream `odvcencio/gotreesitter` is actively maintained (commits daily),
so the Move patch is worth proposing there. But the Cairo grammar it
sources from — `amaanq/tree-sitter-cairo` — has been untouched since
April 2024 and shows no sign of adopting Cairo 2.x. Waiting on it would
block Starknet coverage indefinitely.

Vendoring keeps CBOMkit's parsing behaviour pinned to something we
control and can re-derive from these diffs. Upstreaming the Move change
remains worthwhile and does not conflict.
