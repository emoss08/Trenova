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

Write the **whole sentence** as one message. A count is a plural, not an English word
passed in, and a sentence with a link or bold part in it is one rich message:

```tsx
t("{0, plural, one {# stop} other {# stops}} past free time", count)
rt("Use <link>shared profiles</link> instead.", { link: (c) => <Link to="…">{c}</Link> })
```

Never assemble interface text in a template literal (`` `${count} selected` ``): only
string literals passed to `t`, `translate`, `rt` or `translateRich` reach a catalog.

A module-level label map is declared with `defineLabels` (`@trenova/shared/i18n/labels`), so
its captions reach the catalog and read in the current language wherever they are shown:

```ts
export const DELIVERY_LABELS = defineLabels({ Online: "Online", OnTheJob: "On the job" });
```

`task i18n-check` rejects a sentence split around markup, English handed to a message,
English built in a template literal where a person reads it, an undeclared label map, and a
literal validation message or toast (`tools/fragments.mjs`; a
template that is not interface text takes `// i18n-ignore: <reason>`); see the i18n section of
[generated-artifacts.md](../docs/engineering/generated-artifacts.md) for the rules.

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
build (`client/apps/web/vite/route-catalogs.ts`) then does two things to its source:

- it appends `requireCatalog("routes/<dir>")` to every module in a route folder, so any
  code that loads the module has asked for the strings it renders;
- it chains `.then(afterCatalogs)` onto every dynamic import of app code, so the import
  resolves only once the bundles its module graph required have landed.

Lazy routes wait the same way (`client/apps/web/src/lib/route-catalogs.ts`). The result is
that nothing ever renders in English and then again: a page, or a `React.lazy` panel
borrowed from another route folder, keeps showing its loading skeleton until its strings
are in, and its first frame is translated. A language switch brings every screen already
visited with it. English loads nothing and never waits: the key is the text.

If a bundle fails to download, the screen renders in English rather than not at all, and
the bundle is retried on the next navigation.

**The route rule has two halves.** `featureArea` in `tools/extract-ts.mjs` labels a string
from `src/routes/<dir>/` as `routes/<dir>`, and the Vite plugin requires `routes/<dir>` for
a module in that folder. Change both or neither; `bundles.test.mjs` fails if a route
bundle has no folder behind it.

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
