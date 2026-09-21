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

## The hue plan

Everything coloured in the product resolves to one of eleven hues, declared once
in layer 1.

`--hue-neutral: 260` is the spine, a cool slate. Every grey carries a trace of it
— the canvas, the panels, the rules, the four text weights — low enough that a
panel still reads as white and high enough that the ink reads as ink rather than
as black. A grey at chroma exactly zero is the clearest sign a palette was
inherited rather than chosen.

**The product is drawn in ink.** The primary action is `--ink`, the foreground
colour with its own hover and pressed rungs, not a hue. That is what lets every
colour left on screen mean something: a tone is a severity, an accent is a
category, and the brand marks place and affordance.

`--hue-brand: 262` is cobalt, and it is spent sparingly: links, the focus ring,
selection (a selected row, selected text, a checked box), the active nav row and
tab, the first chart series. It is never the primary button and never a severity.
If you are about to write `bg-brand` on a button, you want `variant="default"`.

Cobalt shares the blue arc with three other things, so that arc is pinned apart:

```
info 222  →  sky 232  →  brand 262  →  indigo 283
```

Brand keeps 30° from info and 20° from each category. Info is a steel cyan at
under half of cobalt's chroma for the same reason: a link and an "In transit"
badge share a row in nearly every table, and they must not read as one thing.

The warm arc keeps its own floor:

```
danger 25  →  warning 78  →  amber 98
```

`pnpm lint:design` fails if any of those gaps closes.

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

`--foreground-on-solid` is near-white in light mode and near-**black** in dark,
because a dark theme's tone fills are the light end of their ramp. Warning is
the exception in light mode too: an amber dark enough to carry white text is a
brown, and the solid is spent almost entirely on dots and bars, so `--warning` is
a true amber and the ink on it is `text-warning-on-solid`, dark in both themes.
`Badge` handles this itself; hand-written `bg-warning` with text on it must not
use `text-foreground-on-solid` or `text-warning-foreground`. `--warning`
shipped for months as `oklch(0.75 0.16 70)` and drew near-white text on a solid
badge at 2.1:1; every solid fill now clears AA against the ink that lands on it,
and the check asserts it.

A tone used as **text** is its readable rung. `text-warning`, `text-success`,
`text-info`, `text-danger` and `text-destructive` resolve to `--x-foreground`
through Tailwind's text-colour namespace, while `bg-warning` stays the solid fill:
the solid is chosen to be seen at six pixels, not read. A tinted ground is
`bg-x-subtle` with `border-x-border`, never `bg-x/10` with `border-x/30` — the
opacity forms were written 362 times at a dozen strengths and none was audited.

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
| `bg-field` | the ground of a form control — use `ui-field`, below, rather than this directly |
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

Labels are sentence case. Do not write `uppercase tracking-wider` on a section
label, a column head or a badge: 375 of them were removed, because a screen of
tracked-out capitals reads as a template and costs legibility at 10px. `uppercase`
is for data that *is* uppercase — a SCAC, a state code, a VIN.

Fonts: `font-sans` (Geist) for everything; `font-mono` (Geist Mono) for numbers
that must align in a column — load numbers, IDs, currency, timestamps. Pair it
with `tabular-nums`.

### Weight

Three steps, and each has to mean something:

| Weight | For |
|---|---|
| `font-normal` (400) | body copy, table cells, values |
| `font-medium` (500) | labels — column heads, field labels, badges, button text |
| `font-semibold` (600) | headings — card and panel titles, section titles |

`font-medium` was written 1,794 times against 94 `font-normal`, so 500 became the
weight everything was set in and emphasis had to be spent on size or colour
instead. The primitives that stamp a weight now follow the table above; a value
in a cell takes no weight class at all.

## Radius

Two values, plus the pill.

| Token | Value | For |
|---|---|---|
| `--radius-control` | 6px | buttons, inputs, chips, menu rows |
| `--radius-surface` | 8px | cards, panels, dialogs, popovers, table containers |
| `rounded-full` | — | badges and avatars |

There were seven in use — `rounded-md` 1,032 times, `rounded-lg` 780, bare
`rounded` 268, `rounded-sm` 108, `rounded-2xl` 99, `rounded-xl` 46 — which is the
same as having none, because nothing could be told apart by its corner. The whole
Tailwind scale is now pointed at the two real values, so existing call sites are
unchanged and an eighth corner has nowhere to land.

## Density

Row rhythm is a token, not whatever padding a cell happened to carry.

| Token | Value | For |
|---|---|---|
| `--row-h` | 30px | table body row, comfortable |
| `--row-h-compact` | 26px | table body row, compact |
| `--row-head-h` | 32px | column header row |
| `--cell-px` | 10px | horizontal cell padding, header and body alike |

`TableRow` spends `h-(--row-h)` and `TableCell` spends `px-(--cell-px)`, so two
tables on the same screen line up. A denser table repoints the token —
`[--row-h:var(--row-h-compact)]` — rather than patching padding onto every cell.

