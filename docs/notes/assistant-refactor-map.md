# Assistant refactor map

The audit behind the Assistant redesign (`design_handoff_assistant/`). Every component in
`client/apps/web/src/components/assistant/` that renders UI, the Desk component that already
does the same job, and what happens to it. Paths are relative to `client/apps/web/src/`.

Verdicts:

- **reuse Desk's** — the widget's copy is deleted and the Desk component, moved to
  `components/desk-chat/`, is mounted in its place.
- **keep (shell)** — the widget's own chrome; restyled or rebuilt for the new shell, not
  duplicated anywhere in the Desk.
- **keep (shared)** — already used by both surfaces, or by a surface that is neither; it stays.
- **delete** — nothing imports it once the duplicates are gone.

## Thread, composer and their parts

| Widget component | Desk equivalent | Verdict |
| --- | --- | --- |
| `message-thread.tsx` (thread, dock, composer wiring) | `desk-conversation.tsx` → `components/desk-chat/desk-thread.tsx` (`DeskThread`) | reuse Desk's |
| `message-items.tsx` (`UserTurn`, `AssistantTurn`, `AssistantProse`, chips, notices, `DayDivider`) | `desk-chat/conversation/desk-turns.tsx` (`DeskQuestion`, `DeskReply`, `DeskProse`), `desk-compact-mark.tsx`, `desk-failures.tsx` | reuse Desk's |
| `message-items.tsx` → `ReasoningDisclosure` | none (Desk never shows reasoning) | moved to `routes/agent-control/_components/activity/reasoning-disclosure.tsx`, its only reader |
| `streaming-turn.tsx` | `DeskStreamingReply` + composer status line (`turn-status.ts`) | reuse Desk's |
| `virtual-thread.tsx` | `use-stick-to-bottom.ts` + the Desk's `content-visibility` rows | reuse Desk's |
| `voice/desk-thinking.tsx` | composer busy ring + `composerStatus` | reuse Desk's |
| `composer.tsx` (42 KB) | `desk-chat/composer/desk-composer.tsx` | reuse Desk's; `ComposerAttachment`, `ComposerPayload`, `readyAttachments`, `activeMentions` move to `components/assistant/composer-types.ts` |
| `composer-hints.tsx` | the composer's typewriter hint (`use-typewriter.ts`) and `dk-hint` line | reuse Desk's |
| `dictation-control.tsx` | `desk-chat/composer/desk-dictate.tsx` | reuse Desk's |
| `model-picker.tsx` | `desk-chat/composer/desk-model-picker.tsx` | reuse Desk's (`model-switch.ts` stays) |
| `agent-starters.tsx` | the home composer's starter presets (`DeskHomeComposer`) | reuse Desk's |
| `agent-ask.tsx` | `desk-chat/composer/desk-agent-picker.tsx` via `useAskableAgent` | reuse Desk's |
| `approval-dock.tsx` (41 KB) | `desk-chat/conversation/desk-approval-card.tsx`, `desk-undo-bar.tsx` | reuse Desk's (`approval-queue.ts`, `approval-actions.ts` stay) |
| `decision-record.tsx`, `decision-outcomes.tsx` | `DeskApprovedCard` and the "Approved" divider | reuse Desk's |
| `deferred-decisions-pill.tsx` | the dock's "N changes wait on you" pill in `DeskThread` | reuse Desk's |
| `read-only-thread-notice.tsx` | the composer `lock` line (`desk-locks.tsx`) | reuse Desk's |
| `floating-slot.tsx` | none needed: the Desk dock stacks its cards | delete |
| `web-citations.tsx` | `desk-chat/conversation/desk-web.tsx` for replies | keep (shared) for `tool-activity.tsx` only |
| `tool-activity.tsx`, `delegate-step.tsx`, `decision-chrome.tsx`, `voice/agent-gutter.tsx` | none: the Desk shows a ledger, never a step list | keep (shared): the agent-control run transcript draws them |
| `choice-prompt.tsx`, `report-run-card.tsx`, `proposal-editor.tsx`, `record-subset-field.tsx`, `proposal-preview/*`, `display-value.tsx`, `voice/artifact-chrome.tsx`, `voice/working-dot.tsx`, `artifact-opener.tsx`, `decision-follow-up.tsx`, `outside-content-badge.tsx` | already Desk's own building blocks | keep (shared) |
| `agent-picker.tsx` | — (an admin form control, not chat) | keep (shared) |
| `page-assistant.tsx` | — (binds a thread to a page) | keep (shell); now renders `DeskThread` |

## Widget shell

| Widget component | Desk equivalent | Verdict |
| --- | --- | --- |
| `assistant-widget.tsx` | — | keep (shell): three layouts (`compact` / `side` / `full`) |
| `assistant-panel.tsx` | — | keep (shell): header, home, history, thread; mounts `DeskThread` |
| `assistant-launcher.tsx` | — | keep (shell): restyled as the beacon; drag, dock, `layoutId`, a11y kept |
| `assistant-edge-tab.tsx` | — | keep (shell): restyled to the orb |
| `assistant-header.tsx` | `desk-topbar.tsx` (thread actions) | keep (shell): rebuilt as the 46px bar with a `⋯` menu |
| `assistant-home.tsx` | `desk-home.tsx` (greeting block, home composer) | delete; the new home reuses `DeskHomeComposer` and `DeskGreeting` |
| `thread-sidebar.tsx` | `desk-rail.tsx` | delete; the full-screen sidebar reuses the rail's knob, row and shelf pieces (`desk-chat/rail/*`) |
| `live-reply-label.tsx` | the rail's state dot (`deskThreadState`) | delete |
| `assistant-placement-menu.tsx` | — | keep (shell): gains the side layout |
| `assistant-resize-handle.tsx`, `assistant-dock-targets.tsx`, `assistant-mark.tsx`, `assistant-surface.tsx` | — | keep (shell) |

## Importers of the files being deleted

| File | Importers outside the deleted set | Repointed to |
| --- | --- | --- |
| `message-thread.tsx` | `page-assistant.tsx`; `routes/formula-template/.../formula-assistant-sheet.tsx`, `formula-studio.tsx` (types `PageBinding`, `PageRequest`); `routes/shipment/.../ai-activity-panel.tsx` (types) | `DeskThread`; types from `use-thread-model.ts` |
| `message-items.tsx` | `components/command-palette/ask-answer-card.tsx` (`AssistantProse`); `routes/agent-control/.../run-transcript-view.tsx` (`AssistantProse`, `ReasoningDisclosure`) | `DeskProse`; `reasoning-disclosure.tsx` |
| `composer.tsx` | `use-thread-model.ts`, `use-composer-context.ts`, the Desk composer files (types and `readyAttachments`, `activeMentions`, `MAX_ATTACHMENTS`) | `composer-types.ts` |
| `streaming-turn.tsx`, `composer-hints.tsx`, `model-picker.tsx`, `approval-dock.tsx`, `virtual-thread.tsx`, `read-only-thread-notice.tsx`, `deferred-decisions-pill.tsx`, `agent-starters.tsx`, `agent-ask.tsx`, `decision-record.tsx`, `decision-outcomes.tsx`, `dictation-control.tsx`, `floating-slot.tsx`, `voice/desk-thinking.tsx` | none outside the deleted set and their tests | — |
| `thread-sidebar.tsx`, `assistant-home.tsx`, `live-reply-label.tsx` | `assistant-panel.tsx`, `assistant-header.tsx` | the new shell |
