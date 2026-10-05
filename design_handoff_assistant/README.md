# Handoff: Trenova Assistant redesign + refactor

## Summary
The in-app Assistant (`client/apps/web/src/components/assistant/assistant-widget.tsx` and friends) already exists. This is a **redesign of its shell** and a **refactor of its chat interior** so the chat *is* Desk v2's chat, rendered compactly.

Today the widget and Desk render conversations with two separate UIs:
- **Widget**: `assistant-panel.tsx` → `message-thread.tsx`, `message-items.tsx`, `streaming-turn.tsx`, `tool-activity.tsx`, `composer.tsx` (42 KB), `model-picker.tsx`, `approval-dock.tsx`, `thread-sidebar.tsx`, `assistant-home.tsx`, `assistant-header.tsx`
- **Desk**: `routes/desk/_components/desk-conversation.tsx`, `conversation/*` (desk-turns, desk-citations, desk-web, desk-approval-card, desk-undo-bar, desk-compact-mark, desk-message-actions, desk-failures, desk-tool-failures, use-stick-to-bottom…), `composer/*` (desk-composer, desk-agent-picker, desk-model-picker, desk-page-chip, desk-context-meter, desk-dictate, desk-uploads, desk-capture, desk-mentions, desk-slash, use-typewriter), styles in `routes/desk/_styles/desk-v2.css`

**Goal: one chat implementation.** Desk's conversation and composer components become shared, the widget mounts them, and the widget's duplicate rendering stack is deleted. Shared logic in `components/assistant/` (`turn-stream.ts`, `activity.ts`, `approval-queue.ts`, `approval-actions.ts`, `compaction.ts`, `composer-commands.ts`, `use-composer-context.ts`, `use-composer-dictation.ts`, `use-thread-model.ts`, `use-opening-question.ts`, `use-active-turns.ts`, `page-context.ts`, `proposal-*`, `readable-values.ts`…) is already used by both and stays where it is.

> **DRY is the requirement.** Don't write new message, streaming, footnote, web-search, approval, undo, compaction, picker, dictation, upload or composer components. Move Desk's existing ones to a shared location, and add a `density="compact"` prop (or a CSS scope) where a 400px panel needs different sizing. Where the widget and Desk duplicate a component, keep Desk's and delete the widget's.

## Design reference
`design/index.html` is an HTML/React prototype; open it and click the trigger (bottom-right) or press ⌘J. Tweaks switch layout (compact / side / full) and theme. Treat it as a visual and behavioural reference, not code to port.
- `app.jsx`, `sidebar.jsx`, the top half of `assistant.css`: **the new shell** (the only genuinely new UI)
- `parts.jsx`, `desk-agents.jsx`, `models.jsx`, `context.jsx`, `pagectx.jsx`, `desk-bits.css`, the bottom half of `assistant.css`: copies of Desk prototype code, showing that the interior is unchanged Desk
- `backdrop.jsx`, `tweaks-panel.jsx`, mock scripts/timers: prototype only

---

## 1. Shell redesign (new UI; replaces existing widget chrome)