## Elevation

There is none. Trenova draws no shadows, anywhere. A surface is told from the one
beneath it by a hairline and a step in lightness; a floating surface by its
`ring-1 ring-foreground/10` and by being inverted (below). A shadow is a blur, and
a blur is the one thing on a dense screen that cannot be aligned to anything.

The four `--elevation-*` tokens still exist and are all `0 0 #0000`, and the whole
Tailwind `shadow-*` scale points at them, so a stray `shadow-md` lands on nothing
rather than on Tailwind's default. Do not give them a value. `pnpm lint:design`
fails on any `shadow-sm|md|lg|…` or coloured `shadow-black/15`.

Two things are box-shadows in CSS and lines on screen, and are fine: the focus
ring, and a zero-blur outline such as `shadow-[0_0_0_1px_var(--brand)]` on a
selected card or the inset hairline on a pinned column.

## Floating surfaces are inverted

Everything that floats from a trigger is dark in light mode: dropdown and context
menus, selects, popovers, hover cards, tooltips. Dialogs and sheets are not — they
replace the page rather than float over it, and follow the theme.

The primitives do this themselves by putting the `dark` class on the positioner,
so the popup and everything in it resolves the dark token set. Never write `dark`
on a `PopoverContent` by hand. A popover that genuinely must follow the theme
takes `inverted={false}`; expect to be asked why.

Because the content is in a dark scope in both themes, anything inside a popover
must be built from tokens. A hard-coded light value that "works" in light mode is
wrong here even before dark mode is considered.

## Controls

`ui-field` owns every state of a form control's box, so all of them behave alike:

| State | Treatment |
|---|---|
| rest | `--field` fill, `--input` hairline. The fill never shares a value with the canvas or the card; it sits below the surface in both themes |
| hover | hairline steps to `--border-strong`; the fill does not change |
| focus | the one focus ring (`ui-focus-ring`, or `ui-container-focus-ring` when a child takes focus) |
| open | a trigger whose popup is open holds that same ring (`data-pressed`, `data-popup-open`, `aria-expanded`) |
| invalid | `fieldInvalidClass`: danger hairline, 10% danger fill, and `--ring` repointed so focus and open turn red |
| disabled | `--sunken` fill, 60% opacity |

`Input`, `Textarea`, `SelectTrigger` and `NumberField` spend it, and so does every
app field — select, autocomplete, multi-select, date, colour, money, number, chips,
phone. A trigger built on `Button` takes `fieldTriggerClass` from
`@trenova/shared/lib/variants/field`, which adds the overrides that stop the
button's own hover and pressed fills from showing through. Do not write
`border-input bg-muted` on a control, and do not assemble a `data-pressed:ring-*`.

`ui-press` gives a filled control its pressed state (a 2.5% give). `Button`
carries it; a hand-built clickable tile that should feel like a button takes it
too.

`ui-shimmer` is the loading sweep. `Skeleton` spends it; do not reach for
`animate-pulse`.

## Motion

One curve family and default speeds, declared in the `@theme` block, so a bare
`transition-colors` already eases the house way.

| Token | For |
|---|---|
| `ease-swift` (the default) | things answering the pointer: hover, press, focus |
| `ease-settle` | things arriving: popovers, sheets, a sliding tab indicator |
| `ease-spring` | a small overshoot, for the one thing that confirms an action |
| `animate-rise` | content arriving in place |
| `animate-confirm` | a check landing, a copied tick |

Motion answers an action. Nothing on a working screen moves on its own: no
looping shimmer on a badge, no pulsing glow, no floating sparkle. A global
`prefers-reduced-motion` rule collapses every animation and transition to a cut.

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
build, a test or a type check. This happened once. `pnpm lint:design` now
compiles the real file and asserts every declared `@utility` reaches the output,
so it cannot happen quietly again.

## Page anatomy

Every page is `PageLayout` from `components/navigation/sidebar-layout.tsx`. There
is no second layout; a page that builds its own top bar is a bug.

**The title bar** is one 44px row: the title at `text-lg` semibold, the
description behind an info icon, actions on the right. The workspace bar above it
already shows where you are, so the page does not spend two more lines saying it
again. Pass `title`, `description`, `actions` and `context` through
`pageHeaderProps`; never mount `PageHeader` by hand.

**A list page bleeds.** When the page body's only child is a data table, the body
drops its padding and the table runs edge to edge: no outer box, no page gutter,
only row hairlines, with the toolbar and the pagination as bars above and below.
Nothing opts in — the body detects it, and the `bleed:` variant lets the table's
chrome answer. A table that shares the page with anything else keeps its frame.
So a list page is exactly this, with no wrapper `div` around the table:

```tsx
<PageLayout pageHeaderProps={{ title: t("Commodities"), description: t("…") }}>
  <DataTableLazyComponent>
    <Table />
  </DataTableLazyComponent>
</PageLayout>
```

