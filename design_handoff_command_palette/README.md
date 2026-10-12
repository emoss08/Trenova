# Handoff: Command palette

## About the design files
`design/Command Palette.html` is a **high-fidelity HTML prototype**. Open it in a browser to see layout, copy, states and behavior. It is a design reference, not code to ship. Recreate it in `client/apps/web/src/components/command-palette/` with the existing React, Tailwind, Base UI and `cmdk` (`@trenova/shared/components/ui/command`) patterns.

- `design/logic.jsx` holds three stand-ins:
  - **The matcher** (`matchTerm`, `scoreItem`, `HL`). The rules are the spec; port them into `@trenova/shared/lib/fuzzy-score` (see "Matching and highlighting").
  - **Nova answers** (`novaAnswer`). These are canned. **Do not port.** Production uses `useAsk` and the assistant agent.
  - **The action builder** (`buildActions`). This mirrors `palette-actions.ts`; extend that file instead.
- `design/data.jsx` is fixture data: shipments, customers, workers, documents, pages, commands and attention rows. Every number in it must come from real data (global search, preview queries, `usePaletteHome`).
- `design/views.jsx` holds the rows (`Row`) and the preview pane (`PreviewPane` plus one body per kind).
- `design/palette.jsx` is the palette: input, scope tabs, list, Nova card, actions menu and footer.
- `design/app.jsx` is a stand-in app shell, so the palette has something to sit over. Prototype only.
- `design/tweaks-panel.jsx` is prototype only. It switches the theme between dark, light and system.

Fidelity: **hifi** for structure, copy and states. Visual values come from `packages/shared/src/styles/tokens.css`. `palette.css` is layered. Earlier blocks are overridden by the `/* compact */`, `/* restrained color */` and `/* v3 */` blocks lower down; **the last rule wins**. Read the values from the rendered page, not from the first block.

### Files
- `palette.jsx`: `Palette` (state, query parsing, sections, keyboard, intent runner, selection indicator, Nova timing) and `NovaCard`
- `views.jsx`: `Row`, `Tile`, `PreviewPane`, and `ShipmentPv` / `CustomerPv` / `WorkerPv` / `DocumentPv` / `PagePv` / `CommandPv` / `AttentionPv` / `AskPv` / `SourcesPv` / `TipsPv`
- `logic.jsx`: the matcher, `search`, `isQuestion`, `novaAnswer`, `tokenize`, `buildActions`
- `data.jsx`: fixtures, `SCOPES`, `SUBJECT_QUESTIONS`, `ATTENTION`

## What changed and why
1. **Highlighting was noise.** `MatchText` split on every occurrence of every term in every field. "AI" lit up *Service f**ai**lures*, *Tr**ai**lers*, *Em**ai**l logs* and the AI in each breadcrumb. Now:
   - only the characters the matcher actually used are marked;
   - only in the field that won the match;
   - as a soft fill, not an underline.
2. **Results were wrong.** Two-letter fragments matched mid-word. Now short fragments only match at a word start or on initials (see "Matching").
3. **The preview was empty for most rows.** A page showed only "Location / Address". Each kind now gets a Desk-artifact body built from data the palette can already fetch (see "Preview pane").
4. **New:**
   - inline Nova answers with linked records you can hover to preview;
   - asking about one record (⌘I);
   - a "Needs you" home section;
   - `>` command mode;
   - a ⌘K actions menu that sits over the preview instead of replacing the list;
   - an "Ask Nova" row at the end of every search, so a search never dead-ends;
   - Esc that steps back one layer at a time.

## Layout
- **Dialog:**
  - `min(1080px, 100%)` × `min(680px, 100vh − 12vh)`, 8vh from the top.
  - Rows: input (54px) · scope tabs · body · footer (38px).
  - `bg-card`, 1px `border`, `rounded-surface`, the existing `ui-lift-float` shadow.
  - Overlay is black at 32% (light) / 55% (dark) with a 3px blur.
- **Body:** grid `minmax(0,1fr) | 420px`. Below 900px the preview is 340px. Below 780px it hides.
- **No accent bars anywhere.** The selected row is a fill only. Remove the `before:bg-brand` bar from `PaletteRow`.

