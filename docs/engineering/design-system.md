# Design system

Tokens live in `client/packages/shared/src/styles/tokens.css`. Components consume
them and define no colour, size or elevation of their own. `pnpm lint:design`
enforces that and runs in CI.

This document says which token to reach for. If none fits, add one — a missing
token is the reason the previous set was bypassed 1,914 times.

## Why the rules are shaped this way

The token set that preceded this one was not ignored; it was incomplete.
`text-destructive` was used 311 times because it existed and was obvious.
`bg-amber-500` was used 77 times because `bg-warning-subtle` did not exist. The
fix was to complete the ladders, not to tighten the rules.

Four defects came out of that audit and are worth knowing about, because each
explains a rule below:

- `--font-mono` was `"Inter Variable", sans-serif`. The `font-mono` utility is
  spent 452 times on load numbers, IDs, currency and timestamps, none of which
  aligned in a column.
- The type scale stepped 10.8 / 11.6 / 12.4 / 13.2px — below the threshold at
  which anyone can tell which is correct — and hit no round number. `text-[11px]`
  was written 424 times, more than `text-lg`, `base`, `xl`, `2xl` and `3xl`
  combined. The scale was not being violated; it was being routed around.
- `--secondary`, `--muted` and `--accent` all held `oklch(0.97 0 0)` in light and
  three different values in dark. Light-mode work could not reveal dark-mode
  inconsistency.
- Light mode defined no `--shadow-*` at all and fell through to Tailwind's
  defaults while dark mode used custom values.

## Layers

Three, and each may only reference the one above it.

| Layer | What | Use in components? |
|---|---|---|
| Primitives | `--hue-*`, the eight categorical hues | No |
| Semantic | `--danger`, `--sunken`, `--foreground-subtle`, … | Yes — this is the API |
| Mapping | the `@theme` block that makes utilities | No |

## Colour

### Tones — severity

Six tones. Each has five rungs, and the rung is chosen by what the colour sits on.

| Rung | Utility | For |
|---|---|---|
| `--x` | `bg-danger` | solid fill: filled buttons, strong badges, status dots |
| `--x-foreground` | `text-danger-foreground` | readable text and icons on a normal surface |
| `--x-subtle` | `bg-danger-subtle` | tinted surface: soft badges, callout rows |
| `--x-subtle-foreground` | `text-danger-subtle-foreground` | text **on** `--x-subtle`, verified ≥ 4.5:1 |
| `--x-border` | `border-danger-border` | hairline around `--x-subtle` |

The tones are `neutral`, `brand`, `info`, `success`, `warning`, `danger`.

`--x-foreground` keeps the meaning it already had here — *the readable version of
this tone* — which is why it is darker than `--x`. Text on a **solid** tone fill
is `text-foreground-on-solid`, not `--x-foreground`.

Pick a tone by what the operator should do, not by what the thing is called. An
overdue invoice is `warning`. A shipment in transit is `info`, not `warning` —
nothing is wrong with it.

### Accents — category

Eight categorical accents on the same hue family: `accent-indigo`, `accent-teal`,
`accent-amber`, `accent-rose`, `accent-emerald`, `accent-sky`, `accent-violet`,
`accent-slate`. Rungs are `--accent-x`, `--accent-x-on-subtle`,
`--accent-x-subtle`, `--accent-x-border`.

Use an accent when the set has **no severity ordering**: a HOS duty status, an
agent trigger type, a pricing method, W-2 versus 1099. None of those members is
further along or worse than another, and giving one a tone says something false —
marking "On Duty" as a warning, or "Flat" as a success.

Use a tone whenever the set *does* have an ordering. If you cannot decide, it has
an ordering.

The same eight hues also drive chart series and agent identity, so a teal series
and a teal agent are the same teal.

### Surfaces

A ladder, not a pile. Each has one job.