**Everything else gets `p-4` and `gap-y-4`** from the body. Do not pass
`className="p-0"` and then re-add `mx-4 mt-3 mb-4` to each child; that idiom put
three different gutters on sibling pages. `p-0` is for a genuine split-pane
workspace that manages its own panes.

**A full-height workspace takes `fill`.** `<PageLayout fill>` makes the page exactly
as tall as the viewport leaves it and lets the body scroll, so the workspace inside
is `flex-1 min-h-0` (with a `min-h-*` floor if it needs one). Do not size a
workspace with `h-[calc(100vh-9.5rem)]`: that number encodes the height of the
chrome above it, and it was wrong the day the title bar changed.

**Create is "New …".** The button, the menu item, the empty state and the panel
title all say `New {thing}` in sentence case, and the button is the ink default.
`toSentenceFragment` from `@trenova/shared/lib/utils` lowers a name without
breaking an acronym ("EDI Partner" → "EDI partner").

### Figures: `KpiStrip`

A row of figures is one joined strip divided by hairlines, not a row of floating
cards: `KpiStrip` with `KpiStripItem` (`components/kpi/kpi-strip.tsx`). Label over
number, no decorative icon, the delta as small toned text with an arrow. A `tone`
puts a status dot beside the label; `onClick` makes the cell a filter and `active`
fills it with the selection colour.

The number is `text-xl` semibold, proportional with `tabular-nums` — `size="lg"`
(`text-2xl`) for the one strip that leads a dashboard. There were fifteen tile
implementations and six number sizes; there is one and two. `KpiCard`, `KpiStat`
and both `StatTile`s render as strip cells when placed inside a `KpiStrip`.

### Panels: `SectionPanel`

A titled block on a dashboard or workspace is `SectionPanel`
(`components/section-panel.tsx`): `rounded-lg border bg-card`, a 36px header with
the title at `text-sm` semibold, an optional `help` note, a `count`, and `action`
on the right. Do not hand-write `<header className="… border-b px-3 py-2">`.

### Label and value: `DescriptionList`

Read-only detail is `DescriptionList` + `DescriptionItem`
(`@trenova/shared/components/ui/description-list`). Three layouts: `stacked`
(label over value, in `columns`), `inline` (label left, value right, aligned in a
grid) and `split` (label and value at opposite ends of a ruled row, for totals).
The label is `text-xs` medium in `--foreground-subtle`; the value is `text-sm` at
weight 400, because a value in a cell takes no weight. `numeric` adds
`tabular-nums`. An absent value is `<DescriptionEmpty />`, an em dash — never a
hyphen.

### Forms, dialogs and sheets

- `FormSection` titles are `text-base` semibold sans. Separate sections with the
  parent's gap, not with per-section `border-t pt-4`.
- A settings page is stacked `Card`s and a `FormSaveDock`; the dock's label is
  "Save changes" and is not overridden.
- A dialog, a sheet and a table panel all close the same way: footer right-aligned,
  Cancel (`outline`) then the primary (ink, or `destructive`). `DialogContent` takes
  `size` (`xs`…`2xl`) instead of an arbitrary `sm:max-w-[520px]`.
- An inline callout is `<Alert variant size="sm">`, not a hand-tinted box.
  `bg-warning/10 rounded-md p-3` was written sixty times at fifteen opacities.

## The assist mark

Anything the system suggests, drafts or fills in on its own is marked with
`AssistMark` (`@trenova/shared/components/ui/assist-mark`): the diamond of an
advisory road sign with a point at its centre. On the road that shape means "take
this into account", which is the standing a machine's suggestion has with a
dispatcher. It is a real `LucideIcon`, so it takes `size`, `strokeWidth` and
`className` and fits any `icon:` slot.

Do not import `Sparkles`, `WandSparkles` or `Wand2` from lucide. Sparkles says
magic, which is the wrong promise for a rate or a settlement, and it is the glyph
every generated interface spends. The assistant's launcher uses `AssistantMark`,
the same diamond with a centre point that breathes on hover.

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

A badge is a pill. It is the one thing on screen that is never a container, and
the shape says so before the colour does — which is also why it is the only
component that does not take one of the two radii.

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
4. Run `pnpm lint:design`. It reads the oklch values straight out of `tokens.css`
   and fails on a pair below AA, a value outside sRGB (the browser would show it
   somewhere other than where you placed it), a hairline that is invisible
   against the surface it divides, or either hue arc closing up.
5. Note it here if it introduces a new concept rather than a rung on an existing
   ladder.

The audit is not a substitute for looking at it. It catches the failures that are
invisible in review — a 2.1:1 badge passes a screenshot check because the text is
*there* — but it has nothing to say about whether the colour is right.

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
pnpm lint:design      # the six rules, with the token to use instead
node scripts/tw-probe.mjs bleed:px-3 text-warning   # does this class emit, and as what?
pnpm lint             # oxlint
pnpm typecheck        # Badge variants and status phases are typed
```
