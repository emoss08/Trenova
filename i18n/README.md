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
| `task i18n` | Re-extract and refresh catalogs |
| `task i18n-report` | Inventory: totals, per-area breakdown, what the filter dropped |
| `task i18n-report -- --rejected` | Audit *why* each literal was filtered out |
| `task i18n-check` | CI gate — fails on missing or orphaned entries |
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

## How strings are found

Both extractors key on *syntactic position*, never on the look of the string, because a
regex cannot tell `label="Save"` from `name="save"` or skip an SVG `d="M12 2L2 7"`.

- **Go** (`shared/cmd/i18n-extract`) reads message literals from known constructors —
  `MultiError.Add`, `errortypes.New*Error`, ozzo `.Error(...)`. An unrecognised
  `New*Error` constructor **fails the build** rather than being skipped, so a whole class
  of messages can never go missing quietly.
- **TypeScript** (`tools/extract-ts.mjs`) parses with Babel and collects JSX text, an
  allowlist of prose-bearing props, and `toast.*` calls.

A sentence split by an interpolation is captured whole, with placeholders:

```tsx
<p>Delete "{name}"? This cannot be undone.</p>   ->   'Delete "{0}"? This cannot be undone.'
```

Recording the two halves separately would be untranslatable — Spanish and Chinese order
that sentence differently, and half a clause gives a translator nothing to work with.
