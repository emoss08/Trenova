# Formula language service

The Formula Studio's editor understands the formula language itself, in the browser, rather
than waiting for the engine to refuse an expression. Everything lives in
`client/apps/web/src/components/formula-editor/`:

| File | What it does |
| --- | --- |
| `expr-parser.ts` | A tokenizer and Pratt parser for expr-lang as the engine runs it (`let` lines, `??`, the ternary, `if … else`, calls, members, slices, predicates, the word operators). It never throws: a malformed expression yields a tree with `missing` nodes and diagnostics anchored to the span they are about, with a one-line fix where one exists (`;` after a `let`, `==` for `=`). |
| `expr-check.ts` | The checks over the tree: unknown names with a suggestion and a declare-as-input fix, unknown fields under a known root, unknown functions, the expr builtins the engine turns off (`now`, `toJSON`, `fromJSON`, `repeat`), arity, operand types (`%` on a value that can hold a fraction, `==` between kinds the engine refuses, a date moved by a bare number) (a flag multiplied, text compared with a number, a number used as a test), a field that can be empty read without a guard, `let` rules (duplicate, unused, read before defined, shadowing a field, ending on a `let`), a value worked out and thrown away, and a result that is not an amount. It also maps the engine's `(line:column)` errors back onto the expression. |
| `expr-lint.ts` | Turns the findings into CodeMirror diagnostics with actions; the same `lintExpression` feeds the studio's problems panel, so the squiggles and the list never disagree. |
| `expr-hover.ts` | The hover card: what a name is, its type, whether it can be empty, its description, and what it resolved to on the last preview. |
| `expr-signature.ts` | Signature help while the cursor is inside a call's parentheses. It tracks the editor's focus (`EditorView.focusChangeEffect`) and leaves with it, as does the hover card, so neither floats over a dialog opened from the page. |
| `expr-format.ts` | The formatter: it prints the tree back one statement per line, a space around every operator, and breaks a line past 80 columns where it reads best (a sum one term per line with the operator leading, a call one argument per line, a ternary one case per line, a `let` after its `=` when the value then fits). The author's parentheses, literals and operator spellings survive; it refuses a formula that does not parse or holds a comment, since either would lose text. Bound to Shift-Alt-F and the format button under the editor, and run over every proposal Nova makes before it lands (`formattedOrAsWritten` in the studio's `formula-draft.ts`). |
| `expr-inline-values.ts` | What each statement came to on the preview load, drawn at the end of its line from the engine's `statements` trace (`traceStatements: true` on the preview request). The values are the engine's; nothing is evaluated in the browser. |
| `editor-theme.ts` (`languageServiceTheme`) | The chrome for all of the above, from tokens only. |

Keep the parser in step with the engine's grammar (`expr-lang` in `services/tms/go.mod`; the
operator table is in `parser/operator/operator.go`, the tokens in `parser/lexer/state.go`).

Nothing about functions is kept in the browser. `GET /formula-templates/schema` (the engine's
`DescribeSchema`) publishes every callable name with its shape: the documented functions
(`engine/functionmeta.go`; executable ones take their arity, result type and numeric arguments
from their Go signatures), the expr builtins the engine keeps (`engine/functionshape.go`, held
equal to expr's own list by a test), and the builtins it turns off with a reason and a
replacement (`engine/disabled.go`). `buildKnownIdentifiers` reads that; until it has loaded,
`ready` is false, no name is called unknown, and the problems panel says names are not checked
yet.

**The conformance corpus** (`services/tms/internal/modules/rating/app/formula/testdata/conformance.json`)
is what keeps the two in agreement. Each case says whether the engine saves an expression and,
where it matters, what it comes to; `conformance_test.go` holds the engine to it and
`__tests__/conformance.test.ts` holds the editor to it (an error exactly where the engine
refuses). The editor tests read the engine's schema from `testdata/shipment.describe.json`, which
`describe_golden_test.go` keeps equal to `DescribeSchema("shipment")`; regenerate it with
`go test ./internal/modules/rating/app/formula/ -run TheEditorFixture -update-description`
after changing the schema or a function. A change to either side that breaks agreement adds a
case first. Word operators the schema publishes beyond
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
