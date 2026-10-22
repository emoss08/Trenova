# Formula language service

The Formula Studio's editor understands the formula language as you type: it underlines what
is wrong, offers the fix, shows what a name is, helps with a call's arguments and formats the
formula. None of that is written in TypeScript. The language (its grammar, types, checks,
messages, formatter and catalog of functions) lives in one Rust crate, `services/crates/rules`,
compiled to the WebAssembly module the server prices with. The browser loads the same module
and asks it (`client/packages/shared/src/rules/formula.ts`). The editor only draws the answers.

Read [rules-engine.md](rules-engine.md) first: "The formula language" for what a formula means,
"The static formula operations" for the exact shapes and diagnostic codes, "The formula server
API" for what crosses HTTP, and "Hosts" for how the browser loads the module.

**The rule that bites:** never parse, check or format the formula language in TypeScript. A
check, a message, a function or a formatting rule is changed in the crate, with a Rust test,
and the module is rebuilt (`task rust:wasm-rules`). The browser and the server then agree
because they run the same bytes. Only two pieces of TypeScript read formula text, and both
only colour it: the CodeMirror highlighter (`expr-language.ts`) and the Desk card's
`tokenizeFormula` (`formula-template-card.ts`). They take their keywords, word operators and
function names from the module's catalog, and neither decides whether a formula is right.

## The wrappers

`@trenova/shared/rules/formula` types the five static operations. Each is synchronous on a
loaded `RulesRuntime`, and every position in and out is UTF-16 code units, which is what
CodeMirror counts.

| Wrapper | Operation | Used for |
| --- | --- | --- |
| `formulaCatalog(rules)` | `formula.catalog` | every function, built-in, disabled and reserved name; read once per loaded runtime (memoised in a `WeakMap`) and validated with `formulaCatalogSchema` |
| `formulaCheck(rules, {source, expect, schema, tables})` | `formula.check` | diagnostics with fixes, `resultType`, the reads, `bindings`, `statements`, `lookupTables`, `empty`, `pending` |
| `formulaFormat(rules, {source, width})` | `formula.format` | the format command and Nova's proposals |
| `formulaSignature(rules, {source, position})` | `formula.signature` | signature help |
| `formulaIdentifier(rules, {source, position, schema})` | `formula.identifier` | the hover card |

The answers of the four per-keystroke operations are typed, not re-validated: they come from
the committed artifact, and validating every keystroke's answer would cost more than the call.

## What the editor knows

`KnownIdentifiers` (`components/formula-editor/known-identifiers.ts`) is built by
`buildKnownIdentifiers(catalog, schema, inputs)` from two sources:

- `GET /formula-templates/schema` (`{schemaId, variables}`, through `useFormulaSchema`): the
  record's fields, and each list's item fields named `list[].field`, kept apart as
  `itemFields` for the hover card on `.weight`;
- `formula.catalog`: the documented functions the reference pane lists, every callable name
  (built-ins included), the disabled names with the checker's message, the keywords (the
  reserved words that are not functions) and the word operators.

The template's inputs are added to the variables. `schema` is what `formula.check` and
`formula.identifier` read; it is `null` until the schema has loaded, and the check then answers
`pending: true`, holds back every unknown name and field, and still checks the language. The
problems panel says names are not checked yet. `useKnownIdentifiers(rules, schemaId, inputs)`
memoises the whole on the runtime, the schema response and the inputs.

## The editor

All in `client/apps/web/src/components/formula-editor/`:

| File | What it does |
| --- | --- |
| `formula-check.ts` | `checkExpression`: `formula.check` against the editor's knowledge. The studio's analysis and the linter ask for the same text within a keystroke of each other, so the last 24 answers are kept per `KnownIdentifiers` (new knowledge starts afresh). |
| `expr-lint.ts` | The CodeMirror linter, 300 ms after the last edit: each diagnostic but `empty` becomes an underline with its translated message, and each fix an action (a `replace` fix applies its changes; a `declare` fix adds an input through `StudioContext.declareInput`). |
| `expr-hover.ts` | The hover card: `formula.identifier` names what is under the cursor and its kind, and the card shows what the catalog or the schema says about it and what it came to on the last preview. |
| `expr-signature.ts` | Signature help: `formula.signature` finds the call and the argument; the signature and description come from the catalog. A `StateField` calls the module synchronously on every cursor move. It tracks the editor's focus and leaves with it, as the hover card does, so neither floats over a dialog. |
| `editor-format.ts` | `formatEditorDocument`, bound to Shift-Alt-F and the format button. A refused formula (empty, not parsing, holding a comment) is left exactly as written, and the page says why. |
| `expr-language.ts` | The highlighter (a `StreamLanguage`, presentation only) and the completions. |
| `expr-inline-values.ts` | What each statement came to on the preview load, drawn at the end of its line from the server's `statements` trace. Nothing is evaluated in the browser. |
| `editor-theme.ts` (`languageServiceTheme`) | The chrome for all of the above, from tokens only. |

