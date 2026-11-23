# Client commands

**The client uses Bun, not Node or pnpm.** On 2026-11-23 `client/` moved from pnpm and Node to
Bun, which installs the packages and runs every script, tool and test. Do not use `pnpm`, `npm`,
`npx` or `node` for anything in the client or in `i18n/tools`: each has a Bun command below.

The Bun version is the `packageManager` field in `client/package.json` (now `bun@1.4.5`). CI
(`.github/actions/setup-client`) reads that field; `deploy/Dockerfile.client` pins the same
version by `BUN_VERSION` and digest, so a Bun upgrade changes both. Install or upgrade Bun with
`curl -fsSL https://bun.sh/install | bash` or `bun upgrade`.

## Every day

From `client/`:

| What | Command |
|---|---|
| Install dependencies | `bun install` (CI and images: `bun install --frozen-lockfile`) |
| Run both apps | `bun run dev` |
| Run one app | `bun run --filter @trenova/web dev` (or `@trenova/dash`) |
| Build | `bun run build` |
| Typecheck | `bun run typecheck` |
| Lint | `bun run lint` |
| Design tokens | `bun run lint:design` |
| Loading skeletons | `bun run lint:skeletons` |
| Single copies of shared libraries | `bun run check:single-copies` |
| All unit tests | `bun run test` |
| Format files you changed | `bunx --bun oxfmt <file> <file>` (by path, never a folder) |

Any script in a workspace package runs the same way: `bun run --filter <package> <script>`.

## Tests

| What | Command |
|---|---|
| Unit tests in every package | `bun run test` (from `client/`) |
| Some web test files | `bunx --bun vitest run --project=unit <files>` (from `client/apps/web`) |
| Storybook tests | `bun run test:storybook` (from `client/apps/web`) |
| Playwright end-to-end | `bun run test:e2e` (from `client/apps/web`) |
| GraphQL script tests | `bun run --filter @trenova/graphql test` |
| i18n tool tests | `task i18n-test` (`bun test i18n/tools/`) |

## Generators and their checks

| What | Generate | Check (CI runs this) |
|---|---|---|
| GraphQL client code, after a `.graphql` change | `bun run --filter @trenova/graphql codegen` | `bun run --filter @trenova/graphql codegen:check` |
| Product guide, after a page or a guide-named label changes | `bun run --filter @trenova/web guide:generate` | `bun run --filter @trenova/web guide:check` |
| Page access screens, after a page's imports change | `bun run --filter @trenova/web access:generate`, then `go run ./internal/api/graphql/gqlext/refgen` from `services/tms` | `bun run --filter @trenova/web access:check` |
| Startup bundle budget, after a build | - | `bun run --filter @trenova/web bundle:check` |
| Icons | `bun run --filter @trenova/shared icons:generate` | - |
| Translations, after user-facing text changes | `task i18n` | `task i18n-check` |

## Mock API and screenshots

From `client/apps/web`:

```bash
bun e2e/mock-api/server.mjs                     # the mock API on :8080
bun run dev                                     # the web app against it
bun e2e/mock-api/notifications-screenshots.mjs  # any *-screenshots.mjs script
```

On spare ports: `MOCK_API_PORT=8090 MOCK_API_ORIGINS=http://localhost:5183
MOCK_API_PUBLIC_URL=http://localhost:8090 bun e2e/mock-api/server.mjs` and
`VITE_API_URL=http://localhost:8090/api/v1 bunx --bun vite --port 5183 --strictPort`.

## Dependencies

| What | Command |
|---|---|
| Add a dependency to a package | `bun add <name>` from that package's folder (`-d` for a dev dependency) |
| Remove one | `bun remove <name>` from that package's folder |
| A version every package shares | add it to `workspaces.catalog` in `client/package.json` and reference it as `"catalog:"` |
| Force a version everywhere | `overrides` in `client/package.json` |
| Patch a dependency | `bun patch <name>`, then `bun patch --commit <folder>` (see [web-build](web-build.md#patched-dependencies)) |
| A package whose install script must run | add it to `trustedDependencies` in `client/package.json`; `bun pm untrusted` lists any that were skipped |
| Run a package's CLI once | `bunx <name>` |

## From `services/tms/`

`task watch-ui` (dev server), `task ui-install`, `task ui-build`, `task ui-lint` and `task ui-test`
call the Bun commands above.

## Traps

- **`bun run test`, never `bun test`.** `bun test` is Bun's own test runner: it skips vitest and
  runs only the `node:test` suites (`scripts/*.test.mjs`, `i18n/tools/`). Its default timeout is
  5 seconds where `node:test` has none, so a test that scans the whole app sets `{ timeout }`.
- **Every workspace package has a `bunfig.toml` with `[run] bun = true`.** Bun reads
  `bunfig.toml` only from the folder a command starts in, and turbo starts each task in the
  package's folder. Without the file, `bun run build` starts vite on Node whenever Node is
  installed. A new workspace package gets the same file.
- **`bun run` drops a lone `--`.** Flags for a turbo-backed script (`dev`, `build`, `lint`,
  `typecheck`, `test`) go through turbo: `bun x turbo run lint -- --format=github`. A plain
  script takes them directly: `bun run lint:design --format=github`.
- **Run a package's binary with `bunx --bun <bin>` from the package's folder.** Without `--bun`,
  bunx starts it on Node. Bun has no `--filter … exec`.
- **Packages live in one store.** The isolated linker keeps every package at
  `client/node_modules/.bun/<name with / as +>@<version>` (plus `+<peer hash>` when it has peers)
  and links it into the `node_modules` of each workspace that declares it. A package imports only
  what its own `package.json` names. `scripts/check-single-copies.ts` reads `bun.lock` and that
  store.
- **New releases wait a day.** `minimumReleaseAge` in `client/bunfig.toml` holds back versions
  younger than a day. To take an urgent fix, name the package in `minimumReleaseAgeExcludes`, and
  drop it once the version is a day old.
- **The bundle budget is gzip as Bun measures it.** Bun's zlib writes output about 0.5% larger
  than Node's at the same level. The budget was rescaled once, on the same build, when the client
  moved to Bun.
