---
name: trenova-frontend
description: How to write frontend code in Trenova's client (client/apps/web, client/apps/dash, client/packages/*). Use before adding or changing any React component, page, route, hook, form, data table, query, GraphQL operation on the client, style, or user-facing text, and when reviewing such a change. Covers where code goes, the page and loading-state anatomy, data fetching and query keys, forms, i18n, styling tokens, bundle-size rules, security, tests, and what to verify.
---

# Writing frontend code in Trenova

The engineering docs are the source of truth; this skill is the map to them plus the rules
that have actually broken `master` or bloated the bundle. Read the doc for the area you touch
before writing code (table below), then follow the rules here.

**When a rule is wrong, change the rule.** If a rule in CLAUDE.md, a doc or a lint config
stops you from doing the better thing, update the rule in the same change and say so in your
report. Never work around a rule silently, and never add an `ignore` comment to get past lint
without a reason that a reviewer would accept.

## Read first

| You are touching | Read |
|---|---|
| Any styling, a new page, a layout, colours, icons, a loading state | `docs/engineering/design-system.md` |
| `vite.config.ts`, chunking, `main.tsx` imports, a heavy dependency, the service worker, the persisted cache, PDFs | `docs/engineering/web-build.md` |
| User-facing text | `docs/engineering/generated-artifacts.md` → "i18n", and `i18n/README.md` |
| A new page, a renamed button a guide names, a record link | `docs/engineering/product-guide.md` |
| A list keyed by kind of record | `docs/engineering/record-kinds.md` |
| A GraphQL schema change | `docs/engineering/generated-artifacts.md` |
| Desk artifacts or cards | `docs/engineering/desk-artifacts.md` |
| The formula editor | `docs/engineering/formula-language-service.md` |
| Realtime updates in the browser | `docs/engineering/realtime.md` |
| Anything in `apps/web` | `client/apps/web/CLAUDE.md` (Effect, tests, formatting, GraphQL transport) |

Copyable patterns for the common tasks are in [patterns.md](patterns.md).

## Where code goes

- `apps/web` is the TMS, `apps/dash` the driver portal. Anything both use (UI components,
  hooks, utilities, types, base services) lives in `packages/shared` and is imported as
  `@trenova/shared/...`. Never import one app from the other.
- A route's own components live in `src/routes/<folder>/_components/`. Code used by several
  route folders lives in `src/components/`, `src/hooks/` or `src/lib/`.
- Utilities go in `lib/` (`@trenova/shared/lib/*` when both apps use them), never inline in a
  component or hook. **Search before writing one**: `escapeRegExp` had been copied into five
  files. Existing helpers worth knowing: `useObjectUrl`, `useElementSize`/`useElementWidth`,
  `useDebounce`, `useSettledValue`, `useLatestCallback` (shared hooks); `usePermission`,
  `useApiMutation`, `useCopyToClipboard`, `useSignOut` (web hooks); `apiUrl`/`apiEndpoint`,
  `escapeRegExp`, `escapeHtml`, `cn`, `toSentenceFragment` (shared lib).
- **A module that pages share holds only what is light.** A key, a constant or a pure function
  imported from a component module brings the whole component and everything it imports.
  `pageAssistantThreadKey`, imported from `page-assistant.tsx`, once put the entire Desk chat
  into the rate confirmation preview. Put shared keys and helpers in a module of their own
  (`page-assistant-thread-key.ts`, `capture-cover-sheet-print.ts`).
- Effect is confined to the API client and SSE client. Components, hooks and forms stay
  Effect-free; forms use zod.

## Pages and loading states

- Every page is `PageLayout` (`components/navigation/sidebar-layout.tsx`) with
  `pageHeaderProps`. Never mount `PageHeader`, never draw a top bar, never `p-0` and re-add
  gutters. A full-height workspace takes `fill`.
- A list page is `PageLayout` > `DataTableLazyComponent` > the table, with no wrapper `div`.
  Create buttons read "New {thing}".