`useExprExtensions(rules, known, …)` builds the extensions once and reads the runtime and the
knowledge through getters, so a new schema or a renamed input reconfigures only the
highlighter's compartment.

## The Studio

`routes/formula-template/_components/studio/model/expression-analysis.ts` (`analyzeExpression`)
is a thin view of `formula.check`: the problems, the error and warning counts, the reads,
`bindings`, `statements`, `lookupTables`, `resultType` (Integer and Duration included) and
`pending`. The rate trace, the readiness checks, the problems panel and Nova's proposal cards
all read it, so none of them can disagree with the underlines.

- **Problems** (`sections/expression-problems.tsx`) lists the findings errors first, with each
  fix beside it, and adds what the server found when it ran the preview: the evaluation error
  (with its range when it is on the main expression) and the findings on the breakdown lines,
  placed by their `scope` (`model/preview-scope.ts`) while the line still reads as it was
  checked.
- **Nova** (`ai/formula-draft.ts`): `formattedOrAsWritten` runs every proposal through
  `formula.format` before it lands, and an input the proposal no longer reads (by the check's
  `inputReads`, so a name in a string, a comment or a `let` is no read) is let go.
- **The preview's answer** (`lib/formula-preview.ts`): `previewFailure` is the evaluation error,
  otherwise the first error-severity diagnostic; `previewWarnings` and `unguardedFields` read
  the `nullable` diagnostics.

## Waiting for the module

Render code calls the module synchronously, so every component that does so renders only
once it has loaded. `useRules()` suspends until then, behind the screen's own skeleton:

- the Studio: `StudioProvider` calls it, under a `Suspense` with `StudioSkeleton` in
  `formula-studio.tsx`, inside the route's lazy boundary;
- `ExpressionEditor` calls it too, so every editor (the main formula, and each breakdown line in
  `breakdown-definition-editor.tsx`) is safe on its own;
- Nova's `ProposalCard`, which formats and checks the proposal as it renders;
- the Desk formula template card, whose highlighter reads the catalog; the card registry's
  `Suspense` shows its skeleton.

`main.tsx` preloads the module on idle, so it is usually ready before any of these mount.
Nothing starts a load in render or `useMemo` other than through `useRules()`.

A 1,000-node formula (the parser's limit) checks in well under a millisecond in the test
runner (`rules/formula.test.ts` holds it under 50 ms), so the linter keeps its 300 ms delay
for typing, not for cost.

## Messages

Every diagnostic, fix label, function description and example, disabled-name message and
evaluation error is a `{message, args}` from the crate, whose template is the catalog key;
`task i18n` extracts the templates. An argument may itself be a `{message, args}` (a value's
description such as "a duration"). Show one only with `translateMessage(message, t)` from
`@trenova/shared/i18n/message`, which translates nested arguments first; `task i18n-check`
reports `t(x.message, ...x.args)` written by hand. Templates are ICU: never a bare `{` or `}`,
and never leading, trailing or repeated spaces (the extractor refuses them).

## Tests

`client/packages/shared/src/rules/formula.test.ts` runs the wrappers against the real module
(the Vitest setups load it from disk): the documented `formula.check` example exactly, UTF-16
positions, a pending schema, formatting, signature and identifier lookups, a nested message
translated in Spanish, the 1,000-node timing, and a handful of `testdata/conformance.json`
cases end to end with the server's `shipment.describe.json` schema. The web app's tests drive
the linter, hover, signature help, the analysis and the consumers through the same module,
with fixtures from the documented contract (`__tests__/formula-fixtures.ts`). The language
itself is tested in the crate (`tests/formula_*.rs`).
