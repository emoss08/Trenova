# Prompt for Claude Code — Desk v2 sync to 1:1

Paste everything below the line into Claude Code, run from the root of the Trenova monorepo (`emoss08/Trenova`, branch `master`). The design reference lives in `design_handoff_desk_v2/` — copy that folder into the repo root (or anywhere Claude Code can read it) first.

---

You are implementing the **Desk v2** design for Trenova, an enterprise transportation management system. Desk is the AI agent workspace: conversations with agents, approvals, artifacts, memory, Watchtower, Decisions.

Yesterday an agent already implemented part of an earlier version of this design. Since then the design has changed a lot. **Your job is to bring the product to a 1:1 match with the current design reference in `design_handoff_desk_v2/`** — every screen, component, state, interaction, animation, copy string and edge case. Treat the reference as the source of truth; treat what's in the codebase today as a starting point that may be partially right, partially outdated, or missing.

## Read first
1. The **full spec is appended at the bottom of this prompt** (also saved as `design_handoff_desk_v2/README.md` / `design/HANDOFF_SPEC.md`). Every screen, component, state, token, and the data each needs from the backend.
2. `design_handoff_desk_v2/design/` — the HTML/JSX prototype. Open `design/index.html` in a browser to click through it (Tweaks panel toggles theme, empty states, failure scenarios, page context, uploads). `design/Desk errors.html` shows every error/limit state side by side. Read the JSX for exact behavior and the CSS for exact values.
3. The existing implementation: `client/apps/web/src/routes/desk/**`, `client/apps/web/src/components/assistant/**`, `client/apps/web/src/components/agent-identity/**`, `client/packages/shared/src/styles/tokens.css`, and the Go services that back them (agents, assistant/conversation streaming, tools, decisions/approvals, watchtower, memory, billing queue, documents/capture).

## How to work
1. **Audit before you build.** Produce a gap list: for every item in the README's "Screens / Views" and "Interactions" sections, mark it *matches*, *partial (what differs)*, or *missing*. Include backend gaps (missing endpoints, fields, events, tables, permissions). Save it as `docs/desk-v2-gap-audit.md` and keep it updated as you go.
2. **Implement every gap**, frontend and backend. If the design needs data or behavior the Go codebase doesn't have — new fields, endpoints, GraphQL types, stream events, background jobs, migrations, permissions, tool definitions — build it in Go following the repo's existing patterns (domain → repository → service → handler/resolver, migrations, tests). Don't fake backend behavior in the client.
3. **Recreate, don't copy.** The prototype is plain React + CSS for speed. Rebuild it with the codebase's real stack, components, routing, data layer and token system. Match the visuals and motion exactly; use the repo's conventions for everything else.
4. **Verify continuously.** Run type checks, lints, unit tests and Go tests as you go. Add tests for new backend logic and for non-trivial client state (queue item readiness, bulk approve, undo window, schedule parsing, memory scoping). Visually compare each screen against the prototype in both light and dark themes.
5. Finish with a short summary in `docs/desk-v2-gap-audit.md`: what changed, migrations added, new endpoints/events, anything you couldn't match and exactly why.

## You have full control
Install any packages, add services, write migrations, change schemas, add workers/queues, restructure code — whatever the design needs. Don't stop to ask permission for implementation choices; make the call a senior engineer would make and note it in the audit doc.

## Scale and performance are requirements — not excuses
Trenova runs large carriers and brokers: tens of thousands of loads, invoices and queue items per org, many concurrent users, long-running agent conversations. Everything you build must hold up at that scale:
- Server-side pagination, filtering, sorting and search for every list (queue, artifacts, memories, decisions, watchtower, conversations). Virtualize long lists and tables in the client.
- Stream agent output incrementally; never re-render the whole thread per token. Memoize markdown parsing per block.
- Bulk operations (bulk approve, assign biller, post) run as idempotent server-side jobs with per-item results, partial-failure reporting and retries — not N client calls.
- Indexes for every new query path; no N+1s; cache what's hot; push real-time updates over the existing subscription/stream mechanism instead of polling.
- The undo window must be enforced server-side (a deferred commit that can be cancelled), not just hidden in the UI.

**Do not use scale or performance as a reason to remove, simplify, or defer anything in the design.** If a feature looks expensive, find the architecture that delivers it as designed — precompute, denormalize, cache, stream, paginate, background it. Every element, state, animation and interaction in the reference ships.

