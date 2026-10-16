# Watchtower — Desk v2 handoff

Watchtower is the Desk page for operational signals: late loads, detention, expiring credentials, insurance lapses, quarantined EDI, failed agent runs, blocked invoices. You work through it **one item at a time, ordered by deadline**. Each item can be handed to an agent, snoozed, or marked done.

The rest of Desk (shell, sidebar, top bar, Decisions, conversation) is already built. This handoff covers **only the Watchtower view** (`view === "watch"`).

Reference: `design/index.html` (open it in a browser). Source: `design/watchtower.jsx` and `design/watchtower.css` (the `/* watchtower v3 */` section and below). It reuses the Decisions queue/card primitives (`dc2-*` in `decisions.css`), which already exist in the codebase as the Decisions flow. **Reuse those components; don't fork them.**

Repo targets (from `github.md`): `client/apps/web/src/routes/desk/_components/watchtower/watchtower-feed.tsx`, `item-row.tsx`, `severity.ts`. The current feed and slide-over panel layout (`.wtv`, `.wr`, `.wp` in the CSS) is the **old** design and gets replaced.

## 1. Layout
- Two-column grid: **280px queue** | **flexible main**, with a 1px `--b-sub` top border. Same layout as Decisions.
- Container breakpoints on `mainc`: at ≤900px the queue is 230px and row subtitles are hidden. At ≤640px the queue is hidden and only the card shows.

## 2. Queue (left)
- **Header:** "Watchtower" and, on the right, "`{cleared} of {total} cleared`" in mono 11px.
- **Progress bar:** 3px, `--success` fill. Width is cleared/total and animates 500ms.
- **"On it" strip** (only while any agent is working): a card chip labelled "On it", one **AgentRing** (18px) per working item (click selects that item), and "`N working`" in mono blue on the right.
- **Lanes**, grouped by time to deadline (`due` in minutes):
  - **Act now:** ≤ 240m, or overdue. Label is red.
  - **Today:** ≤ 1440m.
  - **Coming up:** more than a day out.
  - **For your information:** no deadline.
  - Empty lanes are hidden. Each lane header shows its count. Within a lane, items sort by due ascending, and no-deadline items go last.
- **Row** (uses `dc2-qi`):
  - **Icon tile (22px):** the kind glyph, tinted by severity (Critical red, Warning `--warn`, Info neutral). If an agent owns the item, the tile shows that agent's AgentTile instead.
  - **Title** on one line, ellipsised.
  - **Subtitle:** the subject (e.g. "SEED-SHP-007"). When an agent owns it: "`{Agent} is on it`", or "`{Agent} needs you`".
  - **Trailing:** the due countdown when it's under a day (mono 10.5px). Under 60m it is `.hot` (red, 500). Otherwise a blue unseen dot if the item hasn't been opened.
  - Selected row: card background with a 1px `--b` ring.
- **Empty:** "All clear".

## 3. Card (main, max-width 680, uses `dc2-card`)
Top to bottom:
1. **Deadline clock** (only if `due` exists). A 44px ring that drains over a 4h window, with 1s linear transitions. Colour by lane: now = red, today = `--warn`, later = blue. Beside it is a big mono countdown (22px/600). Below that is the label: "`{dueLabel} in`" when the deadline is in the future, or just the label when it is now or overdue. Formats: `Now`, `51m running` (overdue), `45m`, `3h 10m`, `9 days`.
2. **Kind line:** a severity dot (8px), the kind name (e.g. "Late load"), then "· 4m ago" in subtle text.
3. **Title (h1)** and **summary** (`dc2-why`).
4. **At stake:** a small uppercase label "AT STAKE" followed by neutral pills (e.g. "4 tendered loads", "$18,400 freight"). Hidden if empty.
5. **Record button:** full width with a 1px `--b` ring. Contains the kind glyph, the subject kind over the subject name, the first two facts right-aligned and ellipsised ("Appointment Today 6:00 PM · New ETA …"), and an external-link icon. Opens the record.
6. **Trail** (optional): a vertical timeline of the events that led here. The last event is in the fg colour with a red dot and a soft halo.
7. **Agent block:**
   - **Owned:** a `--sunken` panel (`--warn-sub` when the agent needs you). It shows the AgentRing (22px, progress = step/steps), the agent name, and "Step n of N · {step}" or the agent's note. When the status is `needs`, it adds a **Review & approve** button (ink, sm). Under that is a segmented step bar: done segments are `--success`, the current one is blue and breathing, and future ones are 10% fg.
   - **Unowned:** a ringed suggestion row: the suggested agent's tile, "`{Agent} can take this`", and the agent's description.
- **Card exit:** Done slides right and fades (`out-ok`). Snooze uses `out-no`. Both last 300ms. After that the next item (same index) becomes selected.
- **All clear state:** a green check disc, "All clear", and "You cleared N today. New signals show up here the moment they happen."

## 4. Action bar (`dc2-bar`)
- Left: **Prev (K)** and **Next (J)** nav buttons, then **Ask about this** (chat icon, collapses to icon-only at ≤900px). Ask opens a Desk conversation with the item attached as page context.
- Right: **Snooze `S`**, **Done `E`**, and, when no agent owns the item, a split **Hand off `H`** button (ink) with a ▲ caret. The caret opens an upward menu (290px, fully opaque) listing other agents with their tile, name and description. "Hand off" alone assigns the suggested agent.

## 5. Keyboard
`J`/`↓` next, `K`/`↑` previous, `H` hand off (only when unowned), `E` done, `S` snooze. Keys are ignored while focus is in an input, or when a modifier is held. Register them in `keybinds.config.ts`.

## 6. Behaviour
- Opening an item marks it seen.
- Countdowns tick live. Agent progress arrives over SSE; the prototype simulates it every 3s. When an agent finishes its last step, the status becomes `needs` with the note "Ready for you".
- Hand off: the agent starts at step 0 with the status `working`. The item stays in the queue, now owned by the agent.
- Done and Snooze remove the item, increment cleared, and need optimistic updates with Undo.
- The sidebar Watchtower count equals open items. The Today page's "Critical in Watchtower" count and its link land here.

## 7. Data (per item)
`id, severity (Critical|Warning|Info), kind (run|service|cred|edi|late|detention|insurance|billing), title, summary, createdAt, subject {kind, label, href}, facts[[k,v]], dueAt?, dueLabel?, impact[] , trail[], suggestedAgentId, agent? {id, status: working|needs|done, step, steps[], note?}, seen`.
Kind → icon: run=bot, service=headset, cred=clipboard, edi=file, late=truck, detention=gauge, insurance=shield, billing=receipt.
The seed data in `watchtower.jsx` (`WT_ITEMS0`, `WT_DUE`, `WT_IMPACT`, `WT_TRAIL`) shows every state. Mirror it in dev seeds.