### Input
- **Left:** search icon (17px, `faint`).
- **Chips** (when present) sit before the text:
  - Scope chip: neutral, 24px, label and ✕.
  - Subject chip, while asking about a record: ring mark, record title and ✕.
  - "Commands" chip while the query starts with `>`.
- **Text:** 15.5px.
- **Right:** a clear ✕ (only when there's text), then an `Esc` key cap.
- **Placeholders** by context:
  - All: "Search records and pages, run a command, or ask Nova…"
  - Shipments: "PRO, BOL, customer or lane…"
  - Customers: "Name, code or city…"
  - Workers: "Name, ID or home terminal…"
  - Documents: "File name, type or record…"
  - Pages: "Jump to a page…"
  - Commands: "Run a command…"
  - Subject: "Ask Nova about {title}…"
- **`@` popover:** typing `@` followed by letters at the end of the query opens a 220–248px popover under the input. It lists matching scopes, and the aliases from `search-entity-filter.ts` still apply (`@doc`, `@load`, `@driver`). ↑↓ choose; Enter or Tab commits it as the scope chip and strips the token. Esc closes it and leaves the text.

### Scope tabs
- **Items:** All · Shipments · Customers · Workers · Documents · Pages · Commands.
- **Style:** 26px buttons, 12.5px/500. The active tab is a `surface-hover` fill.
- **Counts:** with a query, each tab shows its count in 10.5px mono `faint`. Tabs with zero matches drop to 50% opacity but stay clickable.
- **No colored type dots.** Tab and Shift+Tab cycle the scope. While asking about a record the tabs are replaced by a 1px divider.

### List
- **Section headers:** 30px tall, 10.5px mono uppercase `faint`. They show a count when it's above 1, and a right-aligned "See all N ›" that switches scope when a group is capped.
- **Rows:** min 46px (single-line rows 36px), 8px radius, gap 12px.
  - A 28px neutral tile.
  - Title (13.5px/500), then one context line (12px `subtle`, parts separated by `·`).
  - Right side: status, count, shortcut or time.
  - The selected row shows a `↵` key cap at the far right, except on commands, which show their own shortcut.
- **Selection:** a single absolutely-positioned `surface-hover` block that slides to the selected row (`transform` and `height`, 170ms, `--ease-settle`). Hover selects. Keyboard selection keeps the row in view with a 36px top margin.
- **Context line per kind** (fields marked † can be highlighted):
  - **Shipment:** customer† · origin → destination† · BOL† (BOL only when it matched). Right: shipment status label.
  - **Customer:** code† (mono) · city† · "$X open". Right: a status label only when not Active.
  - **Worker:** ID† (mono) · role† · current load (mono). Right: status label.
  - **Document:** type† · attached record† (mono) · received.
  - **Page:** trail joined with " › "† ("Workspace" for top-level pages). Right: a pin icon when pinned.
  - **Command:** description†. Right: a check when it's the current choice, plus its shortcut.
  - **Attention:** module. Right: a count chip.
  - **Ask:** "Ask Nova" over “{query}”; in subject mode, the question over "About {title}".
- **Match source:** when a row matched through a field that isn't displayed (a page keyword, a shipment's driver), the right side shows a 16px outlined tag, e.g. "Keyword **assistant**" or "Driver **Leon** Park", with the match highlighted inside it.

## Color: restrained
The palette is neutral. Color is reserved for state that needs attention.
- **Tiles:** every type uses `NEUTRAL_TILE_CLASS` (`bg-sunken text-foreground-muted ring-border-subtle`).
  - Change `PALETTE_ENTITIES[*].tileClass` and `BRAND_TILE_CLASS` usage in rows to neutral.
  - Destructive commands keep `text-danger`.
  - Workers show initials in the tile.
- **Status labels:** an outlined label (1px `border`, `muted` text) with a 5px dot.
  - The dot is `faint` by default; warning `--warn`, problem `--danger`, done `--success`.
  - Don't use `ShipmentStatusBadge`'s filled tones in the palette. Add a `variant="outline"` if needed.