## Definition of done
- Every screen and state in the reference exists in the product and matches it 1:1 in light and dark themes.
- Every interaction works end-to-end against real backend data — no mocked data in shipped code.
- Gap audit shows zero *missing* and zero *partial* items (or a precise technical reason for any that remain).
- All tests, type checks and lints pass.


## Important: Today page
`pages.jsx` contains a `TodayPage` component. It is **NOT linked and must NOT be shipped** — the Today nav item opens the existing home composer. Don't build it.

---

# FULL SPEC


## Overview
Desk is where Trenova users work with AI agents (Billing Specialist, Dispatch desk, etc.). This bundle is the full current design: the conversation, composer, approvals, artifacts (11 kinds), billing queue items, memory, Watchtower, Decisions, the agent capabilities screen, and every error/limit state. An earlier version was partly implemented; this spec is the complete current target.

## About the design files
The files in `design/` are **design references built in HTML/React/CSS** — prototypes that show intended look and behavior, not production code. Recreate them in the Trenova web app (`client/apps/web`) with its existing stack, component library, data layer and tokens, backed by the Go services. All data in the prototype is mock data that must come from real APIs.

## Fidelity
**High-fidelity.** Colors, type, spacing, radii, shadows, motion and copy are final. Match pixel-for-pixel in both light and dark themes.

## How to run the prototype
Open `design/index.html`. The Tweaks panel (toolbar toggle) controls: theme, start view, empty states (no conversations / no artifacts), memory saving mode + demo buttons, current page context, upload demos, and failure scenarios (grouped by kind). `design/Desk errors.html` is a gallery of every error/limit state.

---

## Screens / Views

### 1. Shell
- **Sidebar** (`side.jsx` → `Rail`, 232px): logo, Search (⌘K), New conversation (⌘N); nav items **Today** (opens home), **Watchtower** (count), **Decisions** (count; amber when one is pending), **Memory**; conversation groups Pinned / Today / Yesterday / Previous 7 days with status dots (working, needs approval, failed, new reply), hover actions Pin / Delete (two-step confirm), double-click to rename. Sliding active "knob" behind the selected item.
- **Footer**: avatar, name, Settings, and **Back to Trenova** button. On hover the chevron and Trenova logo swap in a continuous loop: 3.6s cycle, each slides out left / in from right (130% translate + opacity), springy `cubic-bezier(.3,1.3,.5,1)`, ~1.5s hold each. Stops on mouse-out and returns to the chevron.
- **Top bar** (`TopBar`): title per view. In a conversation: agent mark + agent name (**clickable → agent capabilities page**) / conversation title; actions: Replay, **Hand off** menu, **What this agent can do** (shield icon), Workspace toggle with artifact count and a dot (amber when a decision is pending, else new-artifact dot).

