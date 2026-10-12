# Handoff: Triage (replaces `/inbox`)

> Paste `PROMPT.md` into Claude Code. This README is the spec it points to.

## 1. What Triage is
Messages arrive from connected **sources**. Today that is email via inbound mailboxes; EDI, portals and texts come later. **Nova** reads every message and handles whatever a rule allows. Anything left over waits for a person, with the next step already worked out.

Principles:
- **A queue, not a dashboard.** Success means leaving with nothing waiting.
- **Nova acts, people decide.**
  - Everything Nova did on its own is a receipt someone can spot-check.
  - Everything left for a person carries a reason and one obvious next step.
- **Only show numbers someone can act on.** Every count opens the messages behind it.
- **Nova ≠ Desk.**
  - Nova's instructions are set by Trenova. Organizations steer Nova only through **Rules**: which messages, from whom, and what it may do.
  - **Desk** is the chat app where organizations run their own agents. "Work on it in Desk" hands a message and its open actions to one of those agents.
  - The Nova ring mark appears in only three places:
    1. the sidebar Briefing item, which spins on hover;
    2. "Drafted by Nova" on replies;
    3. Nova's card in the Briefing deck.
  - Machine suggestions elsewhere keep the existing `AssistMark`. Desk surfaces never use the Nova mark.

## 2. The prototype
`design/index.html` is React 18 run through Babel in the browser, with demo data. It is a **reference for look and behaviour, not code to port.** Do not copy its CSS (`inbox.css`) or its hand-written components.

The Tweaks panel (toolbar toggle) switches:
- **Inbox state:** live / first day / no mailbox / desk off
- **Time of day:** afternoon / end of day
- **Device:** desktop / phone
- **Rows show:** next step / preview
- **Density**, **Theme**, and **Underline what Nova read**

The "live" state adds a new message every 26 seconds.

Prototype files and what they cover:

| File | Covers |
| --- | --- |
| `briefing.jsx` | Briefing: ring, river, deck, first day, end of day |
| `list.jsx` | Sidebar, message list, Rules page |
| `reader.jsx` | Reader, action bar, sender card, link dialog, shortcuts sheet |
| `actions.jsx` | Actions and information panel |
| `extras.jsx` | Document viewer, shipment and customer peeks, Desk hand-off, connect a source, rule matches |
| `ctx.jsx` | The shared context menu |
| `data*.jsx` | Demo data and next-step logic |
| `app.jsx` | State and keys |

## 3. Fidelity
**High fidelity.** Layout, hierarchy, motion intent and copy are final.
- **Colour, radius, type and elevation** come from `client/packages/shared/src/styles/tokens.css`. Read `docs/engineering/design-system.md` first; `pnpm lint:design` must pass.
- **New token:** the prototype's "over target" amber (`#ffab2e` dark / `oklch(0.6 0.15 62)` light) becomes a token in both `:root` and `.dark`. If an existing warning/amber token already reads correctly, reuse it instead.
- **Floating surfaces** (menus, popovers, the action bar, the context menu) are fully opaque. Use the floating primitives and `ui-lift-*`; never use transparency or blur.
- **Icons:** `@trenova/shared/components/icons` only. The Rules bolt and anything else missing is drawn into the custom set.
- **Copy** goes through `useT()` and matches `i18n/glossary.json`. Labels are sentence case.

## 4. Where it lives in the repo
**Frontend (`client/apps/web`):**

