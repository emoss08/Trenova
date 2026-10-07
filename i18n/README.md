# Translations

Trenova ships in English, Spanish, Traditional Chinese (`zh-TW`) and Simplified
Chinese (`zh-CN`).

## What a developer has to do

Write English. That is the whole contract.

```tsx
t("Create Shipment")
```
```go
multiErr.Add("email", errortypes.ErrRequired, "Email is required")
```

Nobody edits a catalog by hand. `task i18n` finds every user-facing string in the Go
services and the React apps and writes them to `messages.en.json`; the per-locale files
hold the translations.

## The English string is the key

There are no `shipment.header.title` keys to invent or keep in sync. Three things follow:

- **A string is translated once.** `"Status is required"` appears at 114 Go call sites and
  in several forms; it is one catalog entry.
- **Most call sites never change.** The Go extractor reads message literals straight from
  the AST, so the existing `errortypes` and ozzo validation calls are already covered.
- **Editing English can't go stale.** New text is a new key, which `task i18n-check`
  reports as missing and CI rejects. The old entry is pruned as an orphan.

## Commands

| Command | Does |
|---|---|
| `task i18n` | Re-extract, refresh catalogs and emit the runtime catalogs |
| `task i18n-report` | Inventory: totals, per-area breakdown, what the filter dropped |
| `task i18n-report -- --rejected` | Audit *why* each literal was filtered out |
| `task i18n-check` | CI gate — fails on missing or orphaned entries, or stale runtime catalogs |
| `node i18n/tools/i18n.mjs pending es --area routes/shipment` | List what still needs translating |
| `node i18n/tools/i18n.mjs merge es batch.json` | Merge a batch of translations |

## Adding a language

1. Add the tag to `locales.json`.
2. `task i18n` — writes `messages.<tag>.json` containing every string, untranslated.
3. Ask Claude to fill it in.

No code changes, no per-app wiring.

## Files

```
locales.json        the languages we ship          <- edit to add one
glossary.json       pinned freight terminology
messages.en.json    GENERATED source inventory     <- never edit
messages.es.json    translations
messages.zh-TW.json
messages.zh-CN.json
tools/              extractors, codemod, catalog commands
```

The runtime catalogs are generated from these by `task i18n` and are never edited either:
`shared/i18n/catalogs/` (embedded in the Go binary) and
`client/packages/shared/src/i18n/catalogs/<locale>/<bundle>.json` (the browser's).

## How the browser loads catalogs

One catalog per locale grew past a megabyte, and the web app could not draw its first frame
in Spanish or Chinese until all of it had downloaded. The client strings are therefore split
into bundles (`tools/bundles.mjs`), by where they are rendered:

| Bundle | Holds strings used | Loaded |
|---|---|---|
| `core` | by `packages/shared`, or by both apps | at startup, by every app |
| `web` | by the web app's shell, or by more than one of its route folders | at startup, by the web app |
| `dash` | only by the driver portal | never by the web app |
| `routes/<dir>` | only inside `client/apps/web/src/routes/<dir>/` | with that folder's code |

An app names its startup bundles once (`<I18nProvider catalogs={...}>`). The web app's
build appends `requireCatalog("routes/<dir>")` to every module in a route folder
(`client/apps/web/vite/route-catalogs.ts`), and every lazy route waits for the bundles its
module graph asked for before it resolves (`client/apps/web/src/lib/route-catalogs.ts`).
So a page renders translated on its first frame, including components it borrows from
another route folder, and a language switch brings every screen already visited with it.
English loads nothing: the key is the text.

Two consequences worth knowing:

- **The route rule has two halves.** `featureArea` in `tools/extract-ts.mjs` labels a
  string from `src/routes/<dir>/` as `routes/<dir>`, and the Vite plugin requires
  `routes/<dir>` for a module in that folder. Change both or neither; `bundles.test.mjs`
  fails if a route bundle has no folder behind it.
- **A component outside the route folders renders in English until something requires its
  bundle.** Shared code (`packages/shared`, `apps/web/src/components`) is in `core` or `web`
  and always loaded. Code loaded by `React.lazy` from a *different* route folder is
  required when it evaluates, so it renders in English for the moment its bundle takes to
  arrive, then re-renders translated.

## How strings are found

Both extractors key on *syntactic position*, never on the look of the string, because a
regex cannot tell `label="Save"` from `name="save"` or skip an SVG `d="M12 2L2 7"`.

- **Go** (`shared/cmd/i18n-extract`) reads message literals from known constructors —
  `MultiError.Add`, `errortypes.New*Error`, ozzo `.Error(...)`. An unrecognised
  `New*Error` constructor **fails the build** rather than being skipped, so a whole class
  of messages can never go missing quietly.
- **Go templates** (`tools/extract-templates.mjs`) collects every `{{ t "..." }}` and nested
  `(t "...")` call in the built-in document templates (area `template/*`) and, when the
  Cloud edition is overlaid, the Trenova Cloud emails the platform sends (area
  `platform-email/*`, from `services/tms/internal/cloud/platformemailservice/templates`).
- **TypeScript** (`tools/extract-ts.mjs`) parses with Babel and collects JSX text, an
  allowlist of prose-bearing props, and `toast.*` calls.

A sentence split by an interpolation is captured whole, with placeholders:

```tsx
<p>Delete "{name}"? This cannot be undone.</p>   ->   'Delete "{0}"? This cannot be undone.'
```

Recording the two halves separately would be untranslatable — Spanish and Chinese order
that sentence differently, and half a clause gives a translator nothing to work with.

## Maintaining the Go extractor

Go source in this repo carries no comments by convention, so the reasoning behind
`shared/cmd/i18n-extract` lives here.

**`messageArgs` — which argument holds the message.** Keyed by the final identifier of the
call, so `NewBusinessError(...)` and `errortypes.NewBusinessError(...)` both match. Indices
come from the real signatures in `pkg/errortypes/errors.go`; note `NewRateLimitError` puts
its message *second*, behind a field name.

**`noMessage` — recognised, but nothing to translate.** Three kinds live here: containers
that only match the `New*Error` name shape (`NewMultiError`, the Prometheus `NewError`);
constructors taking structured identifiers that compose their own text (the formula and
seeder errors, `NewRequestTooLargeError`); and Temporal control-plane errors, which drive
workflow retry decisions and reach operators through job history, never a translated
end-user surface. Moving one into `messageArgs` is the only edit needed if that judgement
changes.

**Why an unlisted constructor fails the run.** Skipping it would drop a whole class of
messages from the catalog with nothing anywhere failing. The gate is what turned up 25
constructors on the first run.

**The `.Error(...)` collision.** ozzo-validation attaches a message to a rule with
`.Error(msg)` — and zap logs with `logger.Error(msg)`. That one entry was pulling 2,187
internal log lines into the catalog. They are told apart structurally: a log call discards
its value (it is an `ExprStmt`), while an ozzo rule is always built to be passed into
`validation.Field`. See `discardedCalls`.

**`stringLiteral` follows concatenation** so a message split across lines for width is
captured whole, and refuses anything involving a variable or call — those are dynamic and
need an explicit parameterised message instead.
