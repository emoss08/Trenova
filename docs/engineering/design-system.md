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

Nothing that sits **in** the page draws a shadow. A surface is told from the one
beneath it by a hairline and a step in lightness; a floating surface by its
`ring-1 ring-foreground/10` and by being inverted (below). A shadow is a blur, and
a blur is the one thing on a dense screen that cannot be aligned to anything. A
table row, a field, a card, a panel: all flat, all the time.

The four `--elevation-*` tokens still exist and are all `0 0 #0000`, and the whole
Tailwind `shadow-*` scale points at them, so a stray `shadow-md` lands on nothing
rather than on Tailwind's default. Do not give them a value. `pnpm lint:design`
fails on any `shadow-sm|md|lg|…` or coloured `shadow-black/15`.

Two things are box-shadows in CSS and lines on screen, and are fine: the focus
ring, and a zero-blur outline such as `shadow-[0_0_0_1px_var(--brand)]` on a
selected card or the inset hairline on a pinned column.

### Lift

The rule above is about the page. The few things that sit **above** it and are
meant to be picked up — a composer floating over a conversation, an artifact
card in flight, a menu — have their own family, and it is named by where it is
allowed rather than by how big it is:

```
ui-lift-whisper   a hairline ring and one pixel of depth
ui-lift           something you could pick up: a floating composer, a card
ui-lift-float     something over everything else: a switcher, a menu
```

Each is a hairline ring plus a whisper of shadow at `--hue-neutral`, so it reads
as the surface occluding light rather than as grey smudge. The blur is small and
the offset smaller: a lift should be felt before it is seen. In dark the shadow
hue goes near-black, because a grey halo on an already dark surface reads as
fog, not depth.

The naming is the whole guard. There is no `shadow-lg` to reach for and no size
to argue about — you either are a thing that floats, in which case one of three
names fits, or you are not, in which case none of them do. A table row with a
lift is still the defect the rule above was written against.

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

## Assistant surfaces

The assistant is not a chat widget with cards in it. It has a voice inside the
system, and these are the things that carry it.

**The agent's accent is light, not a line.** This used to be a two pixel
hairline down the avatar gutter, and the bar was wrong three ways: it ran the
full height of a scrolling column to carry one bit, it put a hard vertical at
full saturation beside prose, and on a 400px panel it spent a column of width
on it. `AgentGutter` (`components/assistant/voice/agent-gutter.tsx`) now wears
`ui-agent-glow`: a radial wash at the head of the column in the agent's accent,
brightest where its mark is and gone within 200px. It carries two facts at
once — `--agent-glow-rest` says who, and `data-working="true"` lifts it to full
while a turn runs. A surface that only wants the second, like the Desk's
header, leaves the rest at zero.

This is the one place an accent becomes an area rather than a line. Everywhere
else the agent's colour is its mark.

**An agent's mark has to be its own.** `AgentTile` layers three things: the
icon the organization chose, the accent, and a **sigil** — a short arc around
the tile derived from the agent's id. The sigil exists because sixteen icons
and eight accents collide constantly when both are hash-assigned, and an agent
with no starter gets the same robot as every other one, so four desks wore one
face. A mark also knows whether its icon is genuinely the agent's: when it is
not, it draws the agent's initials instead of the fallback. One line, one
weight, nothing to decode — a person only has to notice that two marks are not
the same.

**Motion has four verbs.** They live in `tokens.css` as `--animate-*` and each
means one thing:

```
land         a step of an agent's work arrives   (from the left: the agent's side)
materialise  produced work appears in the workspace (in from the conversation)
draw         a line extends
breathe      work is running
```

`breathe` is a loop, and it earns the exception the way a heartbeat monitor
does: it runs because the thing it describes is still going, and it stops when
that stops. It is one shape, `WorkingDot` in
`components/assistant/voice/working-dot.tsx` — a dot in the agent's accent —
used everywhere "still going" is said in passing: a tool step, a hand-off, a
plan step, a running decision, a report run, the composer's "Replying" hint and
a conversation's row. A spinner here and a pulse there is two vocabularies for
one fact. A list takes it `still`, because a column of pulses is movement
nobody can escape. Everything else still answers an action, and
`prefers-reduced-motion` keeps the state change and drops the travel.

