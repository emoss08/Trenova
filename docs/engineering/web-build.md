# Web build and bundle

How `@trenova/web` is compiled and split, what a first visit downloads, and the traps that
have already been hit. Read this before changing `client/apps/web/vite.config.ts`,
`client/apps/web/vite/chunk-groups.ts`, `client/apps/web/bundle-budget.json`, or the GraphQL
codegen hooks in `client/packages/graphql/scripts/`.

## Compiling

- **React Compiler runs in Rust.** `react({ compiler: true })` uses `oxc-transform-react`, the
  compiler's Rust port, in place of `babel-plugin-react-compiler`. Babel held the main thread for
  nine-tenths of a six-minute build; the build now takes about half a minute. It also compiles
  what Babel 8 with compiler 1.0 silently skipped: every component that destructures a prop with
  a default (`{ icons = [] }`). `"use no memo"` is honoured as before.
- `oxc-transform-react` is a 0.x package, so `@vitejs/plugin-react`'s peer range pins its minor
  (`^0.152.0` means 0.152.x). Upgrade the two together.
- `pnpm build` bundles and does not typecheck; `pnpm typecheck` is its own task and CI job.
  Every tsconfig is `incremental`, so a repeat `tsc -b` checks only what changed (a touched but
  unchanged file costs nothing), and CI restores the build info between runs.

## What a first visit downloads

`index.html` loads the entry, preloads the modules it imports statically, and links one
stylesheet. `pnpm --filter @trenova/web bundle:check` measures that set against
`bundle-budget.json` and runs in CI after the build. When it fails, move what the first page
does not need behind `lazyComponent` (`src/lib/lazy-component.ts`), or raise the budget in the
same change if the growth is intended.

Things that were on every page and are not any more, and how they were kept off:

- The Desk and assistant conversation (with markdown, syntax highlighting and chrono-node): the assistant
  panel and the command palette's answer card are `lazyComponent`s, preloaded on intent.
- The page access panel (about 0.3 kB gzip of startup): the shell holds a
  keybinds-config entry bound in `SidebarLayout`, and a `lazyComponent` of `page-access-sheet.tsx`
  mounted once the app dialogs store marks the dialog `prepared` (the keybind pressed).
  The sheet lazy-loads the panel in turn, so the shell's import names one chunk to preload rather
  than the panel's whole graph; the sheet draws the panel's skeleton meanwhile.
- The Desk's 400 kB of CSS: the Desk loading screen, which the router imports for every page,
  imports `components/desk-chat/desk-frame.css`, the few rules it draws with, not the Desk's
  stylesheets.
- The password and role gates with `react-hook-form`: `routes/session-gates.tsx`, loaded when
  a session meets one.