| Token | For |
|---|---|
| `bg-canvas` | the page ground |
| `bg-card` | panels and cards on the canvas |
| `bg-sunken` | wells, table headers, code blocks — recedes from the card |
| `bg-raised` | popovers, menus, combobox lists |
| `bg-overlay` | dialogs and sheets |
| `bg-surface-hover` / `-active` / `-selected` | interaction fills |

Interaction fills are separate from container surfaces on purpose, so a hover
state never has to borrow a container's colour. If you find yourself writing
`bg-muted/30` to invent a shade, the rung you want is above — that pattern
accounted for 1,036 uses before this ladder existed.

### Content and borders

`text-foreground`, `text-foreground-muted`, `text-foreground-subtle`,
`text-foreground-on-solid`. Anything lighter than `subtle` is decoration, not
content, and belongs to a border.

`border-subtle` for dividers inside a component, `border` for component outlines,
`border-strong` for hover and emphasis.

## Type

Whole-pixel steps, each carrying its line-height. Do not write `text-[11px]`;
`text-xs` *is* 11px, and unlike the arbitrary value it brings a line-height.

| Step | Size | For |
|---|---|---|
| `text-3xs` | 9px | the floor; dense table micro-labels |
| `text-2xs` | 10px | micro labels, table meta |
| `text-xs` | 11px | dense body — the workhorse |
| `text-sm` | 12px | default body |
| `text-base` | 13px | comfortable body |
| `text-lg` | 15px | section titles |
| `text-xl` | 17px | subsection headings |
| `text-2xl` | 20px | page titles |
| `text-3xl` | 24px | page headers |
| `text-4xl` / `5xl` | 30 / 36px | display |

Trenova is deliberately denser than its peers: body text is 12px against roughly
14px at Cloudflare and Azure. That is a product decision, not an oversight.

Display sizes (`xl` and up) carry optical letter-spacing already, so a
`tracking-tight` on a heading is usually redundant.

Fonts: `font-sans` (Inter) for everything; `font-mono` (Geist Mono) for numbers
that must align in a column — load numbers, IDs, currency, timestamps. Pair it
with `tabular-nums`.

## Elevation

Four steps, defined in both themes. Enterprise surfaces are flat: a card is a
border, not a shadow. Reserve shadow for things that genuinely float.

`shadow-flat`, `shadow-raised` (cards), `shadow-overlay` (popovers, menus),
`shadow-modal` (dialogs, sheets).

## Focus

One ring, three utilities. Never assemble a ring out of `focus-visible:ring-*`,
`focus-visible:border-*` or `focus-visible:outline-*` — `pnpm lint:design` fails
on those. There were **34 distinct focus treatments across 16 primitives** before
this, which is why focus never looked like one system.

| Utility | For |
|---|---|
| `ui-focus-ring` | focus lands on the element itself |
| `ui-container-focus-ring` | focus lands on a child and the wrapper draws the ring — a field with an affix, a composer |
| `ui-inset-focus-ring` | no room to bloom outward — a table row, a segmented-control thumb |

An invalid control needs no second ring. Repoint `--ring` locally and the same
utility turns red:

```tsx
<Input className="aria-invalid:[--ring:var(--ring-danger)]" />
```

**One caution when editing `tokens.css`:** never let a comment-terminator
sequence appear inside a comment body. CSS comments do not nest, so it ends the
comment early and every `@utility` after it silently stops generating a rule —
a focus utility that emits nothing removes the focus indicator without failing a
build, a test or a type check. This happened once. `styles/__tests__/tokens.test.ts`
now compiles the real file and asserts every declared `@utility` reaches the
output, so it cannot happen quietly again.

## Badge

Two axes, neither of them a colour.

```tsx
<Badge variant="warning">3d overdue</Badge>              // tone, subtle by default
<Badge variant="success" appearance="solid">Paid</Badge> // tone, solid fill
<Badge variant="neutral" appearance="outline">Draft</Badge>
<Badge variant="accent-teal">Yard move</Badge>           // categorical
```

`variant` is a tone or a categorical accent; `appearance` is `subtle` (default),
`solid` or `outline`.