- Lazy-load everything below the page: `DataTableLazyComponent` only around a component that
  renders `DataTable`; everything else in `LazyComponent` (from
  `@trenova/shared/components/error-boundary`) with a required `fallback` that is that
  component's own `<name>-skeleton.tsx`. The skeleton is the real layout in `ui-shimmer`,
  with `aria-busy` and an `aria-label`, and the component's own data-loading state draws the
  same skeleton. Never a spinner for a page or section.
- Routes are registered in `apps/web/src/router.tsx` with a lazy import; a page that knows
  what it will read exports `prefetch` and uses `createPrefetchLoader`/`lazyPrefetch`
  (`lib/route-prefetch.ts`). A component that should start loading on hover is a
  `lazyComponent` (`lib/lazy-component.ts`) and calls its `preload` on intent.
- Record links come from `recordPath` (`config/record-links.ts`), never hand-built.
- Gate actions with `usePermission(Resource.X, Operation.Y)`; the server checks again, so
  this is for the interface, not for security.
- A new page, or a renamed button a product guide names, means
  `pnpm --filter @trenova/web guide:generate`.

## Data

- **Reads go through TanStack Query.** Query key factories are `createQueryKeys` modules in
  `apps/web/src/lib/queries/<name>.ts`; import the factory from its own module
  (`import { shipment } from "@/lib/queries/shipment"`). There is no merged `queries` object
  and lint rejects one. A new factory is one new module.
- Every `queryFn` forwards TanStack's `signal`, and every GraphQL or REST wrapper takes a
  trailing `{ signal }`.
- **GraphQL**: operations live in `packages/graphql/src/operations/<area>/*.graphql`; run
  `pnpm --filter @trenova/graphql codegen` (or keep `pnpm dev` running). Documents are
  hash-only at runtime. Wrappers in `apps/web/src/lib/graphql/<area>.ts` call
  `requestGraphQL`.
- **Data tables** take `defineDataTableGraphQLConfig({ document, operationName,
  connectionKey })` and infer their row type from the fragment; add a field to the fragment
  rather than hand-writing a row type. `totalCount` is requested on the first page only.
- **Writes** use `useApiMutation`, passing the form so server field errors land on fields;
  invalidate with the factory's keys. A screen that holds in-memory work a reload would lose
  registers a reload guard (`lib/reload-guards.ts`); forms get one from
  `UnsavedChangesGuard`.
- A file the API serves is fetched with `api.download(endpoint, …)` so it carries the
  session's credentials, CSRF and error handling. `apiEndpoint(path)` turns a server path into
  that endpoint.
- Nothing about a business record or a person is ever written to browser storage. The
  persisted shell cache is an allow-list (`SHELL_QUERIES`); every session end goes through
  `clearSessionCache()`.
- `setPartialErrorReporter` is telemetry only; never drive UI from it.

## Forms

React Hook Form with a zod schema from `@trenova/shared/types/<entity>` and `zodResolver`;
field components from `apps/web/src/components/fields/`; table panels through
`FormCreatePanel`/`FormEditPanel`. Footers are right-aligned, Cancel (`outline`) then the
primary. Dialogs take `size`, never an arbitrary max-width. Settings pages are stacked
`Card`s with a `FormSaveDock`.

A toast is for news the screen cannot show (design-system.md → "Toasts"). A save, delete,
toggle or approval the person can see happen takes no success toast, and `useApiMutation`
already reports a failure. Toast a failure with nowhere else to go, work that finished out
of sight, a bulk outcome, or a result that carries its next step (Undo, Download, Open).

## Text and i18n

- Every string a person reads goes through `t()` from `useT()` inside a component, or
  `translate()` outside one. The English string is the catalog key.
- Never build a sentence from pieces: no template literals (`` `${name} saved` `` is
  `t("{0} saved", name)`), no English grammar in arguments, no sentence split around markup
  (use `useRichT()`). Plurals are ICU: `t("{0, plural, one {# stop} other {# stops}}", n)`,
  with every `{n}` outside the branches.
- Module-level label maps use `defineLabels` (`@trenova/shared/i18n/labels`); column builders
  take `t` and memoize on `[t]`.
- New or changed text needs `task i18n`, translations for every locale in
  `i18n/locales.json`, and a passing `task i18n-check`. Moving a component to another route
  folder moves its strings to another catalog bundle, so that needs `task i18n` too.