**The desk at work is on the working line and nowhere else.** The one other
loop is `DeskThinking` in `components/assistant/voice/desk-thinking.tsx`: a
small desk scene at the height of a line of text (`h-4`, 16px), beside the
words of the working line under a reply in progress, in the Desk and in the
popover alike. It is a person in a chair at a desk with a monitor on it and a
lamp, drawn on a 20 by 16 grid in solid shapes — never hairlines, which turn
to mush at 16px. It is the one illustration in the product, so it is the one
place with a palette of its own, and the ink-only rule does not govern it; the
token-only rule does:

| Part | Colour |
|---|---|
| the person, the screen, the screen's glow | `--desk-accent`: the agent's accent, the brand where none is set |
| the chair | `--desk-chair`: the accent sunk into the furniture's ink |
| the desk, its pedestal | `--desk-wood`, `--desk-wood-shade` |
| the monitor's frame, the keyboard, the lamp's arm | `--desk-frame` |
| the lamp's shade and its light | `--desk-lamp`, `--desk-lamp-light` |
| what is on the screen | `--desk-screen-ink`: near-white in light, near-black in dark, because a dark theme's accent is the light end of its ramp |

`ui-desk-mark` reads the accent on the mark itself, so the scene takes the
colour of whichever agent's turn it sits in. It moves the way a sprite does, a
few small frames swapped on a beat rather than anything travelling across the
line, and every motion is transform or opacity. The screen tells the story,
with one pose per phase of a turn, chosen by `thinkingPose` from the same state
the working line's words are read from, so the drawing and the sentence never
disagree:

| Pose | When | What moves |
|---|---|---|
| (arrival) | the mark appears | the chair rolls out, the person sits, and it rolls back in to the desk, once (`animate-desk-sit`, `animate-desk-sit-figure`) |
| `arrive` | the question is being checked, the model is thinking, a reply is starting over | three dots on the screen, one frame at a time (`animate-desk-think-*`) |
| `busy` | a tool is running | the hands bob on the keys and the screen scrolls a line a frame (`animate-desk-type`, `animate-desk-scroll`) |
| `write` | the answer is arriving | lines are written onto the screen one after another while its light swells and flickers softly (`animate-desk-write-*`, `animate-desk-glow`) |
| `settle` | the turn is over, however it ended | the chair eases back from the desk while the scene fades (`animate-desk-settle`, `animate-desk-fade`), then the mark is gone |

It is announced as one `role="status"` named "Working on your answer", whose
content never changes, so a screen reader hears the work start and is never
told about the drawing moving; the SVG itself is `aria-hidden`. Under
`prefers-reduced-motion` every pose is one still frame of the same moment —
dots, lines or a lit screen — and the settle is skipped.

**Artifacts are the product of a turn.** A report answer is a table, an email is a
draft, a plan is a checklist, a record is a card. They render in the pane beside the
conversation and the transcript only refers to them, as chips. Every artifact sits in
`ArtifactChrome` (`components/assistant/voice/artifact-chrome.tsx`): `--radius-surface`,
`border-border`, a 48px header with the kind's mark in a sunken well, the title (semibold)
over one provenance line — "Report · from Report builder · 2 minutes ago" — a status badge
only when the status is not Ready, and the pin. No shadow. The kind's name, icon and source
come from `ARTIFACT_KINDS`, once, for every surface that names one; a body with nothing to
show is `ArtifactNotice`, never a bare grey sentence. Status is a tone
(Pending is `info`, Sent is `success`, Failed is `danger`); the kind is never coloured.