- **Count chips:** `bg-sunken text-foreground-secondary`, mono.
- **Highlight:** our yellow `#ffa31a` (the existing `highlight` token) as a fill: 26% mix in dark, 32% in light. 3px radius, text stays `foreground`.
- **Nova:** the gradient ring mark and the Triage spectrum border (while answering) are the only gradients.

## Matching and highlighting
Port this into `fuzzy-score.ts` (or a palette matcher beside it). It must return **ranges** and the **winning field**, and `MatchText` becomes a range renderer.

Per typed term, best first:
1. **Prefix of the field:** score 1.
2. **Prefix of any later word:** 0.9, minus 0.002 per character of offset. Word starts are:
   - after a non-alphanumeric character;
   - a lower → upper case change;
   - a letter → digit change (so `240193` hits `S-240193`).
3. **Initials:** all characters land in order on word starts, and the field has 2+ words. Score 0.72. "bq" → **B**illing **q**ueue.
4. **Inside a word:** contiguous substring, **only for terms of 3+ characters**. Score 0.5.
5. Otherwise no match.

- **Fields and weights:**
  - Shipment: PRO 1, customer .92, BOL .85, lane .75, driver .6 (shown as a match-source tag)
  - Customer: name 1, code .95, city .6
  - Worker: name 1, ID .9, role .5, home .5
  - Document: name 1, record .8, type .75
  - Page: title 1, keywords .82 (tag), trail .62
  - Command: label 1, keywords .8 (tag), description .45