## Styling

- Tokens only (`packages/shared/src/styles/tokens.css`): no palette classes (`bg-red-500`),
  no arbitrary font sizes, no hex, no shadows on anything that sits in the page, no
  hand-rolled focus rings (`ui-focus-ring`). Form controls spend `ui-field`. A missing colour
  is a new token in both `:root` and `.dark`, checked by `pnpm lint:design`.
- Tone for things with a severity order, categorical accent for categories; a status map
  declares a lifecycle phase and the tone follows.
- Sentence-case labels, no uppercase tracking. Weights mean something: 400 body, 500 label,
  600 heading; a value in a cell takes no weight class.
- Icons from `@trenova/shared/components/icons` only. Machine suggestions wear `AssistMark`.
- Figures are `KpiStrip`, titled blocks `SectionPanel`, read-only detail `DescriptionList`,
  callouts `<Alert size="sm">`.
- Code is shown with `HighlightedCode`, markdown with `AiMarkdown`, PDFs with `PdfFrame`.
  Don't add another renderer for any of them.

## Bundle discipline

The first visit's download is budgeted (`apps/web/bundle-budget.json`, checked in CI). The
rest of the app stays fast only if each page brings only what it uses.

- Nothing heavy is imported statically from the shell, the command palette or Home.
- A heavy library used by one action loads at that action: `const { build } = await
  import("@/lib/heavy-thing")` inside the handler (pdf-lib for cover sheets, see
  `cover-sheet-dialog.tsx`).
- Before adding a dependency, check its size and whether the platform already does the job
  (the browser's PDF viewer replaced pdf.js and react-pdf). Never add a second copy of a
  singleton package (`singleton-packages.ts`, `check:single-copies`).
- No barrels or index objects that merge many modules for convenience: they put everything
  into every importer.
- Chunk groups (`vite/chunk-groups.ts`) follow the rules in web-build.md; never give a startup
  group a `maxSize`.

## React and correctness

- The React Compiler compiles every component; don't scatter `useMemo`/`useCallback` for
  speed. A component that reads a mutable external object (TanStack Virtual, a store mutated
  in place) starts with `"use no memo"`. Tests whose behaviour depends on the compiler are
  named `*.compiled.test.tsx`.
- Side effects live in effects, not in render or `useMemo`. Object URLs come from
  `useObjectUrl`, which survives StrictMode's double mount; a `useMemo` that creates one and an
  effect that revokes it leaves a revoked URL in use.
- Don't name a local `document` or `window`.
- Sanitize nothing by hand: render text as elements. `dangerouslySetInnerHTML` needs a
  reviewed reason. A `blob:` URL runs as the app's origin, so type it explicitly (PDFs are
  re-typed `application/pdf`). External links open with `rel="noopener noreferrer"`.

## Tests

From `client/apps/web/CLAUDE.md`: every behaviour change ships with a test of what a caller
relies on, named for that behaviour, asserting exact output. Write fixtures from the contract
(the GraphQL schema, the Go handler), not from what the code currently returns. Mock a query
factory's own module (`vi.mock("@/lib/queries/<name>", …)`), keeping its other exports with
`importOriginal` when the code under test reads them. Mock the request helper
(`@trenova/shared/lib/api`) rather than a component's internals.

## Before you hand off

Format the files you edited, by path (`oxfmt <files>`), never a whole directory. Then list,
for the person running the PR, exactly which of these the change needs. Run them locally
only when asked:

- `pnpm typecheck` (or `pnpm --filter <package> typecheck`)
- `pnpm lint`, `pnpm lint:design`, `pnpm lint:skeletons`, `pnpm check:single-copies`
- the tests you added or touched (`pnpm --filter @trenova/web exec vitest run <files>`)
- `task i18n` + translations + `task i18n-check` (text changed)
- `pnpm --filter @trenova/graphql codegen` (an operation changed)
- `pnpm --filter @trenova/web guide:generate` (a page or a guide-named label changed)
- `pnpm --filter @trenova/web build` then `bundle:check` (imports, dependencies or chunking
  changed)

Report what you changed, any rule you changed and why, and the list above.