| Screen | Existing files | Notes |
| --- | --- | --- |
| Route shell | `src/routes/inbox/page.tsx` | Keep the URL-held state (`folders.ts`): lane, kind, mailbox, `q`, `message`. Add `view`, `brief`, `rules`. Rename the user-facing label to Triage and keep a redirect from `/inbox`. |
| Sidebar | `_components/folder-rail.tsx`, `folders.ts`, `lanes.ts` | Lanes, plus "Waiting on them" and "Snoozed", kinds with waiting counts, saved views, sources with health, Rules. |
| Message list | `_components/message-list.tsx`, `message-row.tsx`, `day-groups.ts` | Virtualize with `src/components/virtual-rows.tsx`. Sort by due or newest, collapse threads, swipe on touch, bulk bar, presence, new-arrival entry. |
| Reader | `_components/reading-pane.tsx`, `message-trail.tsx`, `quoted-body.ts` | Thread rows, sender card, kind correction, activity trace, floating action bar. |
| Actions and information panel | `_components/desk-reading.tsx`, `next-step.ts`, `attachment-list.tsx`, `classification.ts` | `next-step.ts` grows into a list of actions per message (see `actsOf` in `actions.jsx`). |
| Triage actions | `_components/use-inbox-actions.ts`, `triage-keys.ts` | One hook serves the buttons, keys and context menus. |
| Link | `_components/link-dialog.tsx` | Unchanged behaviour. |
| Keys | `src/config/keybinds.config.ts` | `B J K / I ↵ E # R S M X L A ?` |
| Data | `src/lib/graphql/inbox.ts`, `src/lib/queries/inbox.ts`, `client/packages/graphql/src/operations/inbox/inbox.graphql` | Run `pnpm --filter @trenova/graphql codegen`. |
| Briefing, Rules, context menu, peeks, document viewer | new under `src/routes/inbox/_components/` | Context menu: build on the shared menu primitive in `@trenova/shared`. If none exists, add one there. |
| Product guide | `docs/product-guide/inbox/index.md` | Regenerate the guide (`pnpm --filter @trenova/web guide:generate`). |

**Backend (`services/tms`):**

| Area | Existing files |
| --- | --- |
| Domain | `internal/core/domain/inboundmessage/` (`message.go`, `mailbox.go`, `enums.go`, `review.go`) |
| Ports | `internal/core/ports/repositories/inboundmessage.go`, `internal/core/ports/services/inboundcontent.go` |
| Service | `internal/core/services/inboundmessageservice/` (`service.go`, `read.go`, `classify.go`, `match.go`, `process.go`, `settle.go`, `projection.go`, `attachments.go`, `mailboxes.go`, `retention.go`) |
| Repository | `internal/infrastructure/postgres/repositories/inboundmessagerepository/` (`message.go`, `mailbox.go`, `finder.go`); column helpers in `pkg/buncolgen/inboundmessage_gen.go` |
| GraphQL | `internal/api/graphql/schema/inboundmessage.graphqls`, `resolver/inboundmessageresolver/`, `loaders/inboundmessageloader.go` |
| Ingest | `internal/api/handlers/inboundhandler/`, `internal/infrastructure/inboundmail/fetcher.go` |
| Agent tools (Desk) | `internal/core/services/agenttoolservice/inbound_tools.go`, `agentquerytoolservice/inbound_tools.go` |
| Migrations | `internal/infrastructure/postgres/migrations/*inbound_messages*` and the SQLite twins |
| Seeds | `internal/infrastructure/database/seeds/development/29_inbound_message.go` |

## 5. Screens

### 5.1 Briefing (landing view)
1. **Ring.** Shows how many messages are left for you and fills as they clear. While Nova is loading, one arc sweeps the empty track, then the fill animates to its value and the number counts up. At zero it shows a check.
2. **Nova's line.** Reads like "I took care of 29 today. 8 are yours — Partner has waited longest. About 17 minutes of work." The sender name opens that message. Hold its height while loading so nothing below it jumps.
3. **River.** A timeline from 6am to now with two lanes:
   - Nova's handled messages, as dots;
   - your messages, coloured by kind, with an amber pulse when over target and a check when done.
   Hover shows a tooltip below the dot that stays inside the river; click opens the message.
4. **Deck.** One card per decision group:
   - documents to file → File all
   - drafts Nova can write → Draft all
   - need your call → Start
   - without a record → Link
   - held back → Review

   The front card lists the people, time left on each, an estimate, and one action. Done groups become green chips. Nova's draft card has its own treatment (gradient surface, travelling border, slow glow); its button is a flat violet.
5. **Footer pills.** Handled by Nova (spot-check), corrections this week, coming back later, unhealthy sources, and one rule suggestion.
6. **Background.** A soft violet glow with drifting stars (white in dark, black in light) behind the top of the page. Respect `prefers-reduced-motion`.
7. **End of day.** The deck becomes "Carrying over", with **Hand off to [on-call person]**.
8. **First day.** Add a source → Nova reads the backlog (live count by kind) → first Briefing.