### 2. Conversation (`app.jsx`, `thread.jsx`, `markdown.jsx`)
- 3-column grid (gutter with timestamps / content max `--chat-w` 640px (560 narrow, 820 wide) / margin).
- User messages are right-aligned bubbles. The "Asked from <page>" line under user messages is **removed** (page context is still sent with the message).
- Agent replies stream word by word. Numbered footnote references link to tool steps; hovering one opens a popover with tool key, duration, result, artifact badge and a **"Why this step?"** toggle (see Interactions).
- **Markdown in agent replies** (`markdown.jsx`) — full list:
  - Blocks: ATX headings `#`–`######`, setext headings (`===` / `---`), paragraphs, bulleted lists (`-`, `*`, `+`), numbered lists (`1.` / `1)`, keeping the start number), lists nested by indenting 2+ spaces, task lists, blockquotes (can contain blocks), horizontal rules, fenced code with a language label and Copy button (shows "Copied" for 1.4s), tables (alignment from `:--`/`:-:`/`--:`, numeric columns auto right-aligned in mono tabular figures, horizontal scroll, rows fade in), display math `$$…$$` / `\[…\]` via KaTeX.
  - Inline: bold, italic, bold-italic, strikethrough, inline code, inline links (with title), reference links `[text][id]` / `[text][]` / `[id]` with definitions anywhere, `<https://…>` autolinks, inline math `$…$` / `\(…\)` (currency like `$9,835` is not treated as math; `\$` inside math), hard line breaks (newline, two trailing spaces, trailing `\`), backslash escapes.
  - Streaming: words fade in; partial code blocks render as they grow; an unclosed math block shows raw text until it closes.
  - Not supported by design: images, footnotes, bare-URL autolinking, raw HTML, indented code blocks.
  - Use a real parser (e.g. markdown-it/remark + KaTeX + sanitizer) but keep the exact styles in `desk.css` (`.md*` rules).
- Agent messages can be structured segments (bold, references, artifact badges) **or** a markdown string.
- **Message actions** on hover: Pin as chapter, Copy, Read aloud.
- **Memory recall chip** ("Used 2 memories", expandable with Forget/Undo) and **memory saved** card.
- **Hand-off card** (`threadx.jsx` → `HandoffCard`): agent tile, "Handed off to {agent}", "{time} · this conversation stays open here", "Carried over" chips (summary, N pinned facts, pinned artifacts), "Open their conversation" button.
- **Schedule card** (`SchedCard`): "Scheduled · Results will post into this conversation" + schedule row (pause/resume, delete).
- Error/limit cards: see `Desk errors.html`.

### 3. Dock (composer area)
Top to bottom:
1. **Pinned facts — "Keeping in mind"** (`FactsBar`): pin icon + label, chips (22px tall, pill, sunken bg, × to unpin), dashed "+" chip that turns into an inline input (Enter adds, Esc cancels). Facts are sent with every turn and **survive compaction**.
2. **Decision card attached to the composer** (when a change is pending): amber tinted card with rounded top corners only (16px), no gap; the composer's top corners become square while it's attached (radius transition 320ms). A faint gradient hairline marks the seam. Card rises out of the composer on enter (520ms spring; clip-path reveal from the seam + 10px translate); the info icon does one scale "ping". Contents: title + "on N items", field from (struck) → to, "reversible / can't be undone", buttons Review / Not now / **Approve ⌘↵**. Composer placeholder becomes "Reply, or ask about this change…".
3. **Undo window** (`UndoBar`) replaces the decision card after Approve: green tinted, same attached shape, 22px countdown ring (5s, 1s linear steps), "Approved {title}", "{what} in Ns", buttons **Undo** and **Do it now**. On 0 the change commits; Undo returns it to pending. **Must be enforced server-side as a cancellable deferred commit.**
4. Composer: attachments (max 5), `@` mentions, `/` slash commands (`/schedule`, `/status`, `/quote`, `/report`, `/explain` with typed slots), page-context chip, agent picker, model picker (Auto first), context meter with compaction, dictation, send. Scan from Capture.
5. Hint line ("Press ⌘↵ to approve" when pending).

### 4. Workspace / artifact pane (`artifacts.jsx`, `artifacts-data.jsx`, `artifacts-browse.jsx`, `doc.jsx`, `billitem.jsx`)
- 520–600px right pane. **Card-stack switcher**: clicking the front card fans the stack open (it no longer opens on hover); clicking a card picks it; clicking outside or Esc closes. Last entry "All N artifacts ⌘J". Up/down arrows step through artifacts. ⌘J opens the full All-artifacts browser, pressing again closes.
- Kinds: table, record, rate explanation, email draft, plan, report, run diff, document (serif reading page with citations, text-selection rewrite, edit mode, versions, export), view, decision, extraction, **billing item** (new).
- Footer: copy link (`desk/c/{conv}/a/{slug}` — slug follows the selected billing item), pin, export CSV (tables/reports), open on its own page.
- **Billing queue table** (`TableBody` when rows are `BQ-*`):
  - Checkbox column (16px boxes, ink when checked, indeterminate dash in header), row click opens that item.
  - Status pill reflects live item state (Ready for review / Approved / Posted / On hold) with an amber count badge of checks that need you.
  - **Bulk bar** (floating, bottom of pane): "N selected", "X ready · Y need you · Z already done or held", Clear, **Review Y** (opens first that needs you), **Approve X**. After approving: "Approved N invoices" + **Undo (6s countdown)**. Must be a single server-side bulk job with per-item results.

### 5. Billing queue item artifact (`billitem.jsx`, `billitem.css`)
One artifact that shows whichever queue item is selected; every queue row opens its own item.
- **Header**: mono kicker `BQ-xxxxx · INV-D-xxxxx (INV-xxxxx when posted) · PO xxxxx`, then **up/down arrows with "n of N"** at the right to step through the queue. Customer name 20px/600, bill-to line; right side total 24px mono (animates on change) and "Net N · due {date}".
- **Status line**: status pill (Ready for review = blue dot; Approved = green with check; Posted = ink with check; On hold · reason = amber with pause icon) followed by "Queued {date, time} · {N} days in queue". When on hold, a **Release** button sits at the end of this line. (There is no step/progress bar.)
- **"Before it can be approved"** (only while Ready for review): count "N of 5 need you" / "All clear". Five checks — Biller, Charges match the rate con, Proof of delivery, Bill-to and terms, Not a duplicate. Passing checks render as one compact row (label left, detail right). Failing checks expand with explanation and actions:
  - Biller missing → **Assign to me** / **Someone else…** (list of billers with initials, "you" tag).
  - Item-specific issue, e.g. lumper fee not on the rate con (Keep it · bill $110.00 / Remove it), unsigned POD (Ask the driver for it / Bill without it), detention doesn't match ELD (Bill ELD time · $32.50 / Remove detention). Each shows the contract/agent reasoning.
- **Charges** ledger: Charge (name + basis) · Rate con · Billed. Flagged line highlighted amber until settled; removed lines struck through; adjusted lines show new basis; inline undo button. Total row + "$X over/under the rate con".
- **Shipment**: lane card (truck tile, "From → To", delivered · driver · miles), shipment link, document tiles (POD, BOL, Rate con, Lumper receipt; missing/unsigned doc tinted amber).
- **Activity** timeline (system / agent / you entries; your entries have filled dots).
- **Sticky action bar**: Hold (menu: Waiting on paperwork / Customer dispute / Rate question), reason text ("Assign a biller first", "Settle the flagged check first", "Posting sends it to the customer and can't be undone"), **Approve** (disabled until clear) → **Post $X** → "Posted as INV-xxxxx · sent to {bill-to}".
- State per item persists while navigating; the table, Today counts and the item stay in sync.

### 6. Memory page (`pages.jsx` → `MemoryPage`)
Sidebar → Memory. Max width 720px.
- Header: kicker "MEMORY", "What Desk remembers", one-line explanation.
- **Saving new memories** setting: Automatically / Ask me first.
- **Add a memory** input with scope select (Just you / Billing team / Organization) and Save.
- Filters: All / Just you / Billing team / Organization with counts; search.
- List rows: memory text; meta row with editable scope select, "Saved {date} from “{source}”", "Used N× · last {when}" / "Paused" / "Not used yet"; actions on hover — Edit (inline textarea, Save/Cancel), Pause/Resume, Forget (row becomes "Forgotten. Agents won't use this again. · Undo").

### 7. Agent capabilities page (`AgentPage`)
From the agent name in the top bar or the shield icon. Max width 720px.
- Back to conversation; agent tile (lg), name, "{desc} · {model} · set up by {owner}", On/Off switch.
- **Look things up** (read tools) and **Make changes** (write tools): each row = label + mono tool key + segmented control (Allowed / Ask first / Off; read tools have no "Ask first"). Some rows are locked with a reason (e.g. Post invoices: "Always asks · posting can't be undone"). **Selected state is neutral (no green/amber)**.
- **Hands off to**: topic → agent.
- **Limits**: requests today (143/200), monthly budget ($460/$500, bar turns amber > 85%), largest single change (500 items), "Only change things during business hours" switch.
- These settings must persist and be enforced by the Go agent runtime (tool permission checks, approval gating, limits).

### 8. Hand off (`HandoffMenu`)
Top-bar button opens a 300px menu: "Hand off to · Carries over a summary, pinned facts and pinned artifacts" + agent list (tile, name, description). Picking one creates a new conversation with that agent seeded with the summary/facts/artifacts and inserts the hand-off card here.

### 9. Scheduled requests
- Create by sending a message that starts "every {weekday|day|morning|Monday…} at {time}, {request}" or via `/schedule {when} {request}`. Inserts a schedule card in the conversation.
- Each schedule: prompt, cadence + time, next run, last run, on/off; run now, pause/resume, delete. Runs post results into the originating conversation. Needs a Go scheduler/worker with timezone handling.
- `pages.jsx` also contains a **Today** summary page (`TodayPage`) — it is **not linked** in the current design (Today opens the home composer). Don't ship it unless asked.

### 10. "Why this step?"
In the footnote popover (and as "Why?" on each step in narrated mode), expands three rows: **Saw** (what data the agent looked at), **Because** (the rule/reason it chose this step), **Instead of** (the alternative it rejected). Must come from the agent runtime (store a short rationale per tool call), not hard-coded.

### 11. Watchtower, Decisions, Home, Settings, Search, Errors
Unchanged from the previous handoff in behavior; see `watchtower.jsx`, `decisions.jsx`, `side.jsx` (Home/Composer), `settings.jsx`, `search.jsx`, `errors.jsx` + `error-parts.jsx`. The Tweaks failure scenarios are grouped as: Everything works · Didn't get an answer · A step or change failed · Hit a limit · Can't use this agent · Connection.

---

## Interactions & motion (key values)
- Easing vars: `--settle`, `--spring`, `--ease` in `desk.css`. Common keyframes: `pop`, `nl`, `rise`, `mprow`.
- Decision attach: `decRise` 520ms spring; corner radius transition 320ms `--settle`.
- Undo ring: stroke-dasharray 56.5, 1s linear per tick.
- Back button loop: `sbkc`/`sbkl` 3.6s infinite.
- Bulk bar / menus: `pop` 200–260ms spring.
- Row/list entrances: 18–60ms stagger.
- Keyboard: ⌘K search, ⌘N new, ⌘J artifacts, ⌘↵ approve, Esc closes pane/menus.

## State / data the backend must provide
- Conversations, turns (structured segments or markdown), tool steps with rationale, artifacts (typed payloads, versions), pinned facts per conversation, compaction summaries.
- Decisions with deferred commit + cancel (undo window).
- Billing queue items: charges with rate-con comparison, readiness checks + issues with resolution options, documents, activity log, hold reasons, assign/approve/post/hold/release, bulk approve job.
- Memories: CRUD, scope, pause, usage counts, source, auto/ask setting.
- Agent config: tool permission modes, locks, hand-off routes, limits/usage, business-hours rule, on/off.
- Hand-offs: create seeded conversation.
- Schedules: CRUD, cadence parsing, runner, results posted to conversation.

## Design tokens
Defined in `design/desk.css`. The active palette is the neutral block near line 414 (`/* Vercel-style neutral palette */`):
- Light: canvas `#fafafa`, card `#fff`, sunken `#f2f2f2`, hover `#f2f2f2`, fg `#171717`, fg2 `#2e2e2e`, muted `#4d4d4d`, subtle `#666`, faint `#8f8f8f`, borders `#ebebeb` / `#e5e5e5`.
- Dark: canvas/card `#0a0a0a`, raised `#111`, sunken `#1a1a1a`, hover `#1f1f1f`, fg `#ededed`, fg2 `#d4d4d4`, muted/subtle `#a1a1a1`, faint `#707070`, border-sub `#1f1f1f`; sidebar `#000`.
- Semantic: `--warn`, `--warn-fg`, `--warn-sub`, `--success`, `--success-fg`, `--success-sub`, `--ink`, `--ink-fg`, `--ring`, `--lift`, `--lift-hi` (see the same block).
- Type: Geist (UI, 13px base), Geist Mono / IBM Plex Mono (`--plex`) for IDs, numbers, kickers; Newsreader for the document artifact.
- Radii: 6–8 (buttons, chips), 9–12 (cards, inputs), 14–16 (panels, composer), 999 (pills).
Map these onto `client/packages/shared/src/styles/tokens.css`.

## Assets
- `design/logo.png` — Trenova logo (sidebar + back button).
- Icons are inline SVG paths in `icons.jsx`, `artifacts.jsx` (`AX_ICONS`), `agents.jsx` (`AG_ICON`), `billitem.jsx`; use the repo's icon set where equivalents exist.
- KaTeX 0.16.11 for math.

## Files (in `design/`)
`index.html` (entry), `app.jsx` (app state, turns, approvals, undo, hand-off, schedules, tweaks), `side.jsx` (sidebar, top bar, composer, decision card), `thread.jsx` (prose, footnotes, narrated steps), `threadx.jsx` + `threadx.css` (undo bar, facts, hand-off, schedule card, step rationale, composer-attached decision), `markdown.jsx`, `artifacts*.jsx`, `doc.jsx` + `doc.css`, `billitem.jsx` + `billitem.css`, `pages.jsx` + `pages.css` (Memory, Agent, Today), `memory.jsx`, `context.jsx`, `uploads.jsx`, `mentions.jsx`, `slash.jsx`, `pagectx.jsx`, `models.jsx`, `agents.jsx`, `watchtower.jsx`, `decisions.jsx`, `search.jsx`, `settings.jsx`, `errors.jsx`, `error-parts.jsx`, `Desk errors.html`, `desk.css` and per-feature CSS.