- GraphQL fragment text and unused documents: see below.
- The central `queries` object (`lib/queries/index.ts`), which merged every factory and with
  them every service, type and GraphQL document. It is gone: every module imports the factory
  it reads from that factory's own module (`import { pageFavorite } from
  "@/lib/queries/page-favorite"`), aliased with a `Queries` suffix only where a local of the
  same name would shadow it (`shipment as shipmentQueries`). The keys are unchanged, since each
  factory builds its keys from its own scope. Dropping it from the shell, palette and Home cut
  the startup download from 610 to 560 kB gzip; dropping it everywhere cut what a route page
  downloads beyond startup from a median of 147 to 109 kB gzip (276 of 433 routes smaller). An
  import of `@/lib/queries` fails lint (`no-restricted-imports`). A test that stubs a factory
  mocks that factory's module, keeping the module's other exports with `importOriginal` when
  the code under test reads them; `test/data-table-queries.ts` holds the factories a
  `DataTable` reads, one entry per module. `hooks/__tests__/realtime-query-key-roots.test.ts`
  finds every factory with `import.meta.glob` over `lib/queries/*.ts` and the shared package's,
  so a new factory module is covered without being listed.

`main.tsx` imports `@trenova/shared/styles/app.css` before anything else. A cascade layer is
ranked by where its name first appears, and component stylesheets carry `@layer base` rules
of their own; imported after the router, they put `base` below Tailwind's `theme`.

## PDFs

Every PDF is shown by the browser's own viewer through `PdfFrame`
(`src/components/elements/pdf-frame.tsx`), which brings paging, zoom, search and printing for
free. There is no PDF rendering library in the app: pdf.js and react-pdf were two packages and
hundreds of kilobytes on four screens, and loaded their worker from unpkg.com, which the
Content Security Policy never allowed.

- **Fetched, then framed from a blob.** The API sends `X-Frame-Options: DENY`, so a document
  URL cannot be framed. `usePdfDocument` fetches the file through `api.download` (the session's
  credentials, CSRF and error handling) and `useObjectUrl` turns it into a `blob:` URL.
- **Always typed as a PDF.** A blob URL is the app's own origin. The blob is re-typed
  `application/pdf` whatever the response said, so a file mislabelled as HTML only ever reaches
  the PDF viewer and never renders as a page of the app.
- **What it asks of the viewer** goes in the URL fragment (`pdfViewerFragment`: `#page=`,
  `#zoom=`, `#view=Fit`, `#toolbar=0`). Chrome, Edge and Firefox honour all of them; Safari the
  page. Rotation (intake's preview of how a page will be filed) turns the frame, not the file.
- **No built-in viewer**, usually a phone (`navigator.pdfViewerEnabled === false`): the frame
  offers the file to open in a tab instead.
- **The CSP must keep `frame-src 'self' blob:`**: in `deploy/Caddyfile`, and in Cloud's worker
  (`trenova-cloud-deploy`, `edition/cloudflare/web/worker.ts`). Without it every preview is blank.
- pdf-lib still builds Capture's printable cover sheets. It is imported only when someone prints
  them (`cover-sheet-dialog.tsx`), and the sheet opens in the browser's viewer to print.

## Returning visits

Four pieces make a returning visit open fast, move every open tab onto a new release, and do
neither at the cost of anyone's work or data. All are production-only; the dev server runs
without them.

**Service worker** (`src/service-worker/sw.ts`, built into `/sw.js` by
`vite/service-worker.ts`). `routeRequest` (`src/service-worker/routing.ts`) decides what it
answers: content-hashed files under `assets/` from a cache, and a navigation to a page of the
app with the cached HTML shell. Everything else goes to the network untouched — `/api`,
`/graphql`, event streams, `/dash/` (Dash has its own worker), `/.well-known/`, other origins,
fonts (not content-hashed) and any path with a file extension. It never stores anyone's data,
and it never depends on its caches: a cache that fails sends the request to the network.

Each build ships a different `sw.js` whose manifest (`src/service-worker/manifest.ts`) names
the build's entry script and files. The worker installs only when the server's `index.html`
loads that entry and every startup file comes back as what it claims to be, so a CDN still
serving the previous build mid-release cannot pin the old shell for the whole release. It
never caches HTML as a build file: Cloudflare's static assets answer a missing file with
`index.html` and a 200 (`isCacheableAsset`).

**Moving onto a release** (`src/lib/service-worker.ts`). An open tab asks for a new release
every 30 minutes and when it comes back into view (at most once a minute). When a new worker
takes over the page, the page's next change of page is a full page load of where the person
was going; a filter, sort or table-page change never reloads. It waits while
`reloadIsSafe()` (`src/lib/reload-guards.ts`) says no: anything holding work a reload would
lose registers a guard — `UnsavedChangesGuard` while a form has unsaved edits, the popout
manager while a popout is open (unloading closes them). Code that reloads on its own
initiative must ask the guards; a new kind of in-memory work must add one.

**Release recovery** (`src/lib/chunk-recovery.ts`). A tab still on the old build that asks for
a file the release removed fails as it would anyway (Vite's `vite:preloadError`). Recovery then
asks the server which build it deploys (`deployedBuildId`, the entry in a fresh `index.html`)
and reloads only if that differs from the running build, the guards allow it, and it has not
already reloaded for this build. Offline, a server error or the same build still deployed are
not releases and never reload. Before reloading it lets the release's worker take over
(`settleOnNewestRelease`), so the reload opens the new build rather than the cached old one.

**Persisted query cache** (`src/lib/session-cache.ts`, `src/components/session-query-provider.tsx`).
The shell's queries — sidebar preferences, favorites, the person's organizations, attention
counts, the home layout — are kept in IndexedDB so the frame paints before the network
answers. Only queries on the `SHELL_QUERIES` allow-list are written; a business record or
anyone's personal details never is, because a shared dispatch-office PC keeps whatever is on
disk until someone signs out. Add a query only if a stranger at that PC could read it without
harm. A cache belongs to one person in one organization: `SessionQueryProvider` restores and
saves per scope (TanStack's `PersistQueryClientProvider` restores and subscribes once per page
and would keep saving into the first person's cache after a sign-in as someone else), and
every disk read and write checks the scope signed in at that moment, so a write still in
flight when a session ends is dropped. Caches are discarded by a new build and after a day,
and shell queries stay in memory as long (`retainShellQueries`). Every way a session ends or
changes hands — signing out, switching organization, signing in, an expired session — goes
through `clearSessionCache()`.

**Operating it.** `/sw.js` must be revalidated on every check: Caddy sends `no-cache` for
everything outside `assets/`, and Cloudflare's static assets send
`public, max-age=0, must-revalidate`. The CSP must allow `worker-src 'self'`. To withdraw the
worker, build with `VITE_SERVICE_WORKER=off`: the app registers nothing, and the `sw.js` it
ships deletes the worker's caches, unregisters it and reloads the pages it controlled, in every
browser that installed it. A path on the app's origin that the server answers itself (not the
SPA) must be added to `routeRequest`'s exclusions, or the worker answers it with the shell.

## Chunk groups

`vite/chunk-groups.ts` holds Rolldown's `codeSplitting.groups`; its tests are in
`src/lib/__tests__/chunk-groups.test.ts`. The rules, each learned from a broken build:

- **Match a package by its own `node_modules/<name>/` segment** (`packageTest`). pnpm names a
  package's folder after its peers (`@base-ui+react@1.8.0_date-fns@4.4.0`), so `/date-fns/`
  put all of Base UI in the date chunk, and `/fetch/` caught `route-prefetch.ts`.
- **Group only** the framework core every page needs, or a large library that only loads
  behind a lazy boundary. A library used a little on many pages (icons, date-fns, Base UI)
  is left ungrouped, so each route gets only what it uses.
- **Startup groups** (`tags: ["$initial"]`) gather what the entry imports statically into one
  chunk per source — dependencies, the shared package, the app — and all startup CSS into
  one file. Without them it was a couple of hundred small preloaded chunks.
- **Never give a startup group a `maxSize`.** Cut by size, the startup chunks evaluated out of
  order and the app failed to start with "s is not a function".
- **`$initial` follows re-exports.** One icon in the shell made the whole icon set "initial".
  Modules reached through a barrel are listed in `REACHED_THROUGH_BARRELS` and left to
  automatic chunking.
- A group pulls in its modules' dependencies (`includeDependenciesRecursively`), whatever its
  `test` says about them. Excluding a module from a group does not keep it out if a captured
  module imports it.

## GraphQL codegen hooks

After `graphql-codegen` writes `src/generated/`, two hooks rewrite it (both have `node:test`
suites run by `pnpm test`):

- `strip-fragment-text.mjs` replaces each fragment's GraphQL text with its name. The server
  only accepts persisted operations by hash, and nothing at runtime reads a fragment's text.
- `split-documents.mjs` moves the documents out of `graphql.ts` into one module per operations
  file (`src/generated/documents/<area>/<file>.ts`), re-exported from `graphql.ts`, so a route's
  documents travel with it. It reads definitions with the GraphQL parser and fails loudly if
  the generator's output changes shape.

The package declares `"sideEffects": false`. Import `getFragmentData` from
`@trenova/graphql/fragment-data`, never from the `@trenova/graphql/generated` barrel, which
re-exports `gql.ts` and its namespace import of every document.

## Full-bundle dev server

`pnpm --filter @trenova/web dev:bundled` (`vite --experimentalBundle`) serves the app bundled by
Rolldown, as a build would, instead of as thousands of separate modules: a page loads a few
large files rather than one request per module, and a component edit hot-swaps in about half a
second with the page's state kept. The first page load bundles the shell (about 15 seconds
cold); a route's chunk is bundled the first time something imports it. `pnpm dev` is unchanged
and stays the default. An edit to a module that also exports something other than components
(`sidebar-chrome.tsx` exports `formatAttentionCount`) reloads the page in both modes; that is
Fast Refresh's rule, not the bundler's.

The dev server bundles with `build.rolldownOptions`, so two of them are split by `command`:

- **`strictExecutionOrder` is on in serve.** Without it a module the entry shares with a lazy
  route is hoisted into a chunk the entry imports, and that chunk runs before the entry's own
  body. React's refresh preamble is inlined from `index.html` into the entry's body, so the
  components in the shared chunk checked for it first and threw "@vitejs/plugin-react can't
  detect preamble"; which chunk failed changed with every rebundle (`page-*`, `use-reports-*`,
  `route-command-data-*`). Strict order wraps each module so it runs in source order wherever
  it lands. It costs bytes, so production leaves it off; a production build has no preamble.
- **The chunk groups are build only.** The startup groups pull the shell's modules into chunks
  ahead of the entry, which only moves more components in front of the preamble.

## Evaluated and not adopted

- **`build.chunkImportMap`** stops one changed chunk from changing the hash of every chunk that
  imports it, so a returning user re-downloads only what changed. It inlines a
  `<script type="importmap">` in `index.html`, which a `script-src` without `'unsafe-inline'`
  blocks unless it lists the map's hash. Cloud's CSP (set by the `trenova` Worker) already lists
  `sha256-` hashes for inline scripts, so adopting it means adding the map's hash, which changes
  with every build, to that list at deploy; the self-hosted Caddy CSP would need the same.
- Rolldown `experimental.nativeMagicString` (slower here) and `tsc --checkers 8` (slower and
  9 GB, against 4 checkers' 6 GB).

## Measuring

- **A light helper in a component module brings the component.** `import { pageAssistantThreadKey }`
  from `page-assistant.tsx` put the whole Desk thread into the rate confirmation import's
  preview (306 → 584 kB gzip). Keys, constants and pure functions that pages share live in a
  module of their own (`page-assistant-thread-key.ts`, `capture-cover-sheet-print.ts`), never
  beside the component that uses them.
- To see what fills a chunk, build with a config that adds a `generateBundle` plugin writing
  each chunk's `modules` (`renderedLength` per module), `imports` and `facadeModuleId`; diff two
  builds' module sets for one route to name exactly what it gained.

- Rolldown prints `PLUGIN_TIMINGS` after a slow build: which plugin hook took the time.
  A hook another plugin wraps reports that plugin's time as its own.
- `pnpm --filter @trenova/web bundle:check` after a build for the startup set.
- For which module pulls what into startup, dump `this.getModuleInfo(id).importedIds` from a
  `generateBundle` hook and walk it from `src/main.tsx`.
