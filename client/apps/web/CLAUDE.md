## Effect

Effect v4 (beta). Your training data is predominantly v3 and will be wrong.

- All @effect/* packages share the `effect` version number. Never mix.
- `effect/unstable/*` modules may break in minor releases. Do not import from
  them without asking.
- Effect is confined to the API client and SSE client. Components, hooks, and
  form code stay Effect-free. Terminate every program with Effect.runPromise at
  the TanStack Query boundary.
- Forms use zod, not Schema. Do not unify these.

## Tests

Every behavior change ships with a test, and the test must be DEMONSTRATED RED
against the pre-change code: revert or stash the source change, run the test, show
it fail, restore. A test written after a fix, that has only ever run against the
fixed code, proves nothing. This codebase has already shipped a green suite that
pinned a bug as its specification — the GraphQL transport discarded populated
response documents while every fixture asserted a throw, because the fixtures were
written from the implementation instead of the contract.

Write fixtures from the contract: the GraphQL spec, the Go source, the schema.
Before writing a fixture, ask what the contract permits that no existing fixture
constructs — absent vs null vs empty, N>1 where every fixture uses one element,
fields whose values disagree with each other.

Do not propose manual verification as a substitute for a test. If something is
genuinely untestable, say so explicitly and say why.

## Formatting

Never run a formatter (oxfmt or otherwise) over a whole directory. Format only the
files you edited in the current task, by explicit path.

A directory-wide run produces a formatting-only diff across dozens of untouched
files. That is unreviewable, it buries the real change, and it collides with
concurrent work — a formatting-only edit cannot be cleanly separated from someone
else's in-flight change to the same file.

## setPartialErrorReporter

Telemetry sink ONLY. It is a process-global mutable reporter with no correlation
back to the originating request, operation, panel, or tenant context. Multiple
concurrent queries fire into it indistinguishably.

- Permitted: logging, metrics, tracing.
- Forbidden: any UI rendering or state decision derived from data that arrived
  via the reporter. No toasts, no field-level warnings, no conditional rendering.
- If partial errors must drive UI, the reporter is the wrong mechanism — that
  requires changing requestGraphQL's return type, which propagates to 337 call
  sites. Ask before going down that path.

## GraphQL transport and data tables

- Generated documents are hash-only (`{ __meta__: { hash, kind, name } }`); there is no
  SDL at runtime. Tests that need the operation text read it back from
  `@trenova/graphql/generated/persisted-documents.json` by hash (see
  `hooks/data-table/__tests__/use-data-table-query.test.ts`).
- Every GraphQL query wrapper takes a trailing `{ signal }` option and every `queryFn`
  forwards TanStack's `signal`; a new wrapper or query must do the same.
- Data-table configs are inferred from the document: `defineDataTableGraphQLConfig({
  document, operationName, connectionKey })` derives the row type from the unmasked
  connection node. Do not pass a hand-written REST row type unless the fragment
  structurally satisfies it; add the field to the fragment instead.
- Table queries request `totalCount` only on the first page (`includeTotalCount`); the
  table remembers the total per cursor scope.
- Routes warm the cache through `lib/route-prefetch.ts` (`lazyPrefetch` +
  `createPrefetchLoader`); a page module exports `prefetch` returning the query
  options it will read on mount, using the same key factories as the components.
- After editing a `.graphql` operation run `pnpm --filter @trenova/graphql codegen`
  (or keep `pnpm dev` running, which watches); the server reloads the safelist on an
  unknown hash outside production.
