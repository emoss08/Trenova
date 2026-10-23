# Formula language service

The Formula Studio and the library studio edit one Tariff file each. As you type, the editor
colours it, underlines what is wrong and offers the fix, completes what may be written at the
cursor, shows what a name is and what it came to on the last preview, helps with a call's
arguments, jumps to a declaration (into an imported library), renames a binding everywhere,
draws inferred types and parameter names inline, folds blocks and formats the file. Beside it the
outline, the tests panel and the readiness list follow every change. None of that is written in
TypeScript. The language (its lexer, parser, checker, evaluator, formatter, editor features,
messages and standard library) lives in one Rust crate, `services/crates/rules`, compiled to the
WebAssembly module the server rates with. The browser loads the same module and asks it
(`client/packages/shared/src/rules/tariff.ts`). The editor only draws the answers.

Read [formula-language.md](formula-language.md) first for what a Tariff file means ("Module
operations" lists every `tariff.*` operation and "Wire values" their value forms), and
[rules-engine.md](rules-engine.md) "Hosts" for how the browser loads the module.

**The rule that bites:** never lex, parse, check, evaluate or format Tariff in TypeScript. A
check, a message, a completion, a formatting rule or a standard library function is changed in
the crate, with a Rust test, and the module is rebuilt (`task rust:wasm-rules`). The browser and
the server agree because they run the same bytes. Even colour comes from the module
(`tariff.tokens`), so there is no TypeScript grammar to drift.

## The wrappers

`@trenova/shared/rules/tariff` types every operation the client calls. Each is synchronous on a
loaded `RulesRuntime`; every position and range in and out is UTF-16 code units, which is what
CodeMirror counts. A refused request throws `RulesRequestError {op, code, detail}`.

| Wrapper | Operation | Used for |
| --- | --- | --- |
| `tariffCheck(rules, {source, kind, context})` | `tariff.check` | diagnostics with fixes, the header, the declared inputs, the imports, the outline (items and folding) and the charges |
| `tariffTokens(rules, source)` | `tariff.tokens` | the editor's highlighting and every read-only view of a source (`TariffCode`) |
| `tariffFormat(rules, source)` | `tariff.format` | the format command and Nova's proposals; `formatted: null` with the syntax errors that stop it |
| `tariffTest(rules, {source, context})` | `tariff.test` | the tests panel (tests and properties, a property's cases and counterexample), the outline's dots, readiness, the coverage gutter and Nova's proposal card |
| `tariffScaffoldTest(rules, {source, context, record, inputs, name})` | `tariff.scaffoldTest` | "Save as test" on the sample load (a real shipment is scaffolded on the server) |
| `tariffCatalog(rules)` | `tariff.catalog` | the reference pane and the definition dialog for a standard library name; memoised per runtime in a `WeakMap` |
| `tariffComplete`, `tariffHover`, `tariffSignature`, `tariffDefinition`, `tariffRename`, `tariffPrepareRename`, `tariffInlays` | `tariff.complete`, `.hover`, `.signature`, `.definition`, `.rename` (with and without `newName`), `.inlays` | the editor's features at a cursor or over the visible range |
| `tariffReferences(rules, {source, kind, context, offset, includeDeclaration?})` | `tariff.references` | the marks on every use of the name at the cursor and Find references |
| `tariffActions(rules, {source, kind, context, from, to})` | `tariff.actions` | Promote to input: the refactorings for a selection, each a set of edits and the range to select after |

`wireContext` sends the context the check was given: the libraries (path, version id, source),
the rate tables' shapes, the accessorial codes (left out while unknown, so codes go unchecked)
and the settings. A file kind (`tariff` or `library`) is sent to every editor operation, so a
library's header is never read as a tariff's.

## The context

`useTariffCheckContext()` (`apps/web/src/hooks/use-tariff-check-context.ts`) reads
`GET /formula-templates/context` once per five minutes. The server answers `{context,
libraries}`: the context the module checks against, and each approved library as the Studio
names it (`path`, `libraryId`, `name`, `description`, `versionNumber`). The hook answers
`{context, libraries, isError, refetch}`. `context` is `null` until it has loaded; every caller
then checks against `EMPTY_TARIFF_CONTEXT`, and a failed load shows a warning with Retry above
the source, because the check cannot see the organisation's libraries, tables or codes.
Organisation libraries come in their approved versions; a library is never in its own context.

## The editor

All in `client/apps/web/src/components/formula-editor/`. `TariffEditor` takes `kind`, `value`,
`onChange`, `context`, the last preview's `trace`, the last test run's `coverage`, `highlight`
(a range something outside the editor points at), `onView`, `onOpenDefinition` and `onFormat`, and builds its extensions once,
reading the runtime, the kind, the context and the trace through getters
(`TariffLanguageService` in `tariff-service.ts`), so new knowledge never rebuilds the editor.
`TariffEditorField` is the react-hook-form binding.

| File | What it does |
| --- | --- |
| `tariff-service.ts` | `checkTariffSource`: `tariff.check` with an LRU of four answers keyed by source, kind and context identity, so the linter, the folding, the Studio and the outline ask once per text |
| `tariff-highlight.ts` | a `ViewPlugin` over `tariff.tokens`: the module is asked once per text and only the tokens in the visible ranges become decorations (`visibleTokenDecorations`), classed `cm-tariff-<kind>` and `cm-syntax-<role>` |
| `tariff-lint.ts` | the linter, 250 ms after the last edit: each finding becomes an underline with its translated message and its fixes as actions (`applyTariffFix`, one undoable change); a library's finding is placed on the `use` line that imports it (`toTariffLint`) |
| `tariff-complete.ts` | completion: the module's items in its order (`sort`), filtered client-side by what is typed, a snippet's `${1:x}` fields as tab stops, and an item's own `range` (`.` rewritten to `?.` on an optional) replacing the list's |
| `tariff-hover.ts` | the hover card: the declaration as Tariff code coloured from the module's tokens (`appendTariffCode`), the type and unit, the doc, and what a binding came to on the last preview (`values` is that preview's trace) |
| `tariff-signature.ts` | signature help: the parameter being written in bold, its doc under it; a `StateField` asks on every cursor move |
| `tariff-navigate.ts` | F12 or Mod-click: `tariff.definition` moves to a declaration in this file, or opens the library it is declared in (`onOpenDefinition({module, range, name})`); F2: prepare, then a rename panel that applies every edit as one change or shows why the name cannot be renamed |
| `tariff-inlays.ts` | inferred types after bindings, parameter names before positional arguments and, with a trace, each binding's value, for the visible range; `refreshTariffInlays` redraws them when the trace changes |
| `tariff-fold.ts` | folding from the outline's `folding` ranges (`foldAt` keeps the braces) |
| `tariff-format.ts` | Shift-Alt-F and the format button; a file with syntax errors is left as written and the caller is told why |
| `tariff-references.ts` | once the cursor rests 300 ms on a name, `tariff.references` marks its uses in the file (`cm-tariff-occurrence`, a write or the declaration also `-write`, on the `--editor-reference` tokens); Shift-F12 (`findReferences`) hands the whole list to the service's `showReferences` |
| `references-dialog.tsx` | Find references: the uses "In this file" and "In libraries", each with its line, the use set in the reference tint; Enter or a click goes to a use in this file or opens the library at it (`onOpenDefinition`), the arrow keys, Home and End move between rows. `TariffEditor` owns it, so every editor has it |
| `tariff-actions.ts` | Promote to input (Mod-.): `tariff.actions` for the selection, a menu at the cursor (arrows, Enter, Escape), the chosen action's edits as one change (`userEvent: "refactor"`), then rename opened at its `select` |
| `tariff-notice.ts` | the one-line notice at the cursor when a command has nothing to do there ("Nothing here can be promoted to an input"); it leaves with the next edit, move or click, or after 2.5 s |
| `tariff-coverage.ts` | the last test run's coverage: a gutter bar (`--editor-coverage-gap`) beside each line of the outermost regions no case reached, a tooltip naming the kind of region, the regions mapped through edits until the next run; with no coverage the gutter is reconfigured away (`setTariffCoverage`) |
| `tariff-doc.ts` | `tariffDocText` and `appendTariffDoc` (an author's doc as written, a built-in's `Message` translated), and `unlessUnanswered` |
| `editor-theme.ts` | the chrome for all of the above, from tokens only |

`unlessUnanswered` wraps every assisting call. A request the module refuses is a caller bug, so it
is logged once per operation for the developer, but it never breaks typing: the editor carries on
as if the operation had nothing to offer. A trap or any other failure still throws.

## The Studios

The template studio (`routes/formula-template/_components/studio/`) and the library studio
(`routes/formula-library/_components/studio/`) share their pieces from
`apps/web/src/components/tariff-studio/`:

- `useTariffWorkbench({rules, source, kind, context, tablesStamp})`: the check, the error and
  warning counts, the test count, the tests (`useTariffTests`, run with `tariff.test` once the
  source has rested 400 ms, and not at all while it has errors) and each test's verdict by name.
  The context carries only the shapes of tables with an exact axis, so a run that answers
  `needs` loads those cells (`useTariffTableCells`, `POST /formula-templates/context/cells`,
  cached in the query cache per rate table stamp), lays them into the context and runs again:
  three loads by combination, then the tables whole, as the server does. A test waiting for
  cells has status `needs`; the tests panel says so, and offers Retry when a load fails. Everything a
  Studio shows (the underlines, the outline, the tests panel, readiness, Nova's card) reads this
  one answer, so none can disagree.
- Properties are tests everywhere: the workbench's `testCount` counts `test` and `property` outline
  items (`isTestItem`, `lib/tariff-outline.ts`), a failed property fails the run, blocks review and
  submit and draws a red dot. The workbench hands the editor `coverage` (the run's, or null while
  there is none). `TestsPanel` marks a property and its cases (and skipped), shows a failed one's
  `Counterexample` (each varied value, a monotone failure's out-of-order pair, a refusal) with
  "Save as test" (`onSaveAsTest`, the counterexample's `test` appended through the binding), and
  "Tests reach {reached} of {total} regions" with the unreached regions to reveal
  (`unreachedRegions`, outermost first, `lib/tariff-coverage.ts`). `tariffReadiness` adds a
  `coverage` warning listing those regions; the submit dialogs show it beside the server's checks
  where the server sends no `coverage` row (`withLiveChecks`), each region a link into the source.
- `SourceStatusBar` carries the editor's menu (`EditorActionsMenu`: Go to definition, Find
  references, Rename and, in a tariff, Promote to input, each with its key).
- `ChoicesView` (`components/tariff-bill/`) draws what each `best` or `worst` chose: the preview's
  `choices` (the module's wire form) under the bill, and the shipment's
  `ratingDetail.choices` (GraphQL, read by `readStoredTariffChoices`) on its billing card.
- `useTariffEditorBinding({read, write})`: reveal a range, apply a fix and append a block
  (a test from "Save as test", Nova, or the tests panel's "Add a test") through the editor, so each
  lands in its undo history; before the editor has loaded they are written to the form.
- `OutlinePanel`, `ProblemsList`, `TestsPanel`, `ReferencePane` (the context's tables, codes and
  libraries, the standard library and the language from `tariff.catalog`), `DefinitionDialog`,
  `SourceStatusBar`, `ReadinessView` and `tariffReadiness` (the live copy of the server's
  readiness, keyed as the server keys it; a library must have a test, a tariff only warns),
  `ReviewDiffView`, `ReviewHistoryView`, `SourceDiff`, `TariffNovaPane` and `ProposalCard`, the
  skeletons and the shortcuts.

Nova works on the same file two ways. Its conversation (`TariffNovaPane`, a thread opened with
`POST /formula-templates/ai/thread/` or `POST /formula-libraries/ai/thread/`) reads the editor
as the page draft (`formula` or `formulaLibrary`) and answers with a `formula_propose` or
`formula_library_propose` edit carrying a `SourceProposal`. The server offers one only when it
checks clean and every one of its tests (at least one) passes; `ProposalCard` lands it in the
editor, draws the changed lines with the unchanged runs folded (`diffHunks`), checks and tests it
again against the Studio's context and, for a tariff, rates it on the sample load and three
shipments. The template studio's one-shots answer in place: Explain (`/ai/explain`, the preview
with one sentence per traced step), Suggest tests (`/ai/suggest-tests`, blocks already run
against the draft, added one by one), Describe inputs (`/ai/describe-inputs`, doc comments
written above the inputs as one undoable change, `inputDocEdits`), Why does this fail
(`/ai/explain-test`, under a failing test) and the reviewer's note (`/:id/review-note`).

The preview is rated on the server (`POST /formula-templates/test`): the sample record or a real
shipment, with the sample inputs; the answer is the bill, a refusal or an error, the trace and the
lookups, and the server's run of the tests. `BillView` and `TraceView`
(`components/tariff-bill/`) draw it, and a range in the trace or a bill line lights the source.

## Waiting for the module

Render code calls the module synchronously, so every component that does so renders only once
it has loaded. `useRules()` suspends until then, behind the screen's own skeleton: the Studios'
providers (`StudioSkeleton`), `TariffEditor`, `TariffCode` (the Desk card and every read-only
source), Nova's `ProposalCard` and the definition dialog. `main.tsx` preloads the module on idle,
so it is usually ready first. Nothing starts a load in render or `useMemo` other than through
`useRules()`.

## Messages

Every diagnostic, fix label, refusal, evaluation error and built-in doc is a `{message, args}`
from the crate, whose template is the catalog key; `task i18n` extracts the templates. An argument
may itself be a message. Show one only with `translateMessage(message, t)` from
`@trenova/shared/i18n/message`. An author's doc comment and a charge label are the author's words
and are shown as written.

## Tests

`client/packages/shared/src/rules/tariff.test.ts` runs every wrapper against the real module (the
Vitest setups load it from disk): the check's inputs and outline, a failing test's values, a
rated preview with its trace, formatting, tokens, the catalog, scaffolding and each editor
operation. `components/formula-editor/__tests__/tariff-editor-features.test.ts` drives each editor
feature in a mounted `EditorView` against the same module: completion (with the `?.` rewrite and a
refused request), hover, signature help, go to definition in the file and into a library, rename
and its refusal, inlay hints, folding, lint placement, highlighting of the visible range and
formatting; `tariff-editor-refactor.test.ts` the marks at the cursor, Find references, Promote to
input (one change, then rename), the coverage gutter and its tooltip, and the new keywords and
snippets; `tariff-editor.test.tsx` the references dialog and the coverage prop through
`TariffEditor` itself. The language itself is tested in the crate (`tests/tariff_*.rs`).
