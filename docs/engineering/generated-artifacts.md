# Generated Artifacts and the Checks That Guard Them

Most of this repository's generated code is guarded by a CI job that regenerates it and
fails if the result differs from what you committed. The failure text names the command to
run, but it does not say why the generator refused, and several of these generators fail on
code that compiles and passes every test.

This page is the list of things that have actually broken, and how to check each one before
pushing. It complements the [GraphQL Developer Guide](https://github.com/emoss08/trenova-documentation/blob/main/docs/engineering/graphql-developer-guide.md), which
covers how to build a resource; this covers what breaks afterwards.

## Run the whole gate locally

`Codegen Checks` in `.github/workflows/test-tms.yml` is six independent regeneration steps, plus the GraphQL schema-diff step. All paths
are relative to `services/tms`:

```bash
go run ./internal/api/graphql/gqlexec/gen                           # same as `task gqlgen`; see "GraphQL server code" below
(cd internal/api/graphql/gqlexec/internal/e2e && go run github.com/emoss08/trenova/internal/api/graphql/gqlexec/gen)
git status --porcelain -- internal/api/graphql/generated internal/api/graphql/gqlmodel internal/api/graphql/resolver internal/api/graphql/gqlexec/internal/e2e

go generate ./internal/infrastructure/database/seeder/...          # pkg/seedhelpers/seed_ids_gen.go
go generate ./internal/api/graphql/projection/...                  # internal/api/graphql/projection/specs_gen.go
go generate ./internal/infrastructure/database/reportcatalog/...   # pkg/reportcatalog/catalog_gen.go
go generate ./internal/core/services/agenttoolpolicy/safetydoc/...  # ../../docs/engineering/ai-tool-safety.md
go generate ./internal/api/writecoverage/...                       # ../../docs/engineering/agent-write-coverage.md
```

The REST API reference (OpenAPI) is no longer generated in this repository; it is produced
outside it, so there is no spec to regenerate here.

A clean `git status` after all six means the job will pass.

## GraphQL projections: the one that fails on correct code

`internal/api/graphql/projection/projection.yml` maps every GraphQL field to the database
column it reads. The generator walks **every field of every type** and aborts if a field
resolves to none of: a `buncolgen` column, an inferred relation, or a declared override. A
field that a resolver computes is not special-cased — it has to be declared.

This bites hardest when a column is **removed**. Retiring a database column while keeping
the GraphQL field (deprecated, resolver returning a constant) is the correct non-breaking
path, and it breaks the generator, because the field no longer has a column behind it:

```
Error generating GraphQL projections: type "CustomerBillingProfile" field
"consolidationLookbackDays" is neither a buncolgen column, inferred relation, nor
configured projection override
```

The four override kinds:

| Key | Use when |
| --- | --- |
| `aliases` | The field reads a differently-named column (`totalAmount` → `totalAmountMinor`). |
| `virtuals` | No column at all: a resolver computes it, or it is a deprecated field kept as a constant. |
| `specials` | One field needs several columns (`lastKnownLocation` needs the id and name too). |
| `modelOverrides` | The GraphQL type maps to a different Go type than the name implies. |

So: **removing a column means editing `projection.yml`, not only the migration.** Adding a
resolver-computed field means adding it to `virtuals` in the same commit.

## AI tool safety document

`docs/engineering/ai-tool-safety.md` is the auditable list of what every agent tool may do
without a person: its class, the most it can run at, any condition on the call, whether it
reads text written outside the organization, and its rationale. It is written from the
`Policy()` each tool declares, by building every registered tool with zero-valued
dependencies (`agenttoolpolicy/registered`, the same construction the contract tests use)
and rendering the catalog (`agenttoolpolicy/safetydoc`).

So **changing a tool's policy, adding a tool or removing one means regenerating the
document**:

```bash
cd services/tms
go generate ./internal/core/services/agenttoolpolicy/safetydoc/...
git diff --exit-code -- ../../docs/engineering/ai-tool-safety.md
```

The `AI tool safety document` step runs exactly that and fails on any difference, and
`TestCommittedDocumentIsCurrent` in the `safetydoc` package says the same under `task test`.
Never edit the file by hand: the next generate overwrites it and CI rejects the hand edit.
A tool constructor that dereferences a dependency while building fails the generator the
same way it fails the contract tests; keep constructors to storing what they are given.

## Agent write coverage

`docs/engineering/agent-write-coverage.md` is the ledger of every write a person can make
(each GraphQL mutation and each POST, PUT, PATCH or DELETE route) and the agent tool that
performs it or the reason none should. It is written from the schema, the gin route table
(`api.RouteTable` registers every handler with zero-valued dependencies), the registered agent
tools and the decisions in `internal/api/writecoverage/writecoverage.yml`.

So **adding a mutation or a write route, renaming one, renaming a tool, or editing
`writecoverage.yml` means regenerating the document**, and a new write needs an entry first:
`tools: [...]`, `exempt: <category>` with a `reason:`, or `pending: <what the tool would do>`.

```bash
cd services/tms
go generate ./internal/api/writecoverage/...   # task generate-write-coverage
```

The generator refuses, naming the key and the file, when a write has no entry, an entry names
a write or a tool that no longer exists, or an exemption has no reason. The `Agent write
coverage` step runs it with `-check` (`task generate-write-coverage-check`), which also fails
when the document differs, and `TestCoverageIsCurrent` in the `writecoverage` package says the
same under `task test`. Pending writes are the backlog: they are counted, never failed. A REST
route that reaches exactly the service calls a mutation reaches is merged into it and needs no
entry of its own; the failure says so when an entry for such a route is left behind.

## Agent tool schema contract

`internal/api/toolcontract` holds each agent tool that performs a GraphQL write to the
mutation's input type, read from `internal/api/graphql/schema/*.graphqls` with gqlparser.
Nothing is generated; the check is two Go tests that run under `task test`:

```bash
cd services/tms
go test ./internal/api/toolcontract/
```

`TestEveryBoundToolFitsItsGraphQLInput` fails when a bound tool stops requiring a non-null
input field, takes a parameter the input has no field for (a made-up `moves[].type`), or
offers enum values other than the input's. `TestEveryWriteToolWithAGraphQLInputIsBound` fails
when a create or update tool that `writecoverage.yml` lists against a mutation with an input
object is neither bound nor listed in `Unbound` with a reason. So **adding a create or update
tool, changing a bound tool's schema, or changing a bound input type means updating
`bindings.go`**: a new parameter the input lacks takes an `Extra` reason, a required field
the tool leaves out a `Defaulted` one, a narrower enum a `Narrowed` one. See "Schema
contract" in [agent-write-coverage.md](agent-write-coverage.md).

## Agent prompt and tool snapshots

The `Deterministic` job of `.github/workflows/agent-evals.yml` compares what a model is shown
against checked-in fixtures, so **changing a prompt, a starter template, a tool's description,
schema or policy, or adding or removing a tool means refreshing them**:

```bash
cd services/tms
go test -run TestPromptSnapshots ./internal/core/domain/agentdefinition/ -update
go test -run TestToolCatalogSnapshot ./internal/core/services/agentevalgate/ -update
```

A tool description, a product guide page or an eval request is also embedded, by content
hash, in `agentevalgate/evals/embeddings/nomic-embed-text.json`. Once that fixture is
recorded, editing any of them makes it stale, and the hybrid ranking gate names the one-line
command that re-records it from a local Ollama (see "Ranking" in
[ai-retrieval.md](https://github.com/emoss08/trenova-documentation/blob/main/docs/engineering/ai-retrieval.md)).

A tool description edit can also move `find_tools` ranking below its floors
(`agentevalgate/evals/toolselection.floors.json`); the failure lists the requests that no
longer find their tool. Fix the description rather than the floor. The flag goes after the
package. The job also runs the prompt-injection red-team suite; see
[agent-evals.md](https://github.com/emoss08/trenova-documentation/blob/main/docs/engineering/agent-evals.md) for what it proves and its known gaps.

## GraphQL server code

- `task gqlgen` runs `internal/api/graphql/gqlexec/gen`, not gqlgen's CLI. It uses gqlgen's
  binder and model/resolver plugins, then writes the executor as one package per schema file
  under `internal/api/graphql/generated/exec/`. Never run `go tool gqlgen generate`: it writes
  gqlgen's single-package executor next to the sharded one and the build fails with duplicate
  declarations. See [GraphQL Executor](graphql-executor.md).
- It runs in one pass. The old retry for a model gqlgen had just written is gone, because the
  generator reloads packages after the model plugin runs. If generation fails it puts the
  previous executor, `models_gen.go` and resolver files back, so a failed run leaves a
  buildable tree.
- A new schema file produces a new **untracked** executor package. `git diff` cannot see
  it. Check with `git status --porcelain`, which is what CI does — a local `git diff` will
  tell you everything is fine when it is not.
- The generator refuses gqlgen features the executor does not implement (subscriptions,
  runtime directives, batch resolvers, complexity functions) and names the field. Adding one
  means extending `gqlexec`, not working around the error.
- Field naming comes from `go_initialisms` in `gqlgen.yml`. Keep entries to three letters or
  more: two-letter ones (`AR`, `PO`, `MC`, `CC`) prefix-match SCREAMING enum values and
  mangle the generated constants (`DETENTION_POLICY` became `DetentionPOLicy`). Those few
  fields keep explicit `fieldName:` overrides instead.

## Schema changes

`task gqlschema-diff` (and the CI step) fails on breaking changes. Swapping a field between
wire-compatible scalars (`Int` ↔ `Timestamp`, `String` ↔ `Decimal`) is reported as dangerous,
not breaking. A deliberate redesign ships with the regenerated client in the same PR and
carries the `graphql-breaking-approved` label, which keeps the annotations but stops the
failure.

Prefer deprecating over deleting. A field marked `@deprecated` with a resolver returning a
constant is not breaking — just remember the `virtuals` entry above.

## Authorization

Every root resolver must reach a permission check. This is enforced by a **Go test**
(`TestEveryRootResolverIsAuthorized` in `internal/api/graphql/authzlint`), not by a lint
step, so it surfaces under `task test` rather than `task lint`. Self-scoped operations are
named `My<Thing>`.

## Columns

`task generate-columns` regenerates `pkg/buncolgen` from the domain entities. Adding or
removing a field on a Bun model without rerunning it leaves repositories referencing a
column helper that no longer matches the table — and, as above, can break the projection
generator too. Never hand-write column references; see [docs/bun/buncolgen.md](../bun/buncolgen.md).

## Client checks are narrower than they look

- `pnpm lint` runs `turbo run lint`, and **only `apps/web` and `apps/dash` define a `lint`
  script**. `packages/shared` is not linted at all, so an error there will not fail CI and
  will not show up in a repo-wide `oxlint` run you might do by hand.
- `pnpm fmt:check` is not wired into any workflow. Formatting is convention, not a gate.
- There is no `test` script in `packages/graphql`; its safety net is `tsc -b` plus the
  codegen check.
- **Never run the formatter over a `generated/` directory.** `oxfmt src` will happily
  reformat `src/types/generated/error-enums.ts` and `src/i18n/generated/locales.ts`, and the
  codegen check compares against what the generator emits, not against what is formatted —
  so a purely cosmetic reflow (an array collapsed onto one line) fails CI. Point the
  formatter at the directories you changed, or regenerate afterwards.

## Product guide

`services/tms/pkg/productguide/catalog_gen.json` is built from the web app's router,
navigation, page headers and record-link registry plus `docs/product-guide/**.md`, and the
server embeds it. `pnpm --filter @trenova/web guide:check` (the GraphQL Codegen job in
`test-client.yml`) fails when:

- a routed page has no guide — adding a page means writing its guide;
- a guide bolds a label the app does not show. **Renaming a button or retitling a page
  breaks every guide that names it**, and the message says which file. Fix the guide, not
  the check: the bold is the promise that the words are on screen. A label the app builds
  at runtime ("Publish {n} rows") or passes around untranslated is described in plain words;
- the catalog differs from what the generator writes. Run
  `pnpm --filter @trenova/web guide:generate` and commit the catalog with the change.

The check also runs when only `i18n/messages.en.json` changes, because the label check reads
it. See [product-guide.md](product-guide.md).

## i18n

- **The English source string is the catalog key.** Editing English text creates a new key
  and orphans the old one. Run `task i18n` (extract, sync and emit) and `task i18n-check`
  (the gate) after any change to user-facing text, and commit the runtime catalogs it
  rewrites. `check` fails when a file under `client/packages/shared/src/i18n/catalogs/` or
  `shared/i18n/catalogs/` differs from what `emit` would write, or when one is left over.
- **Client catalogs are split by bundle** (`core`, `web`, `dash`, one per web route folder)
  and the web app loads a route folder's bundle with its code. Moving a component between
  route folders, or into `apps/web/src/components`, moves its strings to a different bundle,
  so it is a `task i18n` change like editing text. See "How the browser loads catalogs" in
  [i18n/README.md](../../i18n/README.md).
- The codemod wraps a string literal only when its object key is in `TEXT_PROPS`
  (`i18n/tools/filter.mjs`) and only **inside a function**. A module-level
  `const NAV = [{ label: "Shipments" }]` evaluates once at import, so wrapping there would
  freeze the caption in whatever language was active then; those maps stay literal and their
  consumers translate at render (`t(item.label)`).
- A column builder is not a component, so the codemod reaches for the module-level
  `translate()`. If its call site memoizes on `[]`, the headers render correctly once and
  then never change language. Column builders take `t: TranslateFn` and their call sites
  memoize on `[t]` — `useT()` returns a new identity per locale precisely so that rebuilds.
- The message formatter is a deliberately small ICU subset, mirrored in
  `shared/i18n/format.go` and `client/packages/shared/src/i18n/format-message.ts`. The two
  must agree exactly, since the same catalog entry is rendered by both. It supports
  positional placeholders and plurals, and **does not resolve a placeholder nested inside a
  plural branch** — keep every `{n}` outside the branches.
- Never fold English grammar into arguments (`"{0} shipment{1}"` with the caller passing
  `""` or `"s"`, `t("{0} {1}", n, n === 1 ? "stop" : "stops")`, `pluralize("notice", n)`).
  Spanish inflects the noun differently and Chinese does not inflect at all. Use a plural:
  `{0, plural, one {# shipment} other {# shipments}}`. When the count is formatted
  (`toLocaleString`), keep it outside the plural and pluralize only the noun:
  `"{0} settled {1, plural, one {stop} other {stops}}"` with the formatted and raw counts.
  A word chosen by a condition (`"ends"` / `"ended"`, `"inbound"` / `"outbound"`) is one
  whole message per case, never an argument.
- Never split a sentence around markup (`{t("You've used")} <b>{pct}</b> {t("of …")}`): each
  half becomes its own entry and no language can move the bold part. Write it whole with
  `useRichT()` from `@trenova/shared/i18n/rich`:
  `rt("You've used <b>{0}</b> of this month's AI allowance", { b: (c) => <b>{c}</b> }, pct)`.
  Tags hold text and placeholders but no other tag, a plural cannot span a tag, and an
  argument is never read as markup.
  Outside a component, `translateRich` (same module) takes the same arguments.
- Never build English in a template literal where a person reads it
  (`` toast.success(`${name} saved`) ``, `` label={`${kind} Postal Code`} ``,
  `` title: `Journal Entry ${n}` ``): the extractor only collects string literals passed to
  `t`, `translate`, `rt` and `translateRich`, so the sentence never reaches a catalog and
  renders in English in every language. Write `t("{0} saved", name)`. Text placed into raw
  HTML (a print document) goes through the file's escaper after translation, since a
  translation is text like any other. A template that genuinely is not interface text (a
  value stored on a record, a log line for engineers) carries a
  `// i18n-ignore: <reason>` comment on or above it.
- A module-level label map (`STATUS_LABELS = { InReview: "In review" }`) never reaches a
  catalog, and a module-level `translate()` would freeze the language active at import.
  Declare it with `defineLabels({ … })` from `@trenova/shared/i18n/labels`: the extractor
  collects its values and every read (`STATUS_LABELS[status]`, `Object.entries(…)`) returns
  the current language, so a render needs no `t()`. Data built once at import — a select's
  options array — takes `sourceLabels(MAP)[key]`, the English source, and its renderer
  translates it, the same contract as a `label: "…"` literal. A record of copy with other
  fields beside the text (an icon, a flag) nests a `defineLabels` for its text, or uses
  getters that call `translate()`. A caption computed at runtime from a key or an enum value
  (`humanizeKey`) goes through `translateLabel()`, which reads the catalog and leaves an
  unknown caption in English.
- Validation messages are read when the form validates, not when the module loads: a zod
  check takes `{ error: () => translate("Name is required") }`, `ctx.addIssue` takes
  `message: translate(…)`, and a react-hook-form rule takes `required: translate(…)`. A toast
  is shown exactly as written, so every literal it receives — including `error.message ||
  "…"` fallbacks and `toast.promise` options — goes through `t()` or `translate()`.
- `task i18n-check` fails on all of these shapes (`i18n/tools/fragments.mjs`: split
  sentences, English arguments, templates where a person reads them — toasts, JSX children
  and attributes, `label`/`title`/`description` keys — module-level label maps, literal
  validation messages and literal toast text), and
  `i18n.mjs merge` refuses a translation whose placeholders, plurals or tags differ from
  its source (`i18n/tools/validate.mjs`).
- `i18n/locales.json` is the only place a language is added; `task i18n` regenerates the
  locale module from it.

## Local environment traps

These look like real failures and are not:

- **`internal/infrastructure/minio` tests fail** with `rootless Docker not found, failed to
  create Docker provider`. They are testcontainers tests and need Docker.
- **`turbo run …` fails** with `Exec format error (os error 8)` in some containers. Invoke
  the package's own binary instead (`cd apps/web && ./node_modules/.bin/tsc -b`).
- **`golangci-lint` refuses to run** when it was built with an older Go than the module
  targets. That needs CI, not a workaround.
- **`node --test i18n/tools/`** resolves a bare directory as a module on Node 22 and fails
  before running anything. Pass a glob.