### 5.2 Queue (list + reader)
- **Opening.** Opening a folder opens its first message on wide screens. An empty folder shows a quiet empty state with "Back to the briefing".
- **Rows.** Kind dot (it becomes a checkbox on hover or when selecting), sender, thread count, presence avatar, owner, then due time or "Xm left / Xh over", subject, and the next step.
- **Reader.**
  - Subject, collapsed older thread messages, and the sender name, which opens a sender card.
  - Chips: the kind (it opens a correction menu), "Not linked", owner, snooze.
  - Message body, file chips, the sent reply (with "drafted by Nova"), and the activity trace.
- **Actions panel** (right column; on narrow screens it sits above the body, and on phones it is a bottom sheet):
  - **Conflict banner**, when one applies.
  - **Actions:** a numbered checklist, each finished inline:
    - file (with page split);
    - load draft form;
    - reply composer (templates, presence warning, error and retry to an alternate address);
    - stop-times check;
    - log a shortage;
    - block the sender;
    - link.
  - **Information:** shipment and customer rows (expand or peek) and a facts grid.
  - **Work on it in Desk** (agent picker) at the bottom.
- **Floating action bar.** Shows the next open step with Do it / Have Nova do it / Review reply, then Mark handled plus "Do this on its own next time" (which creates a rule). Also Wait for reply, Ignore and handled-with-a-note.

### 5.3 Rules
A plain list of sentences: "When a **proof of delivery** arrives from **drivers.trenova.example** on **Intake**, Nova should **file it on the shipment**." Each bold blank is an inline select. A switch sits on the left; hover shows "see what it matched" and delete. Each rule has a quiet line underneath ("142 this month", with "Last: …" on hover). The add row is a greyed sentence with dashed blanks; once complete it shows how many messages from the last 30 days would have matched (and how many look off), then Turn on. No accent bars and no header block.

### 5.4 Context menus
One shared component with:
- a title line, icons, key hints on the right, and submenus that open on hover;
- placement that clamps to the viewport and flips at the edges, with long submenus scrolling;
- destructive items in red at the bottom.

It opens on right-click, or long-press on touch.

Where it appears:
- **Rows** (applies to the whole selection when several rows are selected): open, open in a new tab, handled, ignore, snooze (times and events), wait on them, reopen, assign, correct the kind, link, Desk, copy link or address, make a rule ("Nova should…").
- **Attachments:** preview, download, file on, file as, split pages, copy what Nova read, unfile.
- **Information panel:**
  - shipment or customer: open, peek, copy PRO/BOL/PO, unlink;
  - facts: copy, "This is wrong".
- **Sidebar:**
  - kinds and views: mark all handled, snooze all;
  - views: rename inline, delete;
  - sources: send a test email, settings.

## 6. New backend capabilities
Each one ships with:
- a domain type;
- the port (`core/ports/repositories` + `core/ports/services`);
- the service;
- GraphQL schema and resolver;
- Postgres and SQLite migrations, up and down;
- RLS (`SELECT trenova_rls.reconcile();`);
- tests;
- an entry in `writecoverage.yml`, or an agent tool.

1. **Rules.** Fields: kind, sender (domain or address), source and action, plus enabled, created by and counters. Evaluated in `process.go` after classify and match, so a matching rule settles the message itself (`settle.go`).
   - **Preview:** count matches over the last 30 days, and the matches whose later human outcome disagreed with the rule.
   - **Suggestions:** derived from repeated identical manual outcomes per sender and kind.
2. **Assignment.** An owner on the message, plus an "Assigned to me" lane.
3. **Snooze.** Until a timestamp, or until an event:
   - shipment delivered;
   - document of kind X attached to the shipment;
   - invoice paid.

   Events wake the message through the existing event and Temporal paths; do not poll.
