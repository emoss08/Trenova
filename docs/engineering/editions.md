# Editions

The public repository is the self-hostable product. A hosted edition (Trenova Cloud)
adds UI of its own — self-serve signup, a trial banner, a plan page — from a private
repository. That code is **overlaid** into `client/packages/cloud/` at build time and
compiled into `@trenova/web` through one import. Without the overlay the web app builds,
typechecks, lints and tests exactly the same, and renders nothing extra.

Runtime behaviour still keys on the server: `GET /system/public-config` →
`platformMode` (`packages/shared/src/types/platform.ts`). An overlay must keep gating
its features on that, so a self-hosted install built with the overlay behaves as one
built without it.

## Pieces

| Piece | Where | Public? |
|---|---|---|
| Registry contract (`defineEdition`, types, `resolveEdition`) | `client/packages/edition/src` (`@trenova/edition`) | yes |
| Default no-op edition | `client/packages/edition/src/default-entry.ts` | yes |
| Entry resolution for Vite, Vitest, Storybook | `client/packages/edition/node/resolve-entry.ts` | yes |
| The one place the web app reads the edition | `client/apps/web/src/lib/edition.ts` | yes |
| The overlay | `client/packages/cloud` (`@trenova/cloud`) | moves to the private repo |

`@trenova/edition` is its own package rather than a module in `@trenova/shared` because
it is the API boundary the private overlay compiles against: small, web-app specific
(routes, admin links, layout slots — Dash has none of them) and versioned with the host.
`@trenova/shared` is consumed by both apps and is not a contract.

### How the entry resolves

The web app imports `@trenova/edition-entry`. Nothing depends on `@trenova/cloud` by
name; `scripts`/CI grep for that and it must stay so.

- TypeScript: `apps/web/tsconfig.app.json` maps the specifier to
  `../../packages/cloud/src/index.ts` **then** `../../packages/edition/src/default-entry.ts`;
  `paths` falls through to the second when the first file does not exist.
- Vite, Vitest and Storybook: `editionAlias()` from `packages/edition/node/resolve-entry.ts`
  makes the same choice with `existsSync`. Storybook also picks up the overlay's stories.
- Oxlint's type-aware rules use the same tsconfig.

## The registry

```ts
export default defineEdition({
  id: "cloud",
  name: "Trenova Cloud",
  routes: { public, guest, protected, admin },   // react-router RouteObjects
  adminLinks: [{ href, title, group, resource, requiredOperation, platformMode, after }],
  navItems: [{ moduleId, groupId?, after?, item: { id, label, path, icon?, resource?, platformMode? } }],
  slots: { AppBanner, RootHost, LoginPrompt },
  protectedLoaders: [loader],                     // after the session check; first non-null wins
  plan: { useRestrictions, loadRestrictions, restrictedRedirect, usagePath?, limitCopy? },
  messages: { es: () => import(...), "zh-TW": ..., "zh-CN": ... },
});
```

- **Routes** mount at `public` (no session, no shell — beside the tender offer pages),
  `guest` (beside `/login`; signed-in visitors go home), `protected` (inside the app
  shell, behind `protectedLoader`) or `admin` (children of `/admin`; relative paths).
  Put a `createPermissionLoader` on a guarded route as the host's own routes do.
- **Admin links and nav items** are merged into the host's lists by
  `config/app-navigation.ts`; `after` names the host entry to follow. `platformMode`
  hides an entry on any other install.
- **Slots**: `AppBanner` renders above the app header on every signed-in page,
  `RootHost` once at the router root, `LoginPrompt` the line under the sign-in heading
  (it receives the host's line as `fallback` and must render it when it has nothing to
  add).
- **Plan**: the host owns `PlanCapability`, the `planCapability` filters on navigation,
  `createPlanCapabilityLoader` and the plan-limit dialog for `QUOTA_EXCEEDED` /
  `PLAN_RESTRICTED` / 402 — all of which do nothing without a plan source. The edition
  supplies the restrictions (hook and loader), where a withheld route redirects, the
  plan page the dialog links to, and its own wording (`limitCopy`, usually
  `planLimitCopy(notice, t, wording)` from `@/lib/plan-limit-copy`).
- **Messages**: see below.

The onboarding wizard is public and is not an edition feature; its gate reads
`platformMode` and the onboarding API, both public.

## Translations

Strings that exist only in overlay code live in the overlay's own catalog,
`client/packages/cloud/i18n/messages.<locale>.json` (keyed by English, like the app's).
`task i18n` extracts the overlay when it is in the tree, before pruning the public
catalogs, so a string that moves into the overlay takes its translation with it; a
string the public app also uses stays in the public catalog. `task i18n-check` checks
both. At runtime the edition's `messages` loaders are layered over the app catalog with
`registerCatalogSource` (`packages/shared/src/i18n/runtime.ts`).

## The overlay contract

`client/packages/cloud` must:

1. Default-export a `defineEdition(...)` from `src/index.ts`.
2. Import the host only through `@/…` (the web app's `src`, which it is compiled into),
   `@trenova/shared/…` and `@trenova/edition`. Host modules are not a stable API; an
   overlay is built against the host commit it is overlaid on.
3. Add no external dependency the host does not already lock: its `package.json`
   lists `peerDependencies` and `devDependencies` with **the same specifiers** the web
   app or shared use, so installing it resolves nothing new.
4. Keep `typecheck` (its `tsconfig.json` extends `apps/web/tsconfig.app.json`), `lint`
   and `test` (its `vitest.config.ts` resolves like the web app's) scripts, which turbo
   runs with the rest.
5. Keep its product guides under `guide/` (the public generator reads only
   `docs/product-guide`).

### Building with the overlay

From a checkout of the public repository at the commit the overlay targets:

```bash
cp -R <private>/cloud client/packages/cloud      # the whole package directory
cd client
pnpm install --no-frozen-lockfile                 # adds only the packages/cloud importer
pnpm --filter @trenova/cloud typecheck && pnpm --filter @trenova/cloud test
pnpm --filter @trenova/web build                  # dist/ now contains the Cloud UI
```

`pnpm install --frozen-lockfile` cannot be used on the first install: pnpm requires
every workspace project to have an importer entry, and the public lockfile only keeps
the overlay's while the package is in the public tree. Because the overlay's
specifiers all match the host's, the non-frozen install records the importer from
versions that are already locked and downloads nothing new; a private CI should
assert that the lockfile diff touches nothing outside `importers['packages/cloud']`.

Build-time settings an edition usually sets (all optional, none have vendor defaults
in public code):

| Variable | Used for |
|---|---|
| `VITE_TERMS_URL`, `VITE_PRIVACY_URL` | Terms and privacy links when public-config names none; no link when neither is set |
| `VITE_SUPPORT_EMAIL` | The "Need help?" address on full-window error screens; omitted when unset |
| `VITE_API_URL` | As for any build |

The public client image (`deploy/Dockerfile.client`) deletes `packages/cloud` before
building, so it is always the self-hosted edition. The Cloudflare Workers deployment
files that used to live in the apps (wrangler configs, the edge worker, `_headers`, the
Dash root-index fallback) are kept in `client/packages/cloud/cloudflare/` for the
private build.
