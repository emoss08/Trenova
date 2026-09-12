# Generated Artifacts and the Checks That Guard Them

Most of this repository's generated code is guarded by a CI job that regenerates it and
fails if the result differs from what you committed. The failure text names the command to
run, but it does not say why the generator refused, and several of these generators fail on
code that compiles and passes every test.

This page is the list of things that have actually broken, and how to check each one before
pushing. It complements the [GraphQL Developer Guide](graphql-developer-guide.md), which
covers how to build a resource; this covers what breaks afterwards.

## Run the whole gate locally

`Codegen Checks` in `.github/workflows/test-tms.yml` is five independent steps. All paths
are relative to `services/tms`:

```bash
go tool gqlgen generate || go tool gqlgen generate   # see "gqlgen" below for the retry
git status --porcelain -- internal/api/graphql/generated internal/api/graphql/gqlmodel internal/api/graphql/resolver

go generate ./internal/infrastructure/database/seeder/...          # pkg/seedhelpers/seed_ids_gen.go
go generate ./internal/api/graphql/projection/...                  # internal/api/graphql/projection/specs_gen.go
go generate ./internal/infrastructure/database/reportcatalog/...   # pkg/reportcatalog/catalog_gen.go

go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g ./cmd/cli/main.go -o ./docs --parseInternal --parseDependency
go run ./cmd/openapi-postprocess                                   # then: git diff --quiet -- docs
```

A clean `git status` after all five means the job will pass.

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

## gqlgen

- The model plugin can miss the `models_gen.go` it just wrote when a schema adds a model, so
  the first pass fails and the second succeeds. `task gqlgen` and CI both retry once. A
  single failure is not a real failure; two in a row is.
- `follow-schema` layout writes one file per schema file, so a new schema file produces a new
  **untracked** file. `git diff` cannot see it. Check with `git status --porcelain`, which is
  what CI does — a local `git diff` will tell you everything is fine when it is not.
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

## i18n

- **The English source string is the catalog key.** Editing English text creates a new key
  and orphans the old one. Run `task i18n` (extract + sync) and `task i18n-check` (the gate)
  after any change to user-facing text.
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
  `""` or `"s"`). Spanish inflects the noun differently and Chinese does not inflect at all.
  Use a plural: `{0, plural, one {# shipment} other {# shipments}}`.
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