Variants used to be named by colour — `purple`, `orange`, `teal`. Once a variant
is called `purple` there is no correct answer to "what colour is a tender?", so
every caller answered differently. That is what the split above prevents.

## Status metadata

Status maps declare a **lifecycle phase**, never a colour. The tone follows from
the phase, so a new status cannot invent one.

```ts
const attrs: Record<InvoiceStatus, BadgeAttrProps> = {
  Draft:     { phase: "draft",     text: t("Draft") },
  Submitted: { phase: "awaiting",  text: t("Submitted") },
  Posted:    { phase: "complete",  text: t("Posted") },
  Rejected:  { phase: "failed",    text: t("Rejected") },
};

<Badge variant={phaseTone(attrs[status].phase)}>{attrs[status].text}</Badge>
```

| Phase | Tone | For |
|---|---|---|
| `draft` | neutral | not started, still editable |
| `queued` | neutral | accepted, waiting its turn |
| `active` | info | under way and on track |
| `awaiting` | warning | blocked on a person or outside party |
| `attention` | warning | off the happy path, needs intervention |
| `complete` | success | finished as intended |
| `closed` | neutral | finished, no longer actionable, did not fail |
| `failed` | danger | cancelled, rejected, expired, errored |

For a set that is a classification rather than a lifecycle, use
`BadgeClassAttrProps` with an `accent` instead. Reach for it only when no phase
honestly applies.

## Charts

`--chart-1` … `--chart-8`, on the categorical hue family. Light and dark differ
only in lightness, never in hue, so a series keeps its identity across themes.

Do not hand-pick series colours. Before this, light was a five-hue rainbow and
dark was two hues plus three greys, so the same chart encoded its data
differently depending on the theme.

## Adding a token

1. Check no existing token covers it. Most "missing" colours are a rung you have
   not met — `--x-subtle` and `--x-border` cover most tinted-callout cases.
2. Add it to **both** `:root` and `.dark` in `tokens.css`. A token defined in one
   theme is the `--shadow-*` bug again.
3. Map it in the `@theme` block so it reaches a utility.
4. Check contrast: anything carrying text needs ≥ 4.5:1 against its background
   (≥ 3:1 for large text).
5. Note it here if it introduces a new concept rather than a rung on an existing
   ladder.

## Colours outside the token system

Some colours genuinely cannot be tokens. Roughly 200 hex literals remain, and
each falls into one of these:

| Category | Why it stays |
|---|---|
| Brand marks and national flags | A Microsoft logo and a US flag are specific colours, not themeable ones |
| Map overlays | The Maps API rasterises a literal; `var(--x)` never resolves there |
| Print output | Paper has no dark mode |
| The colour picker | Hex *is* the data the user is choosing |
| Recharts selector overrides | `[stroke='#ccc']` matches what Recharts writes inline — it is what the token overrides |
| Devtools console colours | Never rendered in the app |

Everything else is a token. Notably, **a select option's indicator colour is a
token**, because it renders through `style={{ backgroundColor: color }}` and
inline styles resolve custom properties — so `var(--success)` follows the theme
where `#15803d` stayed the same dark green on a near-black surface:

```ts
export const statusChoices = [
  { label: "Active", value: "Active", color: "var(--success)" },
  { label: "Draft", value: "Draft", color: "var(--foreground-subtle)" },
];
```

Where one list needs two colours from the same family, take a different **rung**
of that family (`--danger` then `--danger-foreground`) rather than a different
hue, so the options stay distinguishable without a red turning teal.

Chart configs take `color` rather than `theme: { light, dark }` for the same
reason: the token already themes itself, so the pair cannot drift apart.

## The escape hatch

Put `design-tokens-ignore: <reason>` in a comment on or just above the line. It
is deliberately visible in review; a silent exception is how the last set eroded.

## Checking your work

```bash
pnpm lint:design      # the four rules, with the token to use instead
pnpm lint             # oxlint
pnpm typecheck        # Badge variants and status phases are typed
```