4. **Waiting on them.** Set when a reply is sent. A reply on the same thread (In-Reply-To / References) or a deadline returns the message to the queue.
5. **Threads.** A thread key is derived at ingest from `Message-ID` / `In-Reply-To` / `References`. Index it.
6. **Reply.** Send from the mailbox's address through the provider. Store outbound messages, surface provider errors, and allow a retry to an alternate address. Use the idempotency key (`docs/engineering/idempotency.md`).
7. **Reply templates.** Per organization, with field placeholders resolved server-side from the message and shipment.
8. **Response targets.** Per organization and kind, with defaults (tender 2h, status request 1h, rate con 2h, POD 8h, invoice and dispute 24h, other 48h). Expose `dueAt` on the message and sort on an index.
9. **Corrections.** Re-labelling the kind, or flagging a fact as wrong, records a correction (reuse `aicorrection` and the training export rules) and adjusts per-sender classification hints. Expose a weekly count and the accuracy change.
10. **Conflict checks.** A duplicate document of the same kind on the shipment, and a tender lane/date that overlaps an existing load. Return them as a structured list on the message.
11. **Attachment split.** Page ranges become separate documents, each with its own type.
12. **Presence.** Per-message "viewing" / "writing" over the realtime service (`docs/engineering/realtime.md`, presence scopes). No polling.
13. **Source health.** Last message received per mailbox, and a test message. Expose `lastReceivedAt` and a health enum.
14. **Briefing aggregate.** A single query returning:
    - handled-by-rule counts by action, with samples;
    - per-hour arrivals split by handled-by-Nova vs left-for-a-person;
    - waiting groups by next action, with the due time on each;
    - the corrections figure;
    - unhealthy sources;
    - the top rule suggestion.

    Cached per tenant (see §7).
15. **End-of-day hand-off.** Reassign everything still waiting to the on-call person, and record it in the activity trace.

## 7. Performance requirements
**Frontend:**
- **Code splitting:** route-level lazy loading for Briefing, Rules, the document viewer, the Desk hand-off, the connect-a-source flow and the peeks. Load the reader's heavy parts (composer, load form) when they are first opened.
- **Lists:** virtualize every list longer than a screen with `virtual-rows.tsx` (rows, the thread list, rule matches, menu lists over about 30 items). Keep row components memoized with stable props, and keep selection in a store slice instead of re-rendering the list.
- **Data:** TanStack Query with normalized keys from `src/lib/queries/inbox.ts`.
  - Prefetch on route (existing `prefetch`), on row hover/focus, and the next J/K message.
  - Optimistic updates with rollback for handle, ignore, snooze, assign, correct and file, with the Undo toast driven from the rollback.
- **Realtime:** apply server-sent events (new message, status change, presence) to the cache with `setQueryData`; never refetch the world.
- **Motion:** transforms and opacity only. The river and the stars must not trigger layout. Pause them off-screen (`IntersectionObserver`) and when the tab is hidden.
- **React practices:** no derived state stored in state; `useMemo` only where profiling shows it matters; split contexts so a keystroke in the composer does not re-render the list.

**Backend:**
- **Indexes:** the queue query is `(tenant, status, due_at)` and `(tenant, status, received_at)`, with cursor pagination (`ListCursor`) and the gated `COUNT` described in `CLAUDE.md`. Add composite indexes for lanes, kinds, owner, snooze-until, thread key and mailbox.
- **No N+1:** per-row data (attachments count, matched shipment/customer, owner, presence) goes through dataloaders in `internal/api/graphql/loaders` backed by `...ByIDs` methods.
- **Counts:** `CountBreakdown` stays one aggregate. Extend it with the new lanes rather than adding queries.
- **Caching:** cache the Briefing aggregate and the rule previews in Redis per tenant, with short TTLs, and invalidate them on the relevant writes. Cache classification hints per sender.
- **Background work:** rule evaluation, conflict checks and snooze-by-event run in workers and Temporal activities, and are idempotent. They are never on the request path of the list.
- **Retention:** keep the existing sweep, and extend it for outbound replies and corrections.

## 8. Data the prototype fakes
Briefing receipts and arrivals, rule counts, accuracy, previews and samples, sender profile stats, ETAs and route progress, presence, live arrivals and reply sending. Each maps to a real capability in §6.
