# Desk v2 — gap audit

Audit of the product against `design_handoff_desk_v2/` (spec: `design_handoff_desk_v2/README.md`, prototype: `design_handoff_desk_v2/design/`). Each item is marked **MATCHES**, **PARTIAL** (what differs) or **MISSING**, with backend gaps after each area. This file is kept up to date as gaps close; the **Status** table below is the running summary.

The Today summary page (`TodayPage` in `pages.jsx`) is out of scope by instruction: the Today nav item opens the home composer.

## Decisions

Calls made where the design and the codebase don't line up one-to-one:

| Topic | Decision | Why |
|---|---|---|
| Memory scope "Billing team" | A memory can be scoped to a **role** (the person's role, e.g. Billing). Scopes: Just you (user), team (role), Organization. | Trenova has no team entity; roles are how people are grouped and permissioned. |
| Who may change shared memories | Anyone on the Desk keeps their own ("Just you") memories with no further permission. Writing, moving into, editing, pausing or forgetting a team (role) or Organization memory needs Agent Memory create/update, and for a role also holding it (directly or by inheritance). Without it shared memories are read-only on the page and in the conversation. | A team or organization memory reaches other people's conversations; this keeps that an Agent Memory power, as AI Control has it. |
| Memory saving mode | "Ask me first" is a per-person preference (`agent_memory_preferences`). It does not raise a proposal: `remember` keeps the memory as a Suggested offer that nothing reads until the person accepts it in the conversation card. A memory approved through a proposal is kept as approved. A "Just you" memory runs unasked like other personal writes. | The card is the asking; a proposal on top of it would ask twice. |
| Agent capabilities page | Everyone who can use an agent can open it read-only; people with agent-definition update permission can change it. | Changing tool modes and limits is an admin power today; the page must not widen it. |
| Tool modes | Allowed = AutoExecute, Ask first = ActWithApproval, Off = tool removed. Propose maps to Ask first. Locked rows come from the tool's ceiling and irreversibility. | Reuses the existing tier model and its enforcement. |
| Undo window | Approve moves a proposal to a new `Approving` status and starts a Temporal workflow that commits after 5s unless an undo signal arrives; "Do it now" signals an immediate commit. | Enforced server-side, durable across restarts, same timer+signal pattern as the agent run workflow. |
| Bulk approve | One idempotent Temporal job per request, per-item results, 6s deferred start that Undo cancels. | Follows `BulkBillingTransferWorkflow`. |
| Schedules | Temporal Schedules per conversation schedule (`conversation-schedule/{id}`), cron + timezone, results posted as a scheduled turn in the origin conversation. | Follows `agentjobs.DefinitionSchedules`. |
| INV-D draft number | Shown as the draft invoice number when one exists; the final number replaces it at posting. | The invoice number is assigned at draft creation; the kicker shows it with the draft prefix until posted. |

## Status

| Area | State |
|---|---|
| 1. Shell, top bar, hand-off | done |
| 2. Conversation, markdown, "Why this step?" | done |
| 3. Dock: facts, attached decision, undo, schedules | done |
| 4. Workspace / artifact pane, bulk approve | done |
| 5. Billing queue item | done (billing table mode and bulk approve from §4 included) |
| 6–7. Memory page, agent capabilities page | done |
| Context meter and compaction | done |
| 11. Watchtower (previous handoff, parked on `wip/watchtower-focus`) | parked: Desk first, by instruction |

---


# Audit 1: Shell, Hand off, Home/Settings/Search, Tokens

Paths: design = `design_handoff_desk_v2/design/`, ours = `client/apps/web/src/routes/desk/_components/`, css = `routes/desk/_styles/desk-v2.css`.

## 1. Shell: Sidebar Rail

### Top
- [MATCHES] Logo, Search (title "Search  ⌘K"), New conversation "+" (title "New conversation  ⌘N", `sb-new` lift + rotate on hover). See desk-rail.tsx:126-149 and side.jsx:31-36.
- [PARTIAL] Logo asset: design uses `logo.png`, ours uses `@/assets/logo.webp` (desk-rail.tsx:1).

### Nav items
- [MATCHES] Today: home icon, opens home (desk-rail.tsx:159-167).
- [PARTIAL] Watchtower count: design always renders `<em class="sb-ct">` (side.jsx:41). Ours hides it when the count is 0 and hides the whole item without Watchtower read permission (desk-rail.tsx:168-179). Ours counts `watchtowerCounts.unresolved` (desk-layout.tsx:430).
- [PARTIAL] Decisions count plus amber: design shows `decN + pending` at all times, with `.w` set when the current thread has pending (side.jsx:42). Ours shows `attention.agentDecisions`, hides it at 0, and sets `dk-w` from `decisionsWaitHere` (desk-rail.tsx:180-193). Amber colour matches via `.dk-sb-ct.dk-w { color: var(--dsk-warn) }` (css:2544).
- [MISSING] Memory nav item (memory icon, `n:mem`), the `/desk/memory` route and the "Memory" top-bar title. `DeskPlace` has no memory (desk-rail.tsx:17), the router has only index, `t/:threadId`, `decisions` and `watchtower` (router.tsx:2004-2045), and DeskIcon has no `memory` glyph (desk-icons.tsx).

### Conversation groups
- [PARTIAL] Groups: Pinned, Today, Yesterday and Previous 7 days match. Ours adds "Previous 30 days" and "Older" (desk-threads.ts:5, desk-rail.tsx:315-330).
- [MATCHES] Status dots work/wait/error/new, their icons, tooltip labels and CSS (desk-rail.tsx:226-234, 333-345; css:2312-2363, 2529-2543).
- [PARTIAL] Tooltip: design appends `· {when}` (`c.w`). Ours has no time.
- [MISSING] Dispatch-kind dot variant `.v`. The CSS exists (css:2327) but the rail never applies `dk-v`. In the neutral palette both variants are `--faint`, so this has no visual effect.
- [MISSING] Empty state: design shows "Conversations" heading + "Your chats will show up here. Press ⌘N to start one." (side.jsx:69). Ours renders nothing, and there is no `.dk-sb-empty` CSS.
- [MATCHES] Hover Pin/Delete with `data-tip`. Two-step delete: the first click turns the button into a red "Delete" pill, mouse-leave resets it, the row slides out over 260ms, then delete is called (desk-rail.tsx:98-109, 238-276; css:3099-3194).
- [MISSING] Double-click rename: design has `onDoubleClick` → `.sb-in` input, Enter/blur commits, Esc cancels, and the top-bar title updates (side.jsx:14, 54-56). Ours has no handler and no `.dk-sb-in`/`.ed` CSS. The backend already supports it: PATCH `/assistant/threads/:id/` takes `Title` (ports/services/assistant.go:33-40), and the client has `updateThread` (services/assistant.ts:141).
- [MATCHES] Sliding knob: measured in `useLayoutEffect` and translated with a 420ms `--settle` transition (desk-rail.tsx:84-96; css:2254-2266).

### Footer
- [PARTIAL] Avatar, name and Settings are present. Ours truncates the name at `w-[100px]` (desk-rail.tsx:286-289).
- [PARTIAL] Back to Trenova button is present and navigates to `/` (desk-rail.tsx:299-307). The animation is **MISSING**. Design is `.sb-back` > `.sb-bk-c` (chevL) + `img.sb-bk-l` (logo), animated by `sbkc`/`sbkl` (3.6s `cubic-bezier(.3,1.3,.5,1)`, 130% translate, desk.css:530-536). Ours renders only a chevron and has no `sbk*` CSS.

## 1. Shell: TopBar

- [MATCHES] Place titles Today, Decisions and Watchtower (desk-topbar.tsx:68-76).
- [MISSING] "Memory" title. Also missing is the agent view title "AgentMark · Billing Specialist / What it can do" (side.jsx:79).
- [PARTIAL] Thread title: tile, agent name, "/" and title are present (ours `DeskAgentTile xs` vs design `AgentMark s=10`). The agent name is a plain span (desk-topbar.tsx:63). Design makes it a button (`ttl-ag`, "What this agent can do") that opens the capabilities page; the button and the AgentPage route are both **MISSING**.
- [MISSING] Replay button (`replay` icon, title "Replay"). In the prototype it only remounts the demo (app.jsx:455), so it may be prototype-only. The icon exists in desk-icons.tsx:62.
- [MISSING] Hand off menu button (see §8). The `handoff` icon is missing from desk-icons.tsx (design icons.jsx:39).
- [MISSING] Shield button "What this agent can do" (desk-topbar.tsx has none; the `shield` icon exists at desk-icons.tsx:75).
- [PARTIAL] Extra controls not in the design: "Download transcript" and Pin/Unpin buttons (desk-topbar.tsx:84-104), plus a mobile menu button.
- [MATCHES] Workspace toggle: panel icon, label, count `.dk-n`, and a dot when the panel is closed and there is a new artifact or a pending decision. The dot is amber (`dk-dot-wait` → `--dsk-warn`, css:11913) when pending and otherwise `--dsk-agent` with breathe at 2.4s (desk-topbar.tsx:105-113; css:140-175).

## 8. Hand off

- [MISSING] `HandoffMenu`: a button titled "Hand off to another agent" that opens a 300px `.hom-p` popover (`pop` 200ms). It has the header "Hand off to" / "Carries over a summary, pinned facts and pinned artifacts" and an agent list (tile, name, ellipsized description), and closes on an outside click (threadx.jsx:28-41, threadx.css:20-27). Not in the client. `useDeskHandoffStore` is unrelated: it moves home-composer files into a new thread (desk-layout.tsx:226-233).
- [MISSING] `HandoffCard`: "Handed off to {name}", "{time} · this conversation stays open here", "Carried over" chips (summary, N pinned facts, artifacts), and "Open their conversation ›". It animates with `nl` 320ms (threadx.jsx:43-54, threadx.css:28-37).
- [MISSING] Seeded new conversation and activity log entry "Handed off to X" (app.jsx:343).
- [PARTIAL] Related existing code: in-turn agent delegation UI (`components/assistant/delegate-step.tsx`, `delegation.ts`). This is model-initiated `delegate_task`, not a user hand-off.

### Backend gaps (hand-off)
Exists (patterns to follow):
- Thread creation path: `POST /assistant/threads/` → `startThread` (assistanthandler/handler.go:67, 272-302) → `assistantservice.StartThread` (conversation.go:41-104) → `CreateThread`. It takes only agent, title, origin and subject.
- `ThreadOrigin` enum (domain/conversation/enums.go:82-92) and `MessageKind` Message/DecisionNote/Delegated (enums.go:27-38).
- Artifact pin: `assistantartifact.Artifact.Pinned` (artifact.go:53) and `POST /threads/:id/artifacts/:artifactID/pin/` (handler.go:182).
- Hand-off routes: `AgentDefinition.DelegateIDs` (agentdefinition/definition.go:115-119), exposed as GraphQL `MyAgent.delegates` (schema/agentdefinition.graphqls:218).
- Fencing for seeded content: `agentruntime.DelegateContext` / `DelegateInput` (agentruntime/delegate.go:168-208).

Missing:
- Pinned facts per thread: no table, field or endpoint (nothing matches `PinnedFact`/`pinned_fact`).
- Conversation or compaction summary: nothing in `domain/conversation`.
- A hand-off record. Options are a table such as `assistant_thread_handoffs` (source/target thread, target agent, user, summary, facts JSONB, artifact_ids, created_at) or `MessageKind("Handoff")` in the origin thread so the card renders from history.
- Thread linkage: `ThreadOriginHandoff` + `HandedFromThreadID` on `Thread` (thread.go:25-80), plus a migration.
- Endpoint `POST /assistant/threads/:threadID/handoff/` {agentDefinitionId}. It should reuse StartThread, seed summary, facts and pinned artifacts for the first turn, and return both ids. The client also needs `handoffThread`.
- Permission check that the target is in `DelegateIDs` or `myAgents`.

### Backend: sidebar counts and status dots
- [MATCHES] The thread list API supplies the dots. `ListThreads` → `markAttention` (assistantservice/conversation.go:135-205) sets `attention {pendingDecisions, lastTurnFailed, unread}` (domain/conversation/attention.go) from `ListThreadAttention` (conversationrepository/attention.go:76). "Working" comes from `GET /assistant/turns/active/` via `useLiveThreadIds` (desk-thread-state.ts). Read marks go through `POST /threads/:id/read/`. **No backend gap.**
- [MATCHES] Decisions count: GraphQL `attentionSummary.agentDecisions` (schema/attention.graphqls).
- [MATCHES] Watchtower count: `watchtowerCounts.unresolved` (schema/watchtower.graphqls:99-130).

## 11. Home / Settings / Search

### Home (side.jsx `Home`)
- [MATCHES] Date, greeting "Good morning, {first}" (ours also has afternoon/evening), headline, TermsNote, a home composer with the placeholder "Ask the Desk anything…", and typewriter presets with "Tab to use · ⌘{n} to ask" (desk-home.tsx:46-161, desk-composer.tsx:300-322).
- [PARTIAL] Headline comes from the briefing API, with fallbacks "Nothing is running here yet." and "How can I help today?". Presets come from agent `starters` rather than the fixed list.

### Settings (settings.jsx)
- [MATCHES] Six sections, "Reset to defaults", Esc and 160ms close. Every other row's copy and options match (desk-settings.tsx:36-534). Settings persist under `trenova-desk-settings` (stores/desk-settings-store.ts:80).
- [PARTIAL] Copy drift:
  - Theme hint: design "…pick light or dark for Desk.", ours "…pick light or dark." (desk-settings.tsx:216).
  - Share row: design "Share the page you're on" / "Send the current page with each message so the agent knows what you're looking at.", ours "Share the page you came from" / "…the page you came to Desk from…" (desk-settings.tsx:338-339).
- [PARTIAL] Option sources: the agent select adds "The last one I used". Scan device and profile come from real data, with "Ask each time" and "Organization default" as the empty options.

### Search (search.jsx)
- [MATCHES] Placeholder, Clear/Esc, the five filters with Tab cycling, Recent searches chips, group labels, the "Nothing matches" empty state, and the footer hints and count (desk-search.tsx:26-420). It is backed by `GET /assistant/search/`.
- [PARTIAL] Recent searches are stored per user rather than fixed. Ours adds a "Nothing here yet" empty state, "1 result" singular, and a 160ms debounce.

## Design tokens (desk.css:414-415 vs tokens.css:337-389 light, 743-789 dark)
- [MATCHES] Every light and dark neutral (canvas, card, raised, sunken, hover, fg, fg2, muted, subtle, faint, the three borders, ink, ink-fg) converts to the same OKLCH. Sidebar is #fff light and #000 dark, mapped via `--dsk-rail`.
- [MATCHES] Blue, blue-sub and blue-fg in both themes, plus `--dsk-error` = #e5484d.
- [PARTIAL] Light `--warn`: design #f5a623 = oklch(0.784 0.159 73), ours `--warning` 0.817 0.164 76, which is the dark #ffb224 value.
- [PARTIAL] Light `--warn-sub`: design #fff4cf = 0.966 0.049 93, ours 0.976 0.024 83.
- [PARTIAL] Light `--success`: design #28a948 = 0.646 0.175 147, ours 0.535 0.127 147.
- [PARTIAL] Dark `--success`: design #00ca50 = 0.732 0.213 148, ours 0.645 0.145 147.
- [PARTIAL] Dark `--warn-fg`: design #f1a10d = 0.769 0.162 73, ours 0.772 0.165 65. This is close.
- [MATCHES] The other semantic tokens.
- [PARTIAL] `--lift`: design light is `0 0 0 1px rgba(0,0,0,.08), 0 2px 2px rgba(0,0,0,.04)`; ours is `--lift-whisper` = 1px ring at fg 8% + `0 1px 2px -1px oklch(.2 0 0/.08)`. Design dark is a 1px white ring at .1; ours is `--lift-ring` at fg 12%.
- [PARTIAL] `--lift-hi`: design light is `…0 4px 8px -4px .06, 0 16px 24px -8px .08`; ours `--lift-float` is `0 4px 8px -4px .12, 0 16px 40px -16px .18`.
- [PARTIAL] `--ring`: design hsla(212,100%,48%,.24), ours `--ring` at hue 258 mixed to 24%. This is close.
- [MATCHES] Dark-mode hook: design `.dsk.dk`, ours the global `.dark` (desk-v2.css:2501-2515). Glow is 0% in both.

---

# Audit 2: Conversation (spec §2) and "Why this step?" (spec §10)

Paths are relative to `client/apps/web/src/` unless they start with `services/` or `design/`.

## Layout and user messages
- [MATCHES] 3-column grid: the gutter, content and margin columns are 168/640/168. `--dk-chat-w` is 640, 560 (narrow) or 820 (wide), and the `-open` variants are 600/520/680. Both apply via `dk-w-${width}` (routes/desk/_styles/desk-v2.css:313, 3499-3519; desk-settings.tsx:525).
- [MISSING] Right-aligned user bubbles. `.dk-q` is a left-aligned 20/18px heading, and only `.dk-long` gets a sunken box (desk-v2.css:379, 2798-2818). The prototype CSS has no right alignment either (design/desk.css:58, 480-482). The spec text is the only source, so the bubble style has to be designed from it.
- [MISSING] Removing the "Asked from <page>" line. `DeskQuestion` still renders `<DeskPageSent>` (conversation/desk-turns.tsx:105), which prints `t("Asked from") <b>{title}</b>` (composer/desk-page-chip.tsx:160-173). It is passed `page=` at desk-conversation.tsx:741 and :942. The prototype's `PageSent` returns null (design/pagectx.jsx:72).
- [MATCHES] Question size classes (mid above 48 characters, long above 90) and the `qin` entrance animation (desk-turns.tsx:57).

## Streaming and structured segments
- [MATCHES] Words fade in: `.dk-prose .dk-w` uses 420ms `--dk-ease`, the same as `.prose .w` (desk-v2.css:420; design/desk.css:95).
- [PARTIAL] Structured segments versus markdown. Ours only takes a markdown string. Inline artifact badges come through `artifact:` links (desk-turns.tsx:124, ai-markdown.tsx:143), not segments. There is no `badge 520ms spring` entrance on an inline badge while it streams (design/desk.css:396).
- [PARTIAL] Footnote references. The prototype puts an agent-authored ref on the last word of a span (design/thread.jsx:27-35). Ours guesses citations on the client by matching tool-result values against the reply text (conversation/citations.ts, `citeSteps`), so the numbers are heuristic. Ours adds ranges and multi-step popovers that list up to 4 steps (desk-citations.tsx:176), which the prototype doesn't have.

## Footnote popover
- [MATCHES] Number, tool key and duration header; done line; "v → r" query row; artifact badge; width 250; 200ms `fnp` animation (desk-citations.tsx:56-82; desk-v2.css:2075-2092).
- [PARTIAL] The header shows the raw tool name (`citation.step.name`), but the prototype shows a dotted key such as `billing.queue.list`. Duration is derived from message `createdAt` seconds for saved turns (components/assistant/activity.ts:151), so it is coarse.
- [MISSING] The "Why this step?" / "Hide reasoning" toggle (`.fnp-why`) and the `StepWhy` rows (design/thread.jsx:56-57, threadx.jsx:76-84, threadx.css:46-52). `StepSummary` has no toggle or state (desk-citations.tsx:41-83), and `ToolStep` has no rationale field (activity.ts:17-38).

## Narrated mode
- [MISSING] The step list ("Worked through N steps in X", `.nsum`/`.nexp`/`.ni`) with a per-step "Why?" button (`.ni-why`, opacity 0 that shows on hover) and the `.ni-wx` rows (design/thread.jsx:98-135). The Desk conversation renders no step list (desk-conversation.tsx:729-934). The only "Source numbers" setting is on/off (desk-settings.tsx:290-302), and there is no ledger/narrated/receipt mode. A "Worked through" string exists only in the generic assistant (activity.ts:890).

## Message actions
- [MATCHES] Pin as chapter (shows "Chapter N" when on), Copy (tick, "Copied" tooltip for 1400ms), Read aloud (4-bar eq with 120ms delays, "Stop reading"), and `stay` while active (conversation/desk-message-actions.tsx). Ours really copies and speaks; the prototype only stubs them. Read aloud is hidden when speechSynthesis is missing.

## Markdown renderer (ai-markdown.tsx: remark-gfm + remark-math + remark-breaks + KaTeX)
- [MATCHES] ATX and setext headings, lists with start numbers, nesting, task lists, blockquotes, hr, strikethrough, inline links with title, reference links `[t][id]`/`[t][]`/`[id]` with definitions anywhere (CommonMark), `<https://…>` autolinks, hard breaks (newline via remark-breaks, two spaces, `\`), backslash escapes, raw HTML dropped, display math `$$` and `\[…\]` (converted in lib/markdown-prepare.ts:65-87), and currency `$9,835` escaped (markdown-prepare.ts:78).
- [PARTIAL] Reference definitions while streaming. `StreamingAiMarkdown` parses each top-level block separately (ai-markdown.tsx:368-374), so `[x][id]` with its definition in a later block stays literal until the reply is saved.
- [PARTIAL] `\$` inside math. The prototype's inline-math regex explicitly allows `\$` (design/markdown.jsx:131). Ours relies on remark-math, and MONEY skips a `$` preceded by `\`. This is not verified; it needs a test.
- [PARTIAL] Unclosed math while streaming. The prototype shows `.md-math-raw` (a mono, subtle `pre`). Ours escapes the dangling `$$`, so it renders as plain prose (markdown-prepare.ts:81-89).
- [PARTIAL] Unsupported-by-design items that ours supports (all via ai-markdown.tsx:325):
  - Bare-URL and `www.` autolinks: remark-gfm autolinks them.
  - Footnotes `[^1]`: remark-gfm renders them.
  - Indented code blocks: CommonMark default, not disabled.
  - Images: rendered as a link labelled with the alt text instead of literal text (ai-markdown.tsx:173-180, 232).
  - Fix: use `remarkGfm` with `{singleTilde:false}` plus custom micromark extensions, or `disable` constructs (`codeIndented`, `gfmFootnote*`, `gfmAutolinkLiteral`) through `micromarkExtensions`.
- [PARTIAL] Code block:
  - The label is empty when there's no language; the design shows "text" (ai-markdown.tsx:99).
  - Copy shows "Copied" for **1500ms**; the design is **1400ms** (ai-markdown.tsx:77).
  - Ours has no check/copy icons and no `.ok` success colour.
  - Header is 34px with padding 0 8 0 14 and 11.5px text; the design is 32px, 0 6 0 12, 11px.
  - `pre` padding is 14/16; the design is 12/14. Radius is 12; the design is 10 (desk-v2.css:12635-12679).
  - Ours adds Shiki highlighting for json/js/graphql/sql, which isn't in the design.
- [PARTIAL] Headings:
  - The design maps level n to `h(n+2)`, with sizes 1.3/1.13/1/.8em (h4 uppercase, subtle) /.93/.86em.
  - Ours renders `#` and `##` both as h3 at 18px, `###` as h4 at 16px, and h5/h6 at 14px. There's no h4 uppercase style (ai-markdown.tsx:192-205; desk-v2.css:12495-12520).
- [MISSING] Table rows fade in. The design has `tbody tr{animation:pop 260ms var(--settle)}` (design/desk.css:86); ours only has a background transition (desk-v2.css:12484).
- [PARTIAL] Tables:
  - Numeric columns right-align and use mono (rehype-numeric-columns.ts; desk-v2.css:12698-12706). Our FIGURE regex accepts units (k, h, mi, lb…) that the design's `MD_NUM` doesn't.
  - Font is 12.5px; the design is .86em.
  - Ours makes the first column bold and `--fg`, which the design doesn't (desk-v2.css:12480).
  - The container is a radius-12 inset shadow; the design is a radius-10 border on `--card`.
- [PARTIAL] Display math box: padding 18/16 and radius 12 (desk-v2.css:12684), against 14/16 and radius 10 (design/desk.css:68).

## Memory
- [MISSING] The recall chip "Used N memories" (`.mrc`): expandable, each item shows text, scope, "Saved {when} from "{src}"", Forget becomes "Forgotten. Desk won't use this again." with Undo, and a chevron that rotates 90° (design/memory.jsx:39-67, memory.css:9-23). Nothing in `routes/desk` or `components/assistant` renders it. `recall_memory`/`remember` are only excluded from citations (conversation/citations.ts:17-18).
- [MISSING] The memory saved card (`MemSave`) in all its states:
  - saved: "Saved to memory", scope, Edit, Undo
  - edit
  - ask: "Remember this for next time?", Just you / Billing team, "Don't save" / "Save memory"
  - removed / declined, each with Undo
  - The `memin` 420ms spring animation (design/memory.jsx:69-94).

## Hand-off and schedule cards
- [MISSING] `HandoffCard`: agent tile, "Handed off to {agent}", "{time} · this conversation stays open here", "Carried over" chips, and "Open their conversation" (design/threadx.jsx:52-63). `useDeskHandoffStore` in desk-conversation.tsx:168 is a composer-seed hand-off, not this card.
- [MATCHES] `SchedCard`: `DeskScheduleCard` (conversation/desk-schedule-card.tsx), drawn from the thread's `Schedule` message with the schedule read from `GET /assistant/threads/:id/schedules/`.

## Backend gaps

### Per-tool-call rationale (Saw / Because / Instead of)
- What exists:
  - `conversation.ToolCallRecord` has only {ID, Name, Arguments, Effect, ProviderData, ProviderID} (services/tms/internal/core/domain/conversation/message.go:163-174).
  - The tool-role `Message` stores `ToolSummary`, `ToolVerdict`, `ToolFailed` and `FoundTools` (message.go:62-69).
  - `Reasoning *ReasoningTrace` is per assistant message, not per call (message.go:91).
  - `AssistantToolStartedEvent` has {CallID, Name, Arguments, Effect} (services/tms/internal/core/ports/services/assistant.go:502-511).
  - `agent.AgentRunStep` has no reason field (domain/agent/agentrunstep.go).
  - The only `rationale` on the client is on proposals (types/assistant.ts:1094).
- Needed:
  1. Prompt/schema: add a reserved optional `_why {saw, because, insteadOf}` (each a short string) to every tool spec. Do this in `specFor` and the toolset builder (services/tms/internal/core/services/agentruntime/toolset.go:707-723, fingerprint.go:100-110), and add a prompt instruction in services/tms/internal/core/domain/agentdefinition/prompt.go.
  2. Strip and validate before dispatch so tools never see it, and clamp each string to about 160 characters. This goes in agentruntime/arguments.go and dispatch.go:113/139, which validate against `ParamSchema()` and would reject the unknown key.
  3. Store it as `ToolCallRecord.Rationale *ToolRationale` (JSONB is already the column type, so no migration is needed). Optionally mirror it on `AgentRunStep` (domain/agent/agentrunstep.go, services/runstepledger/service.go).
  4. Stream it: add `Rationale` to `AssistantToolStartedEvent` and set it in agentruntime/runner.go:242-251. `AssistantMessageEvent.ToolCalls` (runner.go:231-239) carries it automatically.
  5. Client: `toolCallRecordSchema` (types/assistant.ts:439), the tool_started schema (types/assistant.ts:1326), `ToolStep.rationale` (components/assistant/activity.ts:17), the popover toggle (conversation/desk-citations.tsx), and the narrated step list.
- Also needed: a tool key in display form for the popover header, either mapped on the client or added as a dotted key to the tool spec.

### Memory recall and saved events
- What exists:
  - `memoriesForPrompt` calls `RecordUse` (bump `UseCount`/`LastUsedAt`) but does not record which turn used which memories (agentruntime/memory.go:13-42).
  - The `recall_memory` query tool returns results only as tool content (agentquerytoolservice/memory_tools.go:113).
  - `remember` discards the created memory: `_, err = t.memories.Remember(...)` (agenttoolservice/memory_tools.go:121-130). Its default tier is `TierActWithApproval` (:91), which is effectively the "ask" state.
  - `forget_memory` exists (:172-237).
- Needed:
  - Add `UsedMemoryIDs []pulid.ID` (JSONB) on the assistant `Message` (conversation/message.go), filled from `memoriesForPrompt` plus `recall_memory` hits, and return the memory text, scope, created-at and source when the thread is served.
  - Add a `memory_used` stream event (ports/services/assistant.go:244-288) emitted from runner.go.
  - Have `remember` return the memory, then emit `memory_saved` {id, content, scope, kind} and keep the memory ID on the tool message so Edit and Undo can address it.
  - Add a forget/unforget (restore) endpoint for Undo. Today forget means retire; an un-retire is needed, which goes in the AgentMemoryService port.
  - Scope "Just you / Billing team" needs mapping to `Memory.Scope` (domain/agent/agentmemory.go:241).

### Other
- Hand-off cards: there is no conversation-level hand-off record. `MessageKind` covers only delegate steps (message.go:36-49). A hand-off message kind with target agent, new thread ID and carried items is needed.
- Schedule cards: there is no schedule entity or runner tied to a thread. `continuation.go:14` only mentions schedules.
- Structured segments: not needed if markdown stays the format, but agent-authored footnote refs would need a citation marker convention in the prompt (prompt.go) instead of client-side guessing.

---

# Audit 3 — Dock (spec §3) + Scheduled requests (§9)

Paths: P = design_handoff_desk_v2/design, W = client/apps/web/src, D = W/routes/desk/_components, T = services/tms/internal.

## 1. Pinned facts — "Keeping in mind" (P/threadx.jsx:15-27, threadx.css .fx*)
- [MISSING] FactsBar component — the dock (D/desk-conversation.tsx:1068-1210) renders TermsNote, the approved card, the approval card, the pending pill, the context-full card, the usage meter, the composer and the hint. It has no facts row and no pin-icon "Keeping in mind" label.
- [MISSING] Chips: 22px pill, sunken bg, inset b-sub ring, `pop` 220ms, × unpin with 16px round hover target.
- [MISSING] Dashed "+" chip that reads "Pin a fact" when the list is empty. Click opens a 190px inline input with placeholder "e.g. Invoice date is Oct 3". Enter or blur adds it and skips duplicates; Esc cancels.
- [MISSING] Tooltip "Agents keep these in mind for the whole conversation, even after it's compacted". Also missing: `.has-dec>.fx{margin-bottom:8px}`.
- [MISSING] Sending facts with each turn: the client has no facts state. The hand-off card's "N pinned facts" is not implemented either.

## 2. Decision card attached to composer (P/side.jsx:197-222, threadx.css "decision merged into composer")
- [PARTIAL] Placement/shape — `DeskApprovalCard` renders as a separate sibling above the composer, not attached to it. Differences:
  - `.dk-dock .dk-c` keeps `gap:10px` (desk-v2.css:817-821). The prototype uses `gap:0`.
  - The card radius is 12px on all corners (desk-v2.css:1189-1199). The prototype uses `16px 16px 0 0` with `margin-bottom:-1px`.
  - There is no `has-dec` class anywhere.
- [MISSING] Composer top corners squaring while attached. `.dk-cmp` has `border-radius:16px` and only `transition: box-shadow 200ms` (desk-v2.css:~833-835). There is no 320ms `--settle` radius transition, and the composer gets no deeper attached shadow.
- [MISSING] Gradient hairline seam (`::after` linear-gradient at opacity .18).
- [PARTIAL] Entrance — uses `dk-rise 420ms var(--dk-spring)` (desk-v2.css:1199) instead of `decRise 520ms` (clip-path `inset(100% 0 0 0 round 16px 16px 0 0)` reveal from the seam plus translateY 10px). No `decRise` keyframes exist.
- [MISSING] Info-icon `decPing` (900ms, 260ms delay, scale .6 → 1.18 → 1). `.dk-dcx-i` is only coloured warn (desk-v2.css:1201, 2545).
- [PARTIAL] Tint — amber `--dsk-warn-sub` bg with a 35% warn ring (desk-v2.css:2613-2617, 5023). The prototype's attached card uses an inset 32% ring and no outer drop shadow; the implementation's line 2613 adds an outer warn glow.
- [MATCHES] Contents — info icon, `title` + "on {N items}", field `<s>before</s> → <em>after</em>`, "· reversible / can't be undone" (D/conversation/desk-approval-card.tsx:385-408).
- [MATCHES] Buttons Review / Not now / Approve with ⌘↵ kbd (desk-approval-card.tsx:410-422). ⌘↵ is wired via approveRef (desk-conversation.tsx:360-362).
- [MATCHES, additive] Stale ("Redraft with N") and would-refuse variants (desk-approval-card.tsx:306-375). These match the prototype's `ec-stale` dock card (P/app.jsx:430).
- [MISSING] Composer placeholder "Reply, or ask about this change…". The conversation passes only the refusal placeholder (desk-conversation.tsx:1166-1168), so the composer falls back to "Reply to {agent}…" (D/composer/desk-composer.tsx:296-300). Note: the prototype Composer also ignores the passed placeholder outside `home` (P/side.jsx:175), but the spec requires it.

## 3. Undo window (P/threadx.jsx:2-13, threadx.css .udo*, P/app.jsx:315-342)
- [MISSING] UndoBar — Approve fires `decideMyProposal` / `decideMyPlan` / `decideBatch` immediately (desk-approval-card.tsx:88-96, 161-167, 228-229). `onApproved` then shows `DeskApprovedCard` ("Approved" + "{title} on N…" + shine) for `APPROVED_HOLD_MS = 1100` (desk-conversation.tsx:87, 320-341; desk-approval-card.tsx:428-455). This corresponds to the prototype's post-commit `dcx.ok`, not the undo bar.
- [MISSING] Green attached bar; the 22px ring (r=9, stroke 2.2, rotated -90, `stroke-dasharray:56.5`, offset `56.5*(1-n/5)`, `transition: stroke-dashoffset 1s linear`, number inside); a 5→0 tick every 1s.
- [MISSING] Copy "Approved {title lowercased}" / "{what} in {n}s"; **Undo** (undo icon) returns the change to pending; ghost **Do it now** commits immediately; commit at 0.
- [PARTIAL] `desk-countdown.tsx` has a ring countdown used for retry/rate-limit. It could be reused, but it is not the 56.5 undo ring.

## 4. Composer
- [MATCHES] Attachments max 5 (D/composer/desk-attachments.ts:6, 158-162). Full-state title "Up to 5 files per message" (desk-composer.tsx:335).
- [MATCHES] `@` mentions with picker and mirror (desk-composer.tsx:20; desk-mentions.tsx:167, 458).
- [PARTIAL] Slash commands with typed slots — status / quote / report / explain with identical slots and templates (W/components/assistant/composer-commands.ts:26-50; D/composer/desk-slash.tsx). `/schedule` is now listed too, with a "when" slot that takes the whole cadence (`splitScheduleSlots`).
- [MATCHES] Page-context chip with Explain (desk-conversation.tsx:1176-1181; desk-page-chip.tsx:64-75).
- [MATCHES] Agent picker (desk-composer.tsx:16) and model picker with Auto first and default (desk-model-picker.tsx:92-147).
- [MISSING] Context meter + compaction. No ContextMeter, CtxDrain, "Context is nearly full · compacting" / "Compacting the conversation…" status, Cancel, or "You can reply once compacting finishes" placeholder. The only related UI is the hard-stop "This conversation is too long for {model}" card (desk-conversation.tsx:1100-1118). `DeskUsageMeter` is a spend budget, not context (desk-locks.tsx:250).
- [MATCHES] Dictation (desk-composer.tsx:386), send/stop (391-416), and Scan from Capture (`onScan` → `DeskCapturePanel`, desk-composer.tsx:358-362; desk-capture.tsx).
- [MATCHES] Hint line: "Press ⌘↵ to approve" when the card shows, otherwise the "Desk can make mistakes…" disclaimer (desk-conversation.tsx:1196-1207).

## 5. Scheduled requests (§9; P/app.jsx:307, P/pages.jsx:2-31, SchedCard threadx.jsx:58-63)
- [MATCHES] Detection on send (`isScheduleRequest`, composer-commands.ts) routes the message to `POST /assistant/threads/:id/schedules/`; the server parses it (`conversationschedule.ParseRequest`). A message that opens with "every" but no cadence the parser reads ("every time…") stays an ordinary question. The schedule is not answered at once: the prototype only inserts the card, and the first answer comes on the first slot (or `POST /assistant/schedules/:id/run/`).
- [MATCHES] SchedCard with SchedRow, pause/resume, delete and "Schedule deleted". Run now and the per-person list exist in the API; the prototype shows Run now only on the unshipped Today page, so the card has no button for it.

## BACKEND GAPS (services/tms)

**(a) Pinned facts — none exist.** `conversation.Thread` has only a boolean `Pinned` for the rail (T/core/domain/conversation/thread.go:54-59). There is no facts column, table or route.
- **Storage:** follow `PreferredProviderID` / `Taint` on Thread (thread.go:52, 73). Add `pinned_facts JSONB` (an ordered `[]string`, capped). Use a `thread_facts` table if you need per-fact audit.
- **API:** extend `PATCH /assistant/threads/:threadID/` → `updateThread` (T/api/handlers/assistanthandler/handler.go:92-96; service `UpdateThread` T/core/services/assistantservice/conversation.go:320). Ownership is already the authorization there.
- **Injection:** `prepareTurn` reads the thread (assistantservice/turn.go:145-152) and builds `TurnRequest`. Thread facts through `RuntimeContextRequest` into a new `agentdefinition.RuntimeContext.Facts` (T/core/domain/agentdefinition/prompt.go:110-123, filled in T/core/services/agentruntime/context.go:63-76). Render a fenced block in `BuildSystemPrompt` next to `describePage` / `describeMentions` (prompt.go:438-447, 650, 776).
- **Compaction:** there is no conversation-level compaction or summary. The only shortening is replay trimming: `historyLimit = 120` newest messages (assistantservice/conversation.go:22-28), and older tool results cut to 320 chars outside the last 3 turns / 48k budget (agentruntime/messages.go:53-66, 245-275). Facts in the system prompt survive both by construction. The spec's compaction (meter, manual/auto compact, stored "compaction summaries") is an additional backend gap: you need a summary turn/message type plus a replay that starts from the latest summary in `replayHistory` (messages.go:83).

**(b) Undo window — approval executes synchronously.**
- **Current path:** `decideMyProposal` / `decideMyProposals` / `decideMyPlan` (T/api/graphql/schema/agent.graphqls:727-743; resolver agentresolver/agent.resolvers.go:213, 235) call `agentdecisionservice.DecideWithOutcome` (agentdecisionservice/service.go:187-358). In one request it:
  1. creates `AgentDecision`;
  2. moves the proposal Pending → Accepted with a `FromStatus` guard (288-299);
  3. signals the run workflow (`agent-decision`, 538-560);
  4. runs `executor.Run` in `executeIfApproved` (464-490);
  5. records trust and correction, audits `OpApprove` (critical), announces, and starts the follow-up turn (`assistantfollowupservice.FollowUp`).
- **Making it deferred:** split at step 2.
  - New `ProposalStatus("Approving")` in T/core/domain/agent/enums.go:209-226.
  - On `agent_decisions`, add `commits_at BIGINT`, `committed_at`, `undone_at`, `undone_by_user_id`.
  - Approve records the decision, sets Pending → Approving (same guarded `UpdateStatus`), and starts `ProposalCommitWorkflow` with ID `proposal-commit/{proposalID}`. That workflow waits on `workflow.NewTimer(5s)` in a selector with signals `decision-undo` and `decision-commit-now`, copying the timer+signal selector at T/core/temporaljobs/agentjobs/workflow.go:627-639.
  - On timer or commit-now, an activity runs today's steps 3-5. That includes re-checking `ExpectedTargetVersion` / preview, because the record may move in 5s.
  - On undo, the decision is marked undone and the proposal goes Approving → Pending (guarded), followed by `announce`.
- **Endpoints:** add mutations `undoMyDecision(proposalId|planId)` and `commitMyDecisionNow(...)`. Undo must fail with a conflict once committed. Return `commitsAt` on the decision so the client ring is server-anchored.
- **Plans and batches:** apply the same to `decideMyPlan` (agentplanservice) and to the batch path (one workflow per batch).
- **Status handling:** `ExpireStaleProposalsWorkflow` (agentjobs/workflow.go:189) and the decision queues must treat Approving as neither pending nor expired.
- **Audit:** move the `OpApprove` audit to commit and log undo separately.
- **Realtime:** add events for approving, committed and undone through the existing `announce` (service.go:600).

**(c) Schedules — done** (`conversationschedule`, `conversationscheduleservice`, `conversationschedulejobs`; migration 20261231007460). Run now starts the turn directly rather than through the schedule's Trigger, so the click gets the turn or the reason there is none.
- **Runner pattern:** `agentjobs.DefinitionSchedules` keeps one Temporal Schedule per agent, with `ScheduleSpec{CronExpressions, TimeZoneName}`, `Paused`, overlap SKIP, a 5m catch-up window, and memo ownership. It also has `Sync`/`Remove`/`Reconcile` (T/core/temporaljobs/agentjobs/definitionschedules.go:94-246). Copy it as `conversation-schedule/{id}`: pause/resume maps to `Paused`, and "run now" maps to Temporal schedule Trigger.
- **Alternative:** user-owned DB rows dispatched by a minute sweep, as in `report.ReportSchedule` (cron_expression, timezone, enabled, run_as_id, next_run_at, consecutive_failures; T/core/domain/report/schedule.go:207-232) with `reportjobs` dispatch (dispatch.go:65-76 via `cronutils.NextRun`, /home/claude/trenova/shared/cronutils/cronutils.go:15, 25).
- **New table:** `conversation_schedules` (thread_id, user_id, prompt, cadence label, cron, timezone, enabled, last_run_at, next_run_at, last_turn_id).
- **Posting results:** follow `assistantfollowupservice.FollowUp` (service.go:92-159), which builds the thread-owner actor, checks agent access, then `turns.StartTurn` → `assistantjobs.StartTurnWorkflow` (assistantjobs/start.go:33). Add a new `AssistantTurnOrigin("Scheduled")` (T/core/domain/conversation/assistantturn.go:52-55).
- **Timezone:** take it from the creating turn's `runReq.Context.Timezone` (assistantservice/turn.go:209).
- **Cadence parsing:** none exists. Add a server parser from "every {weekday|day|morning|Monday…} at {time}" to a 5-field cron, validated with `cronutils.Validate`. Then either let the turn path intercept a leading `every`/`each`/`/schedule`, or expose a CRUD endpoint the client calls.

**(d) Slash commands with typed slots — client-only.** `composer-commands.ts` parses the slots and `fillCommand` expands the template into plain text, which is sent as an ordinary message (W/components/assistant/composer-commands.ts:94-140; desk-slash.tsx). The server sees only text. `/schedule` is the exception: it needs the server-side schedule creation in (c).

---

# Audit 4 — Workspace / artifact pane (excl. billing-item body)

Paths: W=client/apps/web/src/routes/desk/_components/artifacts/desk-workspace.tsx, B=desk-bodies.tsx, T=desk-table-body.tsx, AB=desk-artifact-browser.tsx, CSS=_styles/desk-v2.css. Prototype: artifacts.jsx (P), artifacts-browse.jsx, doc.jsx, uploads.jsx (ExtractBody), desk.css.

## Pane / switcher
- [MATCHES] Pane width — CSS `.dk-stage.dk-open` 520px (CSS:266), overridden to 600px (CSS:5041), sheet `min(600px,100%)` (CSS:2588); same as desk.css:39/859/438.
- [PARTIAL] Front card opens fan — impl fans on **hover/focus** with 110ms delay (`onMouseEnter={open}` W:106, W:94) and closes on mouseleave/blur. Prototype: click on front card opens (P `if (k===0 && !fan) open()`); hover does nothing. Clicking the front card in impl calls `onPick` (re-picks active) instead of fanning.
- [MISSING] Click-outside / Esc closes fan — no mousedown/Escape listener in ArtStack (prototype adds both in a fan effect). Esc in desk-conversation.tsx:364 closes the whole workspace instead.
- [MATCHES] Click card picks + closes; geometry (gap clamp 34–54, translateY k*gap, scale 1-0.035k, opacity k>2, 16ms stagger), "New" chip, vN chip, count+down on front card (W:110–160).
- [MATCHES] Last entry "All N artifacts" / "Search everything this conversation made" / ⌘J kbd (W:162–185).
- [PARTIAL] Stack membership — impl `lineages.slice(0,8)` (server order pinned→newest, W:537). Prototype: active + 3 newest + pinned + recently viewed (session MRU, 6), cap 8. No MRU.
- [MATCHES] Up/down nav buttons step through all artifacts with wraparound + Close (W go()).
- [PARTIAL] ⌘J — two listeners: desk-layout.tsx:342 only `setPane("open")`; W:407 toggles `browsing`. Pane closed → opens pane on current artifact, not the browser. Second press → closes browser but pane stays open. Prototype app.jsx:153: open&&browse → close pane; otherwise open+browse.
- [PARTIAL] Old-version note — impl "Version {i} of {n}, read {time}" (W:580); prototype shows "From {day} · {turn}" for artifacts from earlier days (`a.old`). Day/turn note is missing.
- [MATCHES] Provenance row (tool code, N rows, N calls, "N changed") — W Provenance vs P Prov.
- [MATCHES] Empty state — "No artifacts yet", ⌘J "opens this panel" (desk-artifacts-empty.tsx:35,42).

## All-artifacts browser (AB)
- [MATCHES] Back/title/count/close, autofocus at 60ms, placeholder "Search titles, tools, turns…", Esc clears then backs out, Enter picks the first match, All/Pinned/per-kind chips with counts, day→turn grouping, `<mark>` highlight, 60-item pages + IntersectionObserver "Loading N more…", "No artifacts match “q”".
- [PARTIAL] Empty hint — "Try a tool name like list_workers or a record number." vs prototype "…like `billing.queue.list` or a load ID". The extra "Nothing pinned yet" state is impl-only.

## Footer
- [PARTIAL] Copy link — the label shows `desk/t/…/a/<slug>` (W:594) but the copied URL is `/desk/t/{thread}?a={lineageId}` (W:56). Spec: `desk/c/{conv}/a/{slug}`. The URL contains no slug, the path is not `/c/`, and no route resolves a slug. The slug comes from the title, not from the selected billing item.
- [MATCHES] Pin/Unpin — server-backed `POST /assistant/threads/:id/artifacts/:aid/pin/`, `dk-on` state.
- [PARTIAL] Export CSV — table_view only (`tabular && artifact.kind === "table_view"`, W:607). report_preview gets no button even though `tabular` includes it. CSV is built client-side from the rows already loaded (artifact-export.ts), so a truncated table exports only the visible rows.
- [PARTIAL] Open on its own page — `<a href={link} target=_blank>` reopens the conversation (W:619). No standalone artifact route.

## Bodies
- [PARTIAL] Table — filter, 3-state sort, money right-aligned, status pill by phase, changed-cell mark, 18ms stagger, truncation copy and footer totals all match (T). Diffs: row click does nothing (only the trailing ext `Link`). The footer label is "N rows" where the prototype says "N items". The version picker is the generic one in the stack, so the prototype's simulated "assigned/posted" ctx flash is absent.
- [MISSING] Billing-queue table mode — no BQ detection anywhere in T. Missing: checkbox column (no `.dk-ax-cb` in TSX or CSS; prototype `.ax-cb` 16px, r4.5, ink when on, `.mid` 8×2 dash), row click → open item, live status pill from item state, amber `.ax-nd` needs-count badge, and the floating `.ax-bulk` bar (pop 260ms spring). Copy missing: "N selected", "X ready · Y need you · Z already done or held", Clear, "Review Y" (opens first that needs you), "Approve X" (disabled at 0). Post-approve "Approved N invoice(s)" + Undo with a 6→0 countdown (1s ticks) is also missing. grep of desk/ finds no "bulk"/"already done or held".
- [MATCHES] Record — header, status pill lg, Route with stops/progress/truck (desk-record-view.tsx:202), fields dl, open button. A billing_queue_item card routes to DeskBillingItem (B, out of scope).
- [MATCHES] Rate explanation — winner, tie note, ledger with running bars 60ms stagger, guard, total, warnings, "N agreements didn't apply" toggle. Minor: guard result formatted as money, while the prototype shows result text.
- [PARTIAL] Email draft — To/Subject/body/"Why this wording"/Copy match, but subject and body are `readOnly` (B:462,469). The prototype lets you edit both and has the "Send for approval" → "Sent for approval" button. Impl shows only state text ("Waiting on your approval"/"Sent"/"Decided"); the decision happens in the approval box.
- [MATCHES] Plan — lead, progress bar, "N of M steps done", step list with done/wait/next and 70ms stagger.
- [PARTIAL] Report — DeskReportBars matches bars/total/stagger when there are ≤12 rows. Missing: the "{dataset}" / "{n} stops · top N" meta wording, and the in-body **Download CSV** and **Open full report** buttons. Long reports fall back to the table.
- [MATCHES] Run diff — before→after header, +/−/~/unchanged counts, totals with up/dn/"no change", change rows 50ms stagger.
- [MISSING] Document (doc.jsx/doc.css) — B:264–281 renders `<h2>` + "Written {time}" + `AiMarkdown`, styled as Geist 14px (CSS:6144); Newsreader is never loaded. Every one of these is absent:
  - toolbar: Read/Edit segment, version popover ("You · / Dispatch · … · Latest"), Export menu (PDF / Word / Copy as Markdown / Copy text)
  - TOC nav and old-version bar with "Back to latest" / "Restore"
  - paper: kicker "{docType} · N words · M min read", byline avatar + "{author} · written {at} · {basis}"
  - citation chips with hover popover (code/label/detail/"Open artifact") and the Sources section
  - selection rewrite popover: Shorter / Plainer / "Ask to change…", busy shimmer, suggestion diff with "N → M words", Keep original / Try again / Accept → "Saved as vN"
  - edit mode with contentEditable, "Click any paragraph to edit"/"Unsaved changes", Discard/Done, "Save as vN"
  - toast
- [PARTIAL] View — explanation with term highlights, entity bar, filter count, "Left out “p”" warnings, "Open in {entity}" all match. Missing: the "{count} results" figure and the preview rows (prototype lists matching rows with pills).
- [PARTIAL] Decision — impl delegates to `RequestedDecisionRecords` (B DeskDecisionBody). The prototype's own layout is not reproduced: title + state chip "Waiting on you / Approved / Set aside", scope·meta line, up to 8 `id · cust · from → to` rows, "+ N more", and "Not now" / "Approve N changes".
- [MISSING] Extraction — the client has no `extract` kind; ArtifactBody `default` shows "This kind of artifact cannot be shown here yet." (W:331). `.dk-ex-*` CSS exists (CSS:8215–8280) but no TSX uses it. Missing: page mock with confidence boxes, pager, "N fields need a look", field rows with confidence bars and hover-linking, "Fix fields", and "Create shipment from this" → "Shipment drafted · waiting on your approval".

## BACKEND GAPS
**(a) Bulk approve.**
- What exists:
  - `billingqueueservice.UpdateStatus` is per item, in one tx (service.go:637). Approve also calls `invoiceSvc.CreateFromApprovedBillingQueueItem`, so it creates the invoice in the same transaction.
  - Transitions (domain/billingqueue/transitions.go) allow only InReview→Approved and Approved→InReview. ReadyForReview cannot be approved directly.
  - HTTP: only per-item `PUT /billing-queue/{id}/status|assign|charges`, plus `POST /billing-queue/transfer/`. There is no bulk endpoint and no "bulk" in the handler or service.
  - Job infra: `temporaljobs/billingtransferjobs.BulkBillingTransferWorkflow` is a run row plus a DB-driven batch loop. It has a cancel signal, finalizeFailure on every exit path, and a zombie reconciler. This is the pattern to copy.
- What to build:
  - a `billingqueue_approval_run` + `_run_item` table with per-item status/error and an idempotency key (client-supplied, unique per tenant)
  - `POST /billing-queue/bulk-approve/` returning the run and its per-item results
  - `GET /billing-queue/bulk-approve/{runId}/`
  - `BulkApproveWorkflow` in billingjobs or a new `billingqueuejobs`: start with a 6s durable timer, listen for an `undo` signal (`POST …/{runId}/cancel/`) that ends the run with no writes, then run per-item activities. Each activity re-checks readiness (biller assigned, detention holds, checks), moves InReview→Approved and creates the invoice, and is retried and idempotent per item. Partial failures are recorded per item.
  - Undo after commit would also have to void or delete the draft invoice. Deferring the commit avoids that.

**(b) Document.**
- What exists: `publish_artifact` (agentruntime/publish.go) and `artifactRecorder.publish` (artifacts.go:193). Republishing with `artifactId` **overwrites the same row** via Upsert. `KindDocument` is not in `versionedKinds` (assistantartifact/lineage.go), so documents get no versions. Payload is just `{format:"markdown", body}`, with no blocks, citations/sources, docType/author/basis.
- What to build:
  - user-facing endpoints: `POST /assistant/threads/:id/artifacts/:aid/versions/` for an edit-mode save, which appends a version with author=user
  - `POST …/rewrite/` taking {blockId|range, mode shorter/plain/ask, prompt}, returning a suggestion without saving; Accept → a new version
  - `POST …/restore/`
  - document lineage/versions (add KindDocument to lineage or a version table)
  - export: PDF/DOCX render endpoint (none exists; `/documents/{id}/download/` is for uploaded documents). Markdown/text can stay client-side.

**(c) Extraction.** There is no `extract`/`extraction` Kind in `assistantartifact.AllKinds()` (enums.go). The only agent path is `get_shipment_draft` (agentquerytoolservice/intake_tools.go:45), which `artifactFromObservation` turns into a generic **entity_card** through the `get_` prefix (artifacts.go:708). Its draft view has confidence but no bounding boxes or pages. To build: KindExtraction, a mapping from `get_shipment_draft` and documentintelligenceservice results (fields, confidence, bbox, page, classification %), and "Create shipment from this" as a proposal.

**(d) CSV.**
- Table export is client-side only (artifact-export.ts), so a truncated table loses rows.
- The server has `GET /reports/runs/:runID/download/` (reporthandler) and `infrastructure/reporting/render/csv.go`, usable for report_run. Nothing covers report_preview or table_view.
- Needed: `GET /assistant/threads/:id/artifacts/:aid/export.csv`, which re-runs the source query unpaged or uses the report renderer.

**(e) Pagination.** `GET /assistant/threads/:id/artifacts/` returns `{results}` with no cursor or count. `ListThreadArtifacts` passes no Limit, so the repo defaults to 100 (max 500) and silently drops older artifacts (assistantartifactrepository/artifact.go:23,113). The browser's "All N" and infinite scroll are therefore capped at 100. Needed: a cursor (createdAt,id), a total count, and a q/kind/pinned filter server-side.

---

# Audit §5: Billing queue item artifact

Prototype: `design/billitem.jsx` and `billitem.css`. Ours: `desk-billing-item.tsx` (`DeskBillingItem`), `billing-item-checks.ts`, `desk-bodies.tsx:78` (mounted only for `card.entity === "billing_queue_item"`), and the `.dk-bq-*` rules in `desk-v2.css` (around lines 12787–13246).

## UI

| # | Spec item | Verdict | Diff |
|---|---|---|---|
| 1 | Kicker `BQ · INV-D/INV · PO` | PARTIAL | Shows `number · proNumber · bol`. Missing: the invoice number (and the draft→posted swap), the PO, and the 3px dot separators. The first id is not `--fg2`. Font is 11.5px `muted`; prototype is 11px `subtle`. |
| 2 | Up/down arrows + "n of N" | MISSING | The component takes a single `itemId` and knows nothing about the queue. |
| 3 | Customer 20px/600 + bill-to line | PARTIAL | Name is 21px with -0.015em; prototype is 20px with -0.02em. The sub line is `code · billType` instead of "AP name · email". It is 12.5px `muted`; prototype is 12px `subtle`. |
| 4 | Total 24px mono, animates on change | PARTIAL | Font matches. The `biamt` 420ms animation, the `key={total}` remount and `tnum` are all missing. |
| 5 | "Net N · due {date}" | MISSING | Shows "Their share of $X" only when the item is a split. |
| 6 | Status pill variants | PARTIAL | Uses the generic `.dk-ax-pill.dk-lg`: 6px dot, always on a sunken background, padding 4px 10px. Prototype: 24px tall, padding 0 10 0 9, 7px dot, b-sub inset ring, `pop 240ms`. The variants are missing: Approved (check on `success-sub`), Posted (check on `--ink`), On hold (pause on `warn-sub` + "· reason"). Ours only recolours the dot. |
| 7 | "Queued … · N days in queue" | PARTIAL | Present, but the day count is hidden when it is 0 or the item is settled. Font 12.5px vs 12px; gap 8px vs 6px 10px; no faint `·` element. |
| 8 | Release button on the status line | MISSING | Release lives in the Hold menu as "Take off hold". |
| 9 | No progress bar | MATCHES | |
| 10 | Checks shown only while Ready for review | PARTIAL | Shown for every status that is not Approved, Posted or Canceled. |
| 11 | "N of 5 need you" / "All clear" | PARTIAL | The denominator is a variable `checks.length` instead of 5. |
| 12 | The five checks | PARTIAL | Only Biller matches. The rest is a dynamic list: Required documents, Shipment details ×n, Worth a look, Service failures, Detention waits on approval, Charges (payer-split error only), Bill-to (credit hold). Missing: Charges match the rate con, Proof of delivery (signed), Bill-to and terms, Not a duplicate. |
| 13 | Compact passing rows | PARTIAL | Layout matches. Differences: title is 500 `--fg` (prototype 400 `--fg2`); 13/12.5px (prototype 12.5/12); detail colour `muted` (prototype `subtle`); flex with gap 12 and padding 12/14 (prototype 20px grid, gap 10, padding 10/12); animation `dk-mprow 320` (prototype `nl 260`). |
| 14 | Expanded failing rows | PARTIAL | Note is 12.5px `subtle` without `text-wrap:pretty`; prototype is 12px `muted` with it. The warn icon has no `warn-sub` background. The fail icon colour is `canvas`; prototype uses `ink-fg`. |
| 15 | Assign to me / Someone else… | PARTIAL | "Assign to me" works (GraphQL `assignBillingQueueBiller`). "Someone else…" is a Link to the queue page, with no inline biller list, `.bi-av` initials or "you" tag. Buttons are 30px/12.5px with a `b-strong` ring; prototype is 28px/12px with a `--b` ring on `--card`. Gap 8 vs 6; margin 10 vs 9. |
| 16 | Issue actions (lumper, POD, detention vs ELD) | MISSING | Only `action:"assign"` exists. Detention holds show but cannot be acted on. |
| 17 | Contract/agent reasoning | PARTIAL | Only static notes on detention holds and payer errors. |
| 18 | Ledger: Charge · Rate con · Billed | PARTIAL | No Rate con column. Uses a `<table>` instead of the `1fr 92px 92px` grid with 26px undo gutter. The basis shown is the kind or "% share", not the rating basis. Header 11.5/500 vs 11; label `--fg` vs `--fg2`; basis 12px with no ellipsis vs 11.5px with ellipsis; numbers 12.5 vs 12. |
| 19 | Flagged / removed / adjusted lines, inline undo | MISSING | |
| 20 | Total + "$X over/under the rate con" | PARTIAL | The total row is 15px; prototype is 13px/600. Over/under and the "vs rate con $X" aside are missing. |
| 21 | Lane card | PARTIAL | Built from the stops; content matches. Gap 12/11, padding 12·14 vs 10·12, icon 32/r9 vs 30/r8, title 13/600 vs 12.5/500, sub 12 `muted` vs 11.5 `subtle`. The arrow is not faint. |
| 22 | Shipment link | MATCHES | Style differs: 12px `--fg` vs 11.5px `--fg2`, and the icon is not `subtle`. |
| 23 | Document tiles, missing/unsigned in amber | PARTIAL | Shows any current docs, up to 8, as divs (not buttons) with no hover. No `.miss` amber state and no tiles for required docs that are missing. minmax 118 vs 130; gap 8 vs 6; ring `--b` vs `--b-sub`; title 12.5 `--fg` vs 12 `--fg2`. |
| 24 | Activity timeline | PARTIAL | Last 8 audit rows, gated on `AuditLog:Read`. Missing: filled dots for your own entries, and any agent vs you distinction. Text is the raw audit comment ("…status updated to OnHold"). Dot 9px vs 8px with no `--card` fill; text 13/500 vs 12.5/400; meta 12 vs 11. |
| 25 | Sticky bar | PARTIAL | Solid `card` background; prototype uses `color-mix 92%` + `blur(8px)`. z 2 vs 4; padding 12 vs 10. Reason text is 12.5 `muted` and left-aligned; prototype is 11.5 `subtle`, `flex:1`, right-aligned. Custom 34px approve button vs `ax-btn ink`; disabled opacity .4 vs .35. |
| 26 | Hold menu with 3 reasons | MISSING | One item, "Put on hold", which sends `{status:"OnHold"}` with no reason. Menu is 240px min-width, r11, `dk-mprow` animation; prototype is 200px, r12, `pop 200 spring`, `lift-hi`. |
| 27 | Reason text variants | PARTIAL | "Assign a biller first" matches. Missing: "Settle the flagged check first", "Posting sends it…", "Release the hold to continue". Ours adds five extra variants. |
| 28 | Approve → Post $X → "Posted as INV…" | MISSING | The footer disappears once the item is Approved. There is no Post step and no posted line. |
| 29 | State persists while navigating | PARTIAL | State is server-side through React Query, so it persists, but there is no in-artifact navigation. |
| 30 | Table ↔ item sync | PARTIAL | Mutations invalidate the list, readiness and audit queries. There is no selected-item store, so the slug does not follow the selection. |

**Bug risk:** `approveBlocker` returns "Start the review first" unless the status is `InReview`, and the card has no Start-review action. Only `PlanAssignBiller` (status_plan.go) moves ReadyForReview to InReview. A ReadyForReview item that already has a biller therefore cannot be approved from the desk.

## Backend gaps

1. **Customer rate con per charge.** `rateconfirmation.RateConfirmation` (rateconfirmation.go:63) is carrier-only (`CarrierID`, `CarrierAssignmentID`, `PayloadSnapshot`) and has no lines. Customer-side expectations already exist in three places:
   - `shipment.RatingDetail` (shipment.go:51): `Breakdown[]{Name,Label,Amount}`, `AgreementID`, `RuleID`, `RateQuoteID`.
   - `ratequote.RateQuote` (quote.go:36): linehaul, fuel, accessorial and total amounts, plus `Trace`.
   - `rateagreement.RateAgreementAccessorial`: per-accessorial `Amount`, `Waived`.

   **Add:** a `ChargeComparison{AdditionalChargeID, Label, Basis, Expected NullDecimal, Billed, Source}` computed in `billingqueueservice.GetByID` next to `PayerShare`. Match `PayerShareLine.AdditionalChargeID` → `AccessorialChargeID` → the agreement accessorial, and take linehaul and fuel from the quote. Expose `rateConTotal` and `lines[].expected` in REST and GraphQL. Add `PayerShareLine.Basis` (payershare.go:29).

2. **Duplicate detection.** None exists. Migration `20260407160506_remove_billing_queue_item_unique_index` dropped `idx_billing_queue_items_shipment_bill_type`. The only remaining guard is `uk_invoices_billing_queue_item`. **Add:** a repository method `FindPotentialDuplicates(shipmentID, billTo, billType, amount)` over non-canceled items and non-voided invoices, matching on shipment, `ExternalReference` or `order.PONumber`. The outcome enum `ExceptionDuplicateCharge` already exists.

3. **Signed POD.** `document.Document` has `Status` and a review approve/reject flow. There is no signature field on `document` or `documenttype` (`documenttype` has only `Code` and `DocumentClassification`). `GetBillingReadiness` (shipmentservice/billing_readiness.go) checks only that each required type is present. **Add:**
   - `documents.signature_status` enum (`Unknown|Signed|Unsigned`), plus `signed_at`.
   - `documenttype.requires_signature`.
   - Populate them from `documentintelligenceservice` extraction.
   - A `signed` flag on readiness `requirements[]`.
   - A driver document-request endpoint for "Ask the driver".

4. **Detention vs ELD.** `DetentionOccurrence` has `ArrivedAt`, `DepartedAt`, `BillableMinutes`, `BillableAmount` and `CalculationTrace`. `DetentionEvidence` records a `Source` (Telematics, Geofence, DriverApp…) for each Arrival and Departure. Telematics has `GeofenceEntry`/`Exit` events. A billing-queue `DetentionHold` exists, enforced by `guardDetentionHolds` (service.go:171). **Add:** a dwell comparison between DriverApp evidence and Telematics/Geofence evidence (minutes and delta). Add `POST /billing-queue/{id}/detention/{occurrenceId}/resolve {eld|drop|keep}` that recalculates through `detentionbillingservice`.

5. **Issue/resolution model.** None exists. The only fields are `ExceptionReasonCode`/`ExceptionNotes` and readiness messages. **Add a table** `billing_queue_issues` with these columns:
   - `item_id`, `check_key` enum (biller, charges, pod, terms, duplicate), `code`, `summary`
   - `reasoning`, `source` enum (Deterministic, Agent), `agent_run_id`, `flagged_charge_id`
   - `options` JSONB `[{key, label, effect:{keep|drop|set, chargeId, amount, basis}}]`
   - `resolution_key`, `resolved_by`, `resolved_at`

   **Add endpoints** `POST …/issues/{issueId}/resolve` and `…/undo`, with matching GraphQL mutations.

   **Recommendation:** compute issues with deterministic checks in `TransferToBillingItems` and `UpdateCharges`. The agent may attach reasoning and contract citations only. Gate Approve in `UpdateStatus` the way `guardDetentionHolds` does. Apply resolution effects through `planChargeUpdate`.

6. **Hold reasons.** `applyStatusFields` (`case StatusOnHold`) stores only `ReviewNotes`. **Add:** a `HoldReasonCode` enum (`WaitingOnPaperwork|CustomerDispute|RateQuestion`) and columns `hold_reason_code`, `held_at`, `held_by_id`. Add `holdReasonCode` to `BillingQueueUpdateStatusInput` (billing_queue.graphqls) and to `UpdateBillingQueueStatusRequest`. Clear them on leaving OnHold.

7. **Release.** The transitions OnHold→ReadyForReview and OnHold→InReview exist (transitions.go). **Add** `status_before_hold` so a release restores the status the item had before the hold.

8. **Post from the item.** Approve already creates the draft invoice: `UpdateStatus` calls `CreateFromApprovedBillingQueueItem` with `DeferToStatement:true`, and auto-post goes through `EnqueueAutoPost`. Posting is `invoiceservice.Service.Post` (service.go:467), and `markBillingQueueItemPosted` then sets the item to Posted. **Add** `POST /billing-queue/{id}/post` and a GraphQL `postBillingQueueItem` that resolves `item.InvoiceID` and calls `Post`. They should return the invoice number and the recipient (`Customer.EmailProfile`).

9. **INV-D vs INV numbers.** `generateInvoiceNumber` (numbering.go) runs once, when the draft is created (create.go:133/352/449). There is no draft number. Either drop INV-D from the design, or add `invoices.draft_number` and assign the final number at Post. Either way, join the invoice `Number` and `Status` onto the item.

10. **Activity log.** `logAction` (service.go:919) writes audit rows with `PrincipalType` and `APIKeyID`. That is enough to tell system, agent and user apart, but the comments are not human-readable, and issues and hold reasons are not recorded. **Add** a `billing_queue_events(item_id, kind, text, actor_type, actor_id, at, payload)` table and `GET /billing-queue/{id}/activity`. This also removes the dependency on `AuditLog:Read`.

11. **"n of N" ordering.** The list is ordered by `created_at DESC, id DESC` (billingqueuerepository/billingqueue.go:338) and has no neighbour lookup. **Add** `GET /billing-queue/{id}/neighbors?filter=…` returning `{prevId, nextId, position, total}` with the same filter and sort as the list, using a keyset query.

12. **PO source.** `order.Order.PONumber` (`orders.po_number`); the item already has `OrderID`. Shipments have no PO, only `BOL` and `ExternalReference`. **Add** an `Order` belongs-to relation or a `poNumber` projection on the item.

13. **Terms and due date.** Terms come from `CustomerBillingProfile.PaymentTerm` (billingprofile.go:40). Due date is `invoice.DueDateFromPaymentTerm`, called in `buildInvoiceEntity` (service.go:1168). **Add** `paymentTerm` and `dueDate` to the item response, taking the draft invoice's `DueDate` when one exists. Expose the AP contact from `Customer.EmailProfile` for the bill-to line and "sent to".

---

# Audit 6/7: Memory page + Agent capabilities page (Desk v2)

Prototype: design/pages.jsx `MemoryPage` (99-163), `AgentPage` (167-206); side.jsx:37 (rail Memory), side.jsx:74,80 (agent name button + shield icon); app.jsx:404-408 (view routing).
Client paths: client/apps/web/src/... ; backend: services/tms/internal/...

## Reachability / routing
- [MISSING] `/desk/memory` route — router.tsx:2003-2050 has only index, `t/:threadId`, `decisions`, `watchtower`.
- [MISSING] `/desk/agents/:id` (capabilities) route — not present.
- [MISSING] Rail "Memory" item — desk-rail.tsx navigates only to /desk, /desk/watchtower, /desk/decisions (173,185).
- [MISSING] Top-bar agent name as button "What this agent can do" — desk-topbar.tsx:61-62 renders a plain `<span className="dk-ttl-a">`.
- [MISSING] Shield icon button in top bar — desk-topbar.tsx:77-112 has only Download, Pin, Workspace.
- [MISSING] Top-bar titles "Memory" and "{agent} / What it can do" — desk-topbar.tsx:67-73 knows only Decisions/Watchtower/Today.
- [MATCHES] TodayPage not built (correct per spec).

## Memory page (client)
No Desk memory page exists. Existing admin-only memory UI: routes/agent-control/_components/memory/{memory-tab,memory-columns,memory-panel,memory-suggestions}.tsx (data table: content, kind, about, status, source, Read N times, last read; memory-columns.tsx:39-188). Data layer to reuse: lib/graphql/agent-memories.ts (createAgentMemory, updateAgentMemory, setAgentMemoryStatus, agentMemoryTableGraphQLConfig).
- [MISSING] 720px page, kicker "MEMORY", h1 "What Desk remembers", explanation copy.
- [MISSING] "Saving new memories" segmented Automatically / Ask me first + dynamic sub-copy.
- [MISSING] Add form: "Teach Desk something, e.g. …" input, scope select (Just you / Billing team / Organization), Save disabled when empty, new row prepended with fresh animation.
- [MISSING] Filter chips All/Just you/Billing team/Organization with counts (counts exclude forgotten); "Search memories" input; "No memories match." empty state.
- [MISSING] Row text; inline scope select; "Saved {date} from “{source}”"; "Used N× · last {when}" / "Paused" / "Not used yet".
- [MISSING] Hover actions: Edit (autoFocus textarea rows=2, Cancel/Save disabled when blank), Pause/Resume, Forget → "Forgotten. Agents won't use this again." + Undo.
- [PARTIAL] Backing data/calls exist in lib/graphql/agent-memories.ts but are wired only into AI Control.

## Memory backend (services/tms)
Domain: internal/core/domain/agent/agentmemory.go:220-262 (`agent_memories`, migration migrations/20261231001200_agent_memories.tx.up.sql + 005200/006200 follow-ups). Service: internal/core/services/agentmemoryservice/service.go. Repo: internal/infrastructure/postgres/repositories/agentmemoryrepository/agentmemory.go. GraphQL: internal/api/graphql/schema/agent.graphqls:497-753.
- [PARTIAL] Scope — `MemoryScope` is only Organization | Agent (agentmemory.go:108-126). No per-user scope and no "team". Trenova has no Team/Department domain (internal/core/domain has none); nearest groupings are `permission.Role` (domain/permission/role.go:29) or business unit (already tenant key). Add: `MemoryScopeUser` + `owner_user_id`, and `MemoryScopeRole` + `role_id` (map "Billing team" → role) — or drop team from the design. Extend `forAgent`/`sameReaders` (agentmemory.go repo:601-630) to include `scope=User AND owner_user_id=actor` and `scope=Role AND role_id IN actor roles`; ContextBuilder.readMemories (agentruntime/context.go:109,178) must pass the run's user + roles.
- [PARTIAL] Scope in API — `AgentMemoryInput` (agent.graphqls:565-575) has no `scope` field; `CreateAgentMemory` resolver (agentresolver/agent.resolvers.go:291-311) never sets it; `NewMemory` (agentmemoryservice/service.go:~196) leaves DB default 'Organization'. Only `ApproveAgentMemorySuggestionInput.scope` exists. Add `scope`/`roleId` to input + RememberRequest; allow scope change on update.
- [MISSING] Paused — statuses are Active/Retired/Suggested/Dismissed (agentmemory.go:132-161). Add `MemoryStatusPaused` (or `paused bool`); `activeOnly` (repo:~636) already filters `status='Active'`, so a new status is automatically excluded from prompts and recall_memory. Update `PlanStatus`/`checkStatusTarget` (status_plan.go:27-80) to allow Active↔Paused.
- [PARTIAL] Forget + Undo — `setAgentMemoryStatus(id, Retired)` soft-deletes (retiredAt/retiredByUserId, status_plan.go:45-60) and `Active` restores, so Undo is supportable today with two calls. Retired memories are excluded by `activeOnly`. No time-boxed undo needed server-side.
- [MATCHES] Usage counts — `use_count`/`last_used_at` (agentmemory.go:251-252) incremented by `MarkUsed` (repo:462-486) via `agentruntime/memory.go:13-43` for memories actually fitted into the prompt (`Definition.FitMemories`, agentdefinition/memoryprompt.go:58). Exposed as `useCount`/`lastUsedAt` in GraphQL. Note: recall_memory reads are not counted.
- [PARTIAL] Source — `source` enum User/Agent/Decision/Feedback plus `sourceRunId`/`sourceProposalId` (agentmemory.go:81-97,244-245). No conversation title. `AgentRun.TurnID` (domain/agent/agentrun.go:60) links to a turn → thread; add a resolved `sourceLabel` (thread title, or "Added by you" for Source=User) field on `AgentMemory` (agentmapping.go) or store `source_thread_id`.
- [MISSING] Per-user "Automatically / Ask me first" saving mode — no setting anywhere. The `remember` tool (agenttoolservice/memory_tools.go:33,85-91) has `DefaultTier: TierActWithApproval`, `MaxTier: TierAutoExecute`, so ask-vs-auto is per-agent tool tier, not per person. Add a user preference (e.g. assistant/desk user settings) and have the runtime clamp `remember`'s tier to ActWithApproval when the person chose "ask" (TierSourcePersonSetting already exists in domain/agent/autonomy.go:58-64 as a concept).
- [PARTIAL] List/filter/search — `agentMemories(input: DataTableConnectionInput!)` (agent.graphqls:709) supports cursor pagination, field filters (status) and sort; repo has a tsvector `search_vector` + textquery.go. Counts per scope would need `totalCount` per filter (3 extra queries) or a new `agentMemoryCounts` query.
- [MISSING] Desk-user permissions — every memory query/mutation requires `ResourceAgentMemory` (resolvers 293,352,603), registered with Category "Administration", `SensitivityRestricted` (domain/permission/resources_billing.go:~293-310). A normal Desk user (Resource.Assistant) cannot list or add memories. Need either a `myMemories`/`createMyMemory` surface gated on `ResourceAssistant` that forces scope=User for non-admins, or a grant model: users may write own User-scope memories; Role/Org scope requires ResourceAgentMemory OpCreate/OpUpdate.
- [MISSING] Visibility filter in list — list endpoint returns all org memories; a Desk list must restrict to User(own)+Role(mine)+Org.

## Agent capabilities page (client)
No Desk page. Admin editor exists: routes/agent-control/_components/agents/agent-form.tsx (toolTiers 101/278-284, delegateIds 103/326-347, monthlyBudgetUsd 458, dailyRunLimit 471, preferredProviderId 634, enabled 744), agent-form-schema.ts (toolDailyLimits 33, memoryTokenBudget 135), budget.tsx, delegates-field.tsx, tool-catalog.ts.
- [MISSING] "Back to conversation" button.
- [PARTIAL] Agent tile lg — `DeskAgentTile` exists (desk-agent-tile.tsx, used at xs in desk-topbar.tsx:61); no lg header usage.
- [MISSING] Name + "{desc} · {model} · set up by {owner}".
- [MISSING] On/Off switch.
- [MISSING] "Look things up" / "Make changes" sections with subtitles "What it can read" / "What it can change, and when it asks you".
- [MISSING] Rows: label + mono tool key + segmented Allowed/Ask first/Off (read rows omit Ask first), neutral selected state (`.mm-seg button.on` card bg, pages.css:65), locked rows disabled with lock icon + reason.
- [MISSING] "Hands off to" (topic → agent) section, subtitle "Questions outside billing go to these agents".
- [MISSING] Limits: Requests today x/y "Resets at midnight", "{Month} budget" $x/$y "resets {date}", bar `.hi` amber when >85%, "Largest single change N items / Bigger batches are split and approved separately", "Only change things during business hours" switch + "7 AM – 6 PM Central".

## Agent config backend
Domain: internal/core/domain/agentdefinition/definition.go:52-134. Admin REST: internal/api/handlers/agentdefinitionhandler/handler.go:54-73 (GET/PUT/DELETE, `/:agentID/budget/`), all on `ResourceAgentDefinition` (Category "Administration"). GraphQL: schema/agentdefinition.graphqls:308-327.
- [PARTIAL] Tool modes — `ToolNames` (Off = absent) + `ToolTiers map[tool]AutonomyTier` (definition.go:66-68) with tiers Propose / ActWithApproval ("Ask first") / AutoExecute ("Automatic") (domain/agent/enums.go:278-280; labels autonomy.go). Enforced via `EffectiveTier` (definition.go:214-229) clamped by `AutonomyCeiling`. Map: Allowed=AutoExecute, Ask first=ActWithApproval, Off=remove from ToolNames. Propose has no design equivalent — decide whether to collapse it into "Ask first". Trust-earned tiers (TierSourceTrustEarned) can raise a tool above the saved tier; page must show effective vs set.
- [PARTIAL] Read vs write split — `ToolPolicy.Kind`/`Effect` (ports/services/toolpolicy.go:11-33) distinguishes; query tools (agentquerytoolservice) have no tier. Expose kind in the tool catalog projection.
- [PARTIAL] Locks + reason — derivable from `ToolPolicy.MaxTier` (< AutoExecute ⇒ "Allowed" disabled), `Reversible=false`, `TierCondition.Description`, `TaintHold.Description`, `Rationale`. No explicit `locked`/`lockReason` field; add a computed `lock {reason, maxMode}` to the tool catalog served by `GET /agent-definitions/tools/`. Design's lock forces "Ask first" entirely (Off also disabled); current model only caps the top.
- [PARTIAL] Hand-offs — `DelegateIDs` (definition.go:115-119) is an ordered agent list; no topic label. `MyAgent.delegates` (agentdefinition.graphqls:213-218) already exposes them to Desk users. Add `delegateTopics map[id]string` (or `[]{agentId, topic}`) to render "Pay rates, people records → Workforce Coordinator".
- [MATCHES] Daily request limit + enforcement — `DailyRunLimit` (definition.go:96) enforced in agentbudgetservice/service.go:100-117 via `CheckRun` (called assistantservice/conversation.go:114, agentrunservice/service.go:216). Counts runs (not requests per person).
- [MATCHES] Monthly budget + enforcement — `MonthlyBudgetUSD` (definition.go:95), `CheckRun` monthly branch (agentbudgetservice:81-98). Usage readout: `Status` (agentbudgetservice:160-215) and Desk-reachable `GET /assistant/threads/:threadID/budget/` (assistanthandler/handler.go:72-76; assistantservice/budget.go:21-66: spentUsd, limitUsd, share, runsToday, dailyRunLimit, resetsAt, dayResetsAt). Note `budgetNearShare = 0.9` (budget.go:16) vs design's 85% amber threshold.
- [PARTIAL] Per-tool daily limits — `ToolDailyLimits` enforced (`CheckTool`, proposalexecutor/executor.go:585, agentruntime/dispatch.go:336); not in design.
- [MISSING] Largest single change (N items, split batches) — no definition field. Only hard-coded caps: `maxBulkRecords = 50` (agenttoolservice/bulk_records.go:23), `MaxBatch = 50` (agentdecisionqueueservice/service.go:27). Add `MaxChangeItems int` to Definition + migration; enforce in bulk tools (`toolschema.KeyMaxItems` sites) and split in proposal creation.
- [MISSING] Business-hours rule — nothing in domain/agentdefinition or services (no business-hours symbols). Add `WritesBusinessHoursOnly bool` + window/timezone (reuse `CronTimezone` or org zone from agentbudgetservice `organizationZones`); enforce in agentruntime/dispatch.go before executing write tools (hold as proposal outside hours).
- [MATCHES] On/Off — `Enabled`, `DisabledAt`, `DisabledByID` (definition.go:72-76); ThreadBudget already returns disabledAt/disabledBy.
- [PARTIAL] Model — `PreferredProviderID` (definition.go:112) → provider.model; runs record `ModelIdentifier`. Needs a resolved model display name on the Desk projection.
- [MISSING] Owner "set up by" — Definition has no created-by/owner field. Add `CreatedByID` (or read from audit log on create).
- [MISSING] Desk-user access — full Definition read/write is admin-only (`ResourceAgentDefinition` on agentDefinition/agentDefinitions resolvers agentdefinition.resolvers.go:154,180 and REST). `myAgents` (ResourceAssistant, :197) returns only name/desc/icon/tools/delegates. Add a `myAgentCapabilities(agentId)` read (tools+effective mode+lock, delegates+topics, limits+usage, hours, enabled, model, owner) gated on Assistant read + agent access; keep mutations (`PUT /agent-definitions/:id/`) on ResourceAgentDefinition OpUpdate and render controls read-only for non-admins (design shows editable controls — confirm intent).

---


---

# Summary

Everything in the handoff's Desk scope is implemented end to end, with the
backend it needs in Go; the Today page is not built, by instruction. What
follows is what changed in the database and the API, and what does not
match the prototype one for one, with the reason.

## Migrations

| Version | What it adds |
|---|---|
| 20261231007460 `assistant_thread_pinned_facts` | The facts a conversation keeps in mind, injected into every prompt. |
| 20261231007470 `agent_proposal_approving` | The `Approving` proposal status the undo window holds a decision in. |
| 20261231007480 `agent_decision_undo_window` | When an approved decision commits, for the server-enforced undo. |
| 20261231007490 `conversation_schedules` | Scheduled requests, the `Schedule` message kind and the `Scheduled` turn origin. |
| 20261231007610 `assistant_artifact_workspace` | Artifact slugs, versions of a document, the extraction kind, and the indexes the paged list reads. |
| 20261231007700 `assistant_conversation_compaction` | Context usage on the thread, the `Compaction` message kind and turn origin. |
| 20261231007710 `agent_memory_audience` | Memories for one person or a role, `Paused`, usage counts, and the per-person saving preference. |
| 20261231007720 `agent_capabilities` | Who set an agent up, its largest change, business hours, hand-off topics and tools switched off. |
| 20261231007730 `assistant_handoff` | The conversation a hand-off came from and the `Handoff`/`HandoffBrief` message kinds. |
| 20261231008100 `billing_queue_review` | Checks, issues, activity, holds and bulk approval runs for the billing queue. |

Each migration that redefines a shared check constraint keeps every value the
earlier ones allowed.

## Endpoints

- Assistant (REST, `/api/v1/assistant`): `POST /threads/:id/handoff/`,
  `POST /threads/:id/compact/`, schedules (`GET /schedules/`,
  `GET|POST /threads/:id/schedules/`, `PATCH|DELETE /schedules/:id/`,
  `POST /schedules/:id/run/`), the paged artifact list
  (`GET /threads/:id/artifacts/` with `limit`, `cursor`, `q`, `kind`, `pinned`),
  `GET /threads/:id/artifacts/by-slug/:slug/`, an artifact's lineage, CSV and
  document export, and document `versions/`, `restore/` and `rewrite/`.
  Pinned facts and auto-compaction ride on `PATCH /threads/:id/`.
- Billing queue (REST): `GET /summaries/`, bulk approval
  (`POST /bulk-approve/`, `GET /bulk-approve/:run/`, `POST /bulk-approve/:run/cancel/`),
  and per item `neighbors/`, `activity/`, `issues/:issue/resolve/`,
  `issues/:issue/undo/`, `release/` and `post/`.
- GraphQL: `undoMyDecision`, `commitMyDecisionNow`; `agentCapabilities`,
  `updateAgentCapabilities`; the Memory page (`deskMemories`,
  `deskMemoriesByIds`, `createDeskMemory`, `reviseDeskMemory`,
  `setDeskMemoryStatus`, `confirmDeskMemory`, `dismissDeskMemory`,
  `setMemorySavingMode`); billing (`postBillingQueueItem`,
  `releaseBillingQueueItem`, `resolveBillingQueueIssue`, `undoBillingQueueIssue`).
- Stream events over the existing assistant stream: `memory_used`,
  `memory_saved`, `context`, `compaction_started`, `compaction_finished`,
  `compaction_cancelled`, and `why` on `tool_started`.
- Temporal: `proposal-commit/{id}` (the undo window), `conversation-schedule/{id}`
  (a Temporal Schedule per request, reconciled hourly), the compaction
  activity, and the billing bulk-approval workflow.

## What does not match one for one, and why

- **Today page**: not built, by instruction; Today opens the home composer.
- **Narrated mode and Replay**: the prototype's narrated mode is unreachable
  there (it hard-codes ledger) and Replay replays canned data; neither ships.
- **Tool and record labels**: the agent page names tools by their humanized
  names ("Get shipment"), and the Memory page's team filter by the person's
  role ("Organization Administrator"), not the prototype's hand-written
  phrases; the data, not the layout, differs.
- **Document blocks**: the prototype's documents are typed demo blocks (a
  loads table with a late column, an "Open draft" note, a source line under
  a table). A document here is the model's markdown, so tables, notes and
  citations render through the markdown set; those three demo-only blocks
  have no source.
- **Artifact badges under a reply**: the prototype lists a reply's artifacts
  as a row of badges under it; per review feedback they open from the
  sentence that names them instead, inline.
- **Short billing queues**: a list of twelve rows or fewer is answered as a
  markdown table in the reply, except the billing queue, which always opens
  as the selectable table the design draws.
- **Email draft**: "Send for approval" hands the edited wording to the
  decision card on the composer, which approves with it as a modification;
  the wording is held in the page, so an edit sent and not approved before a
  reload goes back to the agent's.
- **Extraction actions**: "Create shipment" and "Fix fields" ask the agent,
  which proposes the change for a decision, rather than writing directly.
- **Fonts**: the prototype's Geist and IBM Plex Mono are the app's font
  variables; colours are the `--dsk-*` tokens, light and dark.
