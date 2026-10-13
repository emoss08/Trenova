# Formula language service

The Formula Studio's editor understands the formula language itself, in the browser, rather
than waiting for the engine to refuse an expression. Everything lives in
`client/apps/web/src/components/formula-editor/`:

| File | What it does |
| --- | --- |
| `expr-parser.ts` | A tokenizer and Pratt parser for expr-lang as the engine runs it (`let` lines, `??`, the ternary, `if … else`, calls, members, slices, predicates, the word operators). It never throws: a malformed expression yields a tree with `missing` nodes and diagnostics anchored to the span they are about, with a one-line fix where one exists (`;` after a `let`, `==` for `=`). |
| `expr-check.ts` | The checks over the tree: unknown names with a suggestion and a declare-as-input fix, unknown fields under a known root, unknown functions, arity against the signature the schema publishes, operand types (a flag multiplied, text compared with a number, a number used as a test), a field that can be empty read without a guard, `let` rules (duplicate, unused, read before defined, shadowing a field, ending on a `let`), a value worked out and thrown away, and a result that is not an amount. It also maps the engine's `(line:column)` errors back onto the expression. |
| `expr-lint.ts` | Turns the findings into CodeMirror diagnostics with actions; the same `lintExpression` feeds the studio's problems panel, so the squiggles and the list never disagree. |
| `expr-hover.ts` | The hover card: what a name is, its type, whether it can be empty, its description, and what it resolved to on the last preview. |
| `expr-signature.ts` | Signature help while the cursor is inside a call's parentheses. |
| `expr-inline-values.ts` | What each statement came to on the preview load, drawn at the end of its line from the engine's `statements` trace (`traceStatements: true` on the preview request). The values are the engine's; nothing is evaluated in the browser. |
| `editor-theme.ts` (`languageServiceTheme`) | The chrome for all of the above, from tokens only. |

Keep the parser in step with the engine's grammar (`expr-lang` in `services/tms/go.mod`; the
operator table is in `parser/operator/operator.go`). Word operators the schema publishes beyond
the built-in four (`startsWith`, `endsWith`, `contains`, `matches`) are passed in through
`KnownIdentifiers.operatorNames` and parsed at comparison precedence.

Diagnostic messages are user-facing and go through `translate`, so they are part of the i18n
catalogs. A message is an ICU pattern: never put a bare `{` or `}` in one; pass the brace as an
argument (`translate("'{0}' opened here is never closed", "{")`).

The studio reads the service through `routes/formula-template/_components/studio/model/expression-analysis.ts`
(`analyzeExpression`), which is also what the rate trace, the readiness checks and Nova's
proposal cards use. The problems panel under the editor (`sections/expression-problems.tsx`)
lists the findings errors first, folds a long list to its first rows, and offers each finding's
fix beside it; a declare fix goes through `StudioContext.declareInput`, which adds the input and
takes the person to its row.