- **Scoring:** every term must match some field. The score is the mean of best term × weight, plus 0.04 when every term landed on the title. A multi-word query that is a prefix of the title scores 1.05 and highlights as one run.
- **Highlights** go only on the field each term won; ranges are merged. A trail is highlighted only when the trail won. Never highlight a term in a field it didn't win.
- **Top hit:** the best result in All when its score ≥ 0.9.
- **Section order in All:** Nova (when it's a question) · Top hit · Shipments · Customers · Workers · Documents (3 each) · Pages (4) · Commands (3) · the "Ask Nova" fallback.
  - The fallback heading is "Nova", or "Try asking instead" when nothing matched.
  - Remote record search keeps its 200ms debounce and its 2-character minimum. Local pages and commands are instant. The order never shifts under the cursor.

## Home (empty query, All)
In order:
1. **Jump back in:** the last 5 records opened, with a relative time on the right ("8m"). From `useRecentRecordsStore`.
2. **Needs you:** attention rows with a count chip. Keep the existing `attention` source; the prototype shows late shipments, unassigned for tomorrow, ready to invoice and expiring medical cards.
3. **Pinned:** the user's pinned pages.
4. **Quick actions:** New shipment, Upload a document, New conversation, Operational insights.

**A scope with an empty query** lists:
- Records scope: "Recent" (records of that type), then all records of that type.
- Pages: all pages.
- Commands: every command, grouped (Create · Nova · Go to · Notifications · Appearance · Account).

## Preview pane (Desk artifact vocabulary)
**Style:** the right column uses the Desk artifact pane styling (`desk-art-kinds.tsx`, `desk-workspace.tsx`). Background is `bg-card` with a 1px left border. The body scrolls in a `ScrollArea` with 16px/20px padding and an 18px gap; it re-keys and fades in (220ms) when the selection changes, after the existing 90ms settle. The action bar is pinned to the bottom.

**Every body has the same frame:**
- **Source line:** a mono `code` chip with the kind, then `·`-separated facts. Example: `shipment` · Live · Updated 13:05.
- **Head:** the title at 22px/500 (PROs and file names in Geist Mono) with the subtitle under it at 13px `muted`. A status label sits on the right: a `bg-sunken` pill with a 6px dot.
- **Key/value rows:** label left (`subtle`), value right, each row with a 1px bottom border.
- **Section headings:** 10.5px mono uppercase `faint`, with an optional right-aligned note.
- **Related record rows:** a 26px neutral tile, title (mono for PRO and file), a sub line, a right note and a chevron. Hovering fills the row; clicking opens the record.

**Per kind:**
- **Shipment:**
  - Route card (`bg-sunken`, 12px radius) like the Desk record artifact:
    - origin and destination with their times;
    - a dashed line, with the driven part solid in `fg`;
    - a 24px truck marker at the progress point (grows on mount);
    - a bottom strip with "N mi · N to go" on the left and the lateness and ETA on the right (warn dot when late).
  - Then the one-line note when there is one ("Receiver open until 18:00 — still inside the window").
  - Rows: Driver (opens the worker; "Unassigned" with a warn dot), Total, Tractor, Trailer.
  - Then **Latest**, a timeline of the last events. The newest has a solid `fg` dot.
  - From `shipmentPreviewQuery`. Drop the Google map from the palette preview; the route card replaces it.
- **Customer:**
  - Stat strip (`bg-sunken`, columns split by 1px lines): Open balance · Past terms · Credit used. A warn or danger dot follows a value that needs attention.
  - **Aging:** an 8px stacked bar (Current / 31–60 / 61–90 in neutral greys, 90+ `danger`) and a 4-column legend.
  - **Open loads:** related rows for the customer's active shipments.
  - Rows: Contact, Credit limit.
- **Worker:**
  - Hours-of-service strip with 3 columns (Drive · Duty window · Cycle), each with the time left and a 4px bar. The bar is `warn` when tight.
  - **On load:** a related row.
  - Rows: Home terminal, CDL, Medical card ("Expires …" with a warn dot under 30 days).
  - From the HOS/ELD source the worker page already uses.
- **Document:**
  - A 180px framed page placeholder (swap in the real first-page thumbnail from document content).
  - **Read by Nova:** the extracted fields as rows, with the confidence as the section note.
  - **Attached to:** a related row.
  - Row: Came in through.
- **Page:**
  - Subtitle is the trail joined with " / ".
  - The stat strip shows the page's live counts. Add a small `glance` per page to `route-command-data.ts`; it is optional and hidden when there's no data.
  - **Start something here:** the related create commands, with shortcuts.
  - **You were just in:** recent records of the page's type.
  - Row: Address (mono).
- **Command:** description and shortcut. A "Current" label for the active theme.
- **Attention:** the count in a stat strip, the reason line, and **Includes**, with a few of the records as related rows.
- **Ask:** the question in a `bg-sunken` block, plus one line on what happens.
- **After a Nova answer, nothing selected:** "What Nova read", the records linked in the answer.
- **Nothing to show:** tips (`@`, `>`, `?`, ⌘K).

**Action bar** (10px/12px padding, top border):
- an ink primary button with the first action and an inverted `↵` key cap;
- "Ask" with the ring mark, for records;
- a spacer;
- 30px icon buttons: Copy link (⌘L), Open in new tab (⌘↵), All actions (⌘K).

## Actions menu (⌘K or →)
- **Placement:** a 300px popover anchored bottom-right, above the footer, over the preview. Use the dark popover surface from `dropdown-menu.tsx`.
- **Contents:**
  - header: the item's tile and title, plus an `Esc` key cap;
  - a filter field, which takes focus;
  - rows of 30px each, with icon, label and shortcut.
- **Keys:** ↑↓, Enter runs, and Esc / ⌘K / ← or Backspace on an empty filter closes and refocuses the input.
- **Actions** extend `palette-actions.ts`:
  - **All records:** Open (↵) · Ask Nova about this (⌘I) · Open in new tab (⌘↵) · Copy link (⌘L) · Copy PRO / name / file name (⌥C).
  - **Shipment:** Assign a driver (only when unassigned) · Open {customer} · Copy BOL.
  - **Customer:** Open customer ledger · New shipment for {customer}.
  - **Worker:** Open compliance · Open time off · Open {current load}.
  - **Document:** Download · Open {attached record}.
  - **Page:** Open · New tab · Copy link · Pin to home / Unpin (⌘P) · its related create commands.
- **Copy, pin and mark-read** keep the palette open and confirm in the footer (`intentKeepsPaletteOpen`). Theme commands also keep it open, so you can see the change.

## Nova
**Triggers:**
- the query ends with `?`;
- or it starts with a question word ("how, what, why, which, who, when, where, is, are, did, does, do, can, could, should, will, show me, list, find me, tell me, summarize, draft, any") and has 3+ words.

Extend `ask-question.ts` with that list. Enter on the Ask row starts the answer.

**Answer card** (`AskAnswerCard`, restyled) sits at the top of the list:
- 12px radius, `bg-sunken`, 1px `border-subtle`, the Triage spectrum border animating while it works.
- Header: ring mark (spins while busy) · **Nova** · "Working…" / "Answering" / "Answered from your records" · the subject on the right. Mark-to-label gap is 6px, and the ring is optically centered on the label.
- Then the question in `subtle`.
- **Working:** the tool steps from the turn's activity (`activity.ts`), each with a spinner and then a check.
- **Answering:** streamed text (13.5px/1.6) with a caret.
  - Record references render as inline mono chips: neutral, 19–21px, `border`; hovering fills.
  - **Hovering a chip previews the record in the right pane**; clicking opens it.
  - References come from the turn's cited entities (`AssistantEntityRef`), never from parsing model text.
- **Done:** footer buttons on one line, `nowrap`: Continue in Desk (⌘↵) · Copy · Follow up.
- The ask row is hidden while its answer shows. Editing the query clears the answer.

**Asking about one record:**
- Entry: ⌘I, the Ask action, or the Ask button in the action bar.
- The input gets the subject chip, and the tabs collapse.
- With an empty query, the list offers 3 questions per type. Copy is in `SUBJECT_QUESTIONS`; keep it.
- After an answer, the remaining questions show under "Ask something else".
- Backspace on an empty input or Esc drops the subject. Keep `askSubject` in `command-palette.tsx`; the mention is passed to `ask()`.

## Keyboard
| Key | Does |
| --- | --- |
| ↑ ↓ | Move the selection (loops) |
| ↵ | Primary action: Open / Run / Ask |
| ⌘/Ctrl ↵ | Open in new tab; Continue in Desk when an answer is done |
| ⌘K or → (caret at end) | Actions menu |
| ⌘I | Ask Nova about the selected record |
| ⌘L · ⌥C · ⌘P | Copy link · Copy ID · Pin page |
| Tab / ⇧Tab | Next / previous scope |
| ⌫ on empty | Drop the subject, then the scope |
| Esc | Step back one layer: @ popover → actions → answer → subject → query → scope → close |
| `@type` / `>` | Scope by type / command mode |

**Footer:** the key-cap hints follow the selection, then a right-aligned hint:
- "End with ? to ask Nova" by default;
- "Nova will answer this" when the query is a question;
- "⌫ Stop asking about {title}" in subject mode.

Confirmations replace the hints for 1.8s with a success check and `foreground` text, e.g. "Link copied", "Pinned to home".

## Empty state
- **Heading:** "Nothing called “{q}”", plus " in {scope}" when scoped.
- **Helper line:**
  - All: "Check the spelling, try a PRO or BOL number, or ask Nova."
  - Scoped: "Try everything instead, or ask Nova — it reads records the search index doesn't, like stop notes and email."
- **Below it are real, selectable rows:**
  - Ask Nova;
  - "Search everything for “{q}”" (when scoped).

## Motion
- Dialog in: 280ms, opacity plus `translateY(-6px) scale(.985)` to 0, `--ease-settle`.
- Selection block: 170ms slide.
- Preview body: 220ms fade and rise.
- Chips: 220ms scale-in.
- Route card: the progress and truck marker grow from 0 on mount.
- Nova border: a slow spin while answering.
- Nothing else moves.

## Components to use
- `CommandDialog`, `CommandList`, `CommandItem` (keep `shouldFilter={false}`, `loop`, controlled `value`)
- `ScrollArea` for the list, the preview body and the actions list
- `Kbd`, `KbdGroup`
- `DropdownMenu` surface styles for the actions menu (it is keyboard-driven from the input, so a positioned panel inside the dialog is fine)
- `DescriptionList` restyled to the key/value rows, or a small `PreviewFields` component in `preview-frame.tsx`
- `AssistMark` for the ring
- The existing preview queries, `prefetchRecordPreview`, `usePaletteHome`, `usePaletteRemoteSearch`, `usePinnedPages` and `usePaletteIntentRunner`