### 1.1 Trigger ("beacon"), replaces `assistant-launcher.tsx`
Keep the launcher's existing behaviour: `ASSISTANT_SURFACE_ID` layout morph into the panel, drag to dock via `nearestDock` / `AssistantDockTargets`, `launcherLabel` a11y text, ⌘J, `pendingCount` from `useAttentionSummary`, and `writingCount` from `useLiveReplyCount`. Replace only its look:
- A 38px pill (`radius 19`, `bg-popover`, Trenova's lift shadow), 18px from the corner.
- **Idle:** a 30px orb: a 2px conic ring using Desk's rainbow-ring stops, rotating every 9s, around a 4px foreground dot. On hover the ring grows 1px and the label slides out (`grid-template-columns 0fr → 1fr`, 380ms settle easing): "Ask {last agent}" plus `⌘` `J` Kbd.
- **Writing** (`writingCount > 0`): ring spins at 1.4s, dot scales to 0, label shows the current step (shimmer) plus elapsed seconds in mono. Multiple threads: "{n} writing".
- **Pending** (`pendingCount > 0`): ring and dot turn warning colour (dot breathes), label "**{n} change(s)** need your approval" plus a small "Review" pill. Pending leads when both apply, matching the current rule.
- **Replied while closed** (from `reply-ready.ts`): dot scales to 1.6, label "**{agent}** replied", until the panel is opened.
- Keep the hide affordance and `assistant-edge-tab.tsx`, restyled to the same orb.
- Reduced motion: no rotation, no breathing.

### 1.2 Panel layouts: changes to `assistant-widget.tsx`, `lib/assistant-dock.ts`, `assistant-store.ts`
Three layouts: the store's `expanded: boolean` becomes `layout: "compact" | "side" | "full"`. Migrate the persisted value: `expanded: true` → `"full"`.
- **Compact** (default): floating at the dock corner, 400px wide, radius 18. On home it's **auto height**; inside a thread or history it grows to `min(660px, 100dvh - 36px)` with a smooth height transition. Keep `AssistantResizeHandle` and saved `panelSize` working in this mode (the saved size becomes the thread height and width).
- **Side**: docked full height on the dock's side, 400px, no radius, a hairline on the inner edge. **The app shell's main content gets matching padding** (460ms settle transition) so the page reflows instead of being covered. Add the side option to `assistant-placement-menu.tsx`.
- **Full**: the existing expanded modal (keep the scrim, Esc to shrink, `aria-modal`), max `1080 × 760`, radius 18, grid `236px | 1fr`.

### 1.3 Header: replaces `assistant-header.tsx` in compact/side
A 46px bar: back chevron (in a thread or history), title (thread title or "Assistant"), then icon buttons: Conversations, New, Dock to side / Float, Full screen, Close. Keep the existing thread actions (delete, download transcript, **Open in Desk**) in a `⋯` menu next to the title. In full mode the header is the main column's 52px bar: sidebar toggle, centred title, Shrink, Close.

### 1.4 Home: replaces `assistant-home.tsx`
- **Compact/side**: heading "Ask about anything you can see" (18/600), the subline "Reads what's on your screen. Changes wait for your approval.", a **Recent** list (3 rows: `DeskAgentTile` 20px, title, mono relative time; "All conversations" opens history), then the composer at the bottom.
- **Full**: Desk's home: reuse `desk-home.tsx`'s greeting block (mono date line with a breathing dot, heading, subline) at 30px, with the composer centred.
- Both use **Desk's home composer treatment** (the slow rainbow ring plus `use-typewriter` presets: Tab to use, ⌘1–9 to ask). In compact/side the hint collapses to Kbd only and the suggestion ellipsizes.
- Agent choice comes from the composer's agent picker (`useAskableAgent`), not a separate grid.

### 1.5 History (compact/side): new view, built from existing data
The Conversations button swaps the body for a searchable list. Use `thread-grouping.ts` / `thread-rows.ts` / `useLiveThreadIds`: groups Waiting on you (pending approvals) / Today / Yesterday / This week; each row shows an 18px agent tile, title, "Agent · 13h ago". Back returns to the previous thread.

### 1.6 Full-screen sidebar: replaces `thread-sidebar.tsx`
Model it on `desk-rail.tsx`; **reuse its row, knob and search pieces** rather than writing new ones (extract them if they're private to the rail).
- Top: logo tile + "Assistant", New button (the + rotates 90° on hover).
- Search row ("Search · ⌘K") that becomes an input; Esc clears and closes it.
- Groups in mono 10.5px: Waiting on you (warning colour), Today, Yesterday, This week.
- 30px rows: an agent-accent dot, the title, and the time in mono shown only on hover/active; pending rows get a breathing warning dot.
- A **sliding knob** under the active row (`translateY` + `height`, 420ms).
- Footer: lock icon + "Read-only until you approve".
- ⌘\ or the header toggle collapses the column to 0 (460ms).

### 1.7 Compact fitting rules for shared Desk components
Apply these only under compact/side density:
- **Composer toolbar:** the page chip and model picker show their icon only (no label); the agent name truncates at 110px.
- **Composer popovers** (attach, agent, page, model, context): anchored above the composer at the composer's full width (`left:0; right:0; width:auto`). In full layout they use Desk's own sizes and positions.
- **Footnote popover:** clamp the 250px card 10px inside the scroll container horizontally, and flip it below the marker when less than 150px is free above.
- **Approval card actions** wrap to their own right-aligned row under a hairline.
- The composer textarea is 13.5px (Desk's is 14.5px), and the question and prose type steps down 1px.

### 1.8 Palette
The widget uses Trenova's **neutral** tokens (`tokens.css`, chroma 0) rather than Desk's hue-260 tint. Desk's shared components must read colours from CSS variables so the widget scope can override them; don't fork the CSS.

---

## 2. Reuse from Desk unchanged (move to shared, then mount)
Suggested home: `client/apps/web/src/components/desk-chat/` (or a similar shared folder). Desk then imports from there too. **Move** the files; don't copy them.

- Thread rendering: `desk-conversation.tsx`'s turn list, split into a `DeskThread` that both surfaces render (Desk-only pieces such as artifacts, handoff, schedules and pinned facts stay behind props or slots that the widget leaves off). Plus `desk-turns.tsx`, `desk-citations.tsx` (ledger footnotes, FnPop, "Why this step?"), `desk-web.tsx` (live web search, cites, sources), `desk-message-actions.tsx`, `desk-failures.tsx`, `desk-tool-failures.tsx`, `desk-poorly-read.ts`, `use-stick-to-bottom.ts`, `turn-status.ts`, `web-cites.ts`, `citations.ts`.
- Approvals: `desk-approval-card.tsx`, `desk-undo-bar.tsx`, `undo-window.ts`, `approval-facts.ts`, `desk-confetti.tsx` (optional).
- Compaction: `desk-compact-mark.tsx`, `desk-context-meter.tsx`.
- Composer: `desk-composer.tsx`, `desk-agent-picker.tsx`, `desk-model-picker.tsx`, `desk-brand-mark.tsx`, `desk-page-chip.tsx`, `desk-dictate.tsx`, `desk-uploads.tsx`, `desk-attachments.ts`, `desk-capture.tsx`, `desk-mentions.tsx`, `desk-slash.tsx`, `use-typewriter.ts` (or the new `hooks/use-typewriter.ts`, but pick one and delete the other).
- Shared bits: `desk-agent-tile.tsx`, `desk-icons.tsx`, `desk-countdown.tsx`, `desk-error-card.tsx`.
- Styles: split the conversation and composer rules out of `routes/desk/_styles/desk-v2.css` (304 KB) into a shared stylesheet imported by both. Compact overrides live in one small file scoped to the widget.

Behaviour that comes along for free and must match Desk exactly:
- **Questions:** short / mid / long sizes.
- **Replies:** word-by-word streaming in ledger mode (inline numbered footnotes, no step list).
- **While working:** the composer status line with the busy ring; the live web-search card is the only thing that appears in the thread.
- **Web search:** cite pills plus the sources list.
- **Message actions:** Copy and Read aloud.
- **Approvals:** the approval card docked above the composer → the confirmed state → the 5s undo bar → the "Approved" divider.
- **Compaction:** the context meter → the composer drain state and Cancel → the "Compacted" divider with its expandable summary.
- **Composer:** @mentions, /commands, uploads, Capture, dictation.

## 3. Delete after the refactor
These are widget-only duplicates; remove them along with their tests once the shared components cover their behaviour, and port any test cases that are still relevant to the shared components:
`message-thread.tsx`, `message-items.tsx`, `streaming-turn.tsx`, `tool-activity.tsx`, `composer.tsx` (keep exported types/helpers such as `ComposerAttachment` and `readyAttachments`, moving them into `desk-attachments.ts` or a `composer-types.ts`), `composer-hints.tsx`, `model-picker.tsx` (keep `model-switch.ts`), `approval-dock.tsx` (keep `approval-queue.ts` / `approval-actions.ts`), `thread-sidebar.tsx`, `assistant-home.tsx`, plus any widget-only CSS.
Before deleting, grep for other importers (e.g. `page-assistant.tsx`, record pages) and repoint them to the shared components.

## 4. Not in the widget
These Desk features aren't shown in the widget. Hide them with props or slots; don't fork the components:
- **Artifacts:** the workspace pane, artifact cards and auto-open (the "Open in Desk" action covers this).
- **Desk-only content:** receipt/narrated trace modes, pin as chapter, replay, the handoff menu, schedule cards, pinned facts.
- **Desk's other pages:** Watchtower, Decisions, Memory, Today, agent settings.
- **Desk's three-column layout:** the gutter/margin columns.
- Never in either: follow-up suggestion chips, the per-message agent/time row, the retry button.

## 5. Acceptance
- [ ] A **single implementation** of the thread, composer, pickers, approval card, undo bar, compaction and citations, imported by both `/desk` and the widget. No `assistant/*` file duplicates a `desk-chat/*` component.
- [ ] Desk looks and behaves identically after the move, and its existing tests pass.
- [ ] Beacon states: idle / hover / writing / pending / replied; drag-to-dock, hide plus edge tab, ⌘J, Esc, and the layout-morph open.
- [ ] Compact layout: auto height on home, grows into a thread, resizes. Side layout reflows the page. Full layout: the sidebar with the knob, ⌘K search and ⌘\ collapse.
- [ ] All composer menus fit inside a 400px panel, and the footnote cards never clip.
- [ ] The persisted store migrates `expanded` → `layout`.
- [ ] Light/dark, the neutral palette, reduced motion, and keyboard/a11y labels preserved.
- [ ] Tests: beacon state derivation, the layout migration, popover and footnote clamping, and `DeskThread` rendering in both densities.