**An artifact shows what a person can reason with, never what the model worked from.** A
tool's result carries ids, the tenant, a write's version and which way a metric counts as
worse; the artifact stores a projection of it instead (`assistantservice/artifact_display.go`,
mirrored for older payloads and the step details in `components/assistant/readable-values.ts`).
Every column and field has a display type — text, prose, date, date and time, money, number,
percent, category, status, yes/no, flag, measurements, links — and anything without one is
left out rather than printed as JSON. A record's id survives only as the key its row's link is
built from, through `recordPath`. Drawn, a table is the row tokens at the compact rhythm: the
record's name first and pinned while the rest scroll sideways, a status as a phase-toned badge,
a category as words, figures right-aligned in tabular numerals, dates in the reader's own
timezone, and prose and measurements in the row's detail (a `bg-sunken` well) rather than in a
cell. A card is a `DescriptionList` with the same values, its prose set out below the fields.
A flag that does not hold — "stale: no" — is not drawn at all.

**The conversation says what the agent did, in the words of what it did.** A person's
message sits in a `bg-sunken` well under "You" — never their name and avatar, which read
as someone else once the thread was shared. The reply runs open across the column under
the agent's mark and name, once: a reply that took four model steps is saved as four
messages, and `turnPlacements` heads the first and continues the rest beneath it, with
how long the whole reply took beside the time. Both share the left edge; the difference
is voice, not side.

Tool calls are drawn by their **effect** — `lookup`, `change`, `navigate`, `discover`,
`present`, `ask` — which the server sends on every call and `toolEffect` derives from the
name when it does not. Reads fold into one line ("Looked up 3 records", "Found 25+
records"); an action never does, because "opened a page" counted as "looked up 1 record"
is the report that started this. An action stands on its own line in ink with its mark in
a small sunken well — "Opened Report library", "Ran Late loads", "Saved Shipments
for Peak Distributing", "Proposed a change" — and a read is quieter, in muted text with a
bare mark. The verb is the client's and translated; the server's one-line `summary` only
ever supplies a name, and a count phrase from it ("3 customers") is read for its number,
never shown. A line opens onto the call as labelled values — what was asked, what came
back, list results named by their first few records — with the literal JSON behind a
second "Details" click.

While a reply is being written, one **working line** at its foot says what is happening
now — "Looking up Peak Distributing…", "Writing the answer…" — with the step count and the
elapsed time (and how many steps run at once, when more than one does), beside the desk at
work. The desk is the size of the text beside it, before the first word as much as after it.
The plain moments have a few ways of being said — "Thinking…" or "Pulling up a chair…" to
begin, "Writing the answer…" or "Putting it into words…" while it streams — and a turn keeps
one of them from start to finish, so replies do not all read as the same machine while one
reply never changes its words for no reason. The words change when the work does and each
change rises into place; a step joins the list above when it lands, as a check on the confirm
spring. Nothing else moves: no shimmer on "Thinking", no spinner on a card.

**Decisions and their outcomes are one card that changes.** A proposal or a plan waiting on
someone is the artifact chrome in miniature — the kind's mark in a sunken well, the title
in semibold, the state as a phase badge, the buttons under a hairline. Decided, it becomes
a receipt of what came of it in place: the outcome line rises, the mark settles, and
"approved" and "done" stay two facts. A write the agent may make on its own (a report saved
to the person's own list) is recorded at the `AutoExecute` tier and reads "Done on its own",
never as approved or waiting. The note that starts the turn after a decision is drawn as a
decision between hairlines, first line only, whatever else the message carries: the lines
after it are instructions to the agent.

**Decisions are a queue with keys.** A row says who proposed what in one sentence; the
detail beside it says why and what it would change. `j`/`k` walk the rows, `x` marks one
for the batch, `a`/`r`/`m` act on the focused row, `Shift+A` approves the batch. The batch
bar is a floating, inverted surface (it is a popover in all but name), and a batch decides
one tool at a time because an approver reads the tool once. The keys are shown as `Kbd`
next to the buttons they mirror, not explained in prose.

**The composer carries context, not chrome.** What a person hands over with their words
rides as chips inside the box: a file as a `rounded-full` chip with its name and progress, a
named record as a chip with the at-sign mark, and the page as the existing pin chip, which
also says how many rows or filters it carries. A slash lists commands ahead of the agent's
questions; a command with slots is filled in the box, its empty slots shown as a hint row
under the text in `font-mono`, never as a form. An at-sign opens a listbox over the
organization's records in the same style as the command list. The dictation control is a
plain icon button until it is listening; then it is a `danger-subtle` pill with a stop square
and a three-bar level meter that moves only while someone is speaking, so a quiet room is a
still control. Where dictation cannot run the button stays, disabled, and its tooltip says why;
a refused microphone or a browser without a speech service is said under the box, never
swallowed. That line under the box says one thing at a time and never wraps — the keys to
send while typing, the slash and the at-sign on an empty box, what the microphone is doing —
and the full list is behind the keyboard button at its end.

**The launcher is a signal, and it says what it means.** At rest the corner mark
does nothing at all — it is on every page, so anything that moved would be
movement nobody can escape. When decisions are waiting it grows into a pill and
says "3 waiting". It used to say that with a beam travelling its border and a
numbered dot: two devices for one fact, one of them a loop running on every
screen for as long as anything was pending. A shape that changes is a stronger
signal than a shape that moves, and it can carry a word — a bare `3` in a
corner could be anything. The visible count caps at 99+; the accessible name
does not, because "99+ changes" is worse than the number. The count is the
attention summary's `agentDecisions`, the same number the sidebar and the Desk
show, never a second query.

### The Desk is a room

The Desk runs **outside the app shell** — hoisted out of `AppLayout` into its
own route group, so it wears no sidebar, no breadcrumb and no page header, and
takes the whole viewport. It is the one surface in Trenova allowed its own
light, and it has its own tokens for it:

```
--desk-canvas    the room
--desk-column    the conversation, lit from within
--desk-hairline  what separates the two
```

In dark the column is a step **above** the canvas rather than below it, because
a lit surface comes forward in dark and recedes in light. That inversion is why
one set of values never works for both.

The layout is a 48px strip and two columns: the conversation on the left at
`clamp(26rem, 38%, 34rem)`, the work it produced filling the rest. The
conversation is the narrower half on purpose — prose is unreadable past about
70 characters, so the extra width goes to the tables and drafts that can use
it. There is one strip of chrome at the top of the room, not one per column,
which is why the conversation's title, pin and transcript live in the shell
rather than inside the thread. Folded away with `⌘\`, the workspace gives its
width back rather than leaving a gap.

The conversation list is a switcher behind one control, not a rail. A person
picks a conversation perhaps twice an hour and then reads and writes in it for
the rest of the hour; a permanent 260px column answers a question asked twice.

Both the Desk's front page and the corner panel open onto the same `AgentAsk`
(`components/assistant/agent-ask.tsx`) rather than a directory. An empty text
field is a better first screen than a good menu. Who is being asked is one
control inside the box, `AgentPicker`: a searchable, virtualised list read from
the server a page at a time, recent agents first — never a chip per agent, which
is a wall once an organization has sixty. The questions under the box are the
agent's own `starters`, from the server, and they trade places when the agent
changes.

The front page opens with a greeting under the day's own light (`DeskGreeting`
in `routes/desk/_components/desk-greeting.tsx`). Behind the first lines sits
`ui-daylight`, two soft pools of the part of the day in the person's timezone
(`skyPhase`): dawn is `--daylight-sun` gold with rose further off, day is sky
with a little gold, dusk is rose with violet, night is indigo with violet. Each
pool is the categorical accent at `--daylight-strength` and fades to nothing
inside its own box, so there is no edge to find, and the page is clipped at the
window rather than the column so the light never stops at a line. The greeting
leads with `SkyMark`, the same sky as a letter-sized glyph in solid shapes: the
sun on the horizon, the sun high, the sun going down, a moon and a star. The
greeting is a label (500), the date beside it quieter in `--foreground-subtle`,
and the headline under both is the page's one heading (600). The page arrives
once, in reading order, a beat apart on `animate-rise` — the greeting and its
light, the headline, the line under it, the ask box, then the rest — and then
holds still. That wash is all the colour here: the agent's accent belongs to a
conversation, and the front page has not started one.

## Checking your work

```bash
pnpm lint:design      # the seven rules, with the token to use instead
node scripts/tw-probe.mjs bleed:px-3 text-warning   # does this class emit, and as what?
pnpm lint             # oxlint
pnpm typecheck        # Badge variants and status phases are typed
```
