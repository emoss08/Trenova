# The Desk as the operation's room

The Desk is one room: the rail, the canvas (Today, Needs you, a record's case, a line of Nova
handled, Triage) and Nova's margin, under one top bar. This page is how the parts that join them
work. The approved design and its prototype live in the git-ignored `design-plans/desk-workspace/`;
the surfaces copy its anatomy, and their styles live in `routes/desk/_styles/desk-room.css` under
the `dk-w-` prefix.

| Part | Where |
|---|---|
| The top bar's situation line, Search or do, Ask | `desk-topbar.tsx`, `desk-situation.tsx` |
| Nova's margin | `margin/desk-margin.tsx`, `stores/desk-margin-store.ts` |
| Needs you, a row, an item's detail, deciding in place | `deskattentionservice`, `deskAttention`, `attention/` |
| Nova handled and one line in full | `deskactivityservice`, `deskActivity`, `deskActivityEntry`, `today/`, `handled/` |
| A record's case | `record-case/`, `/desk/r/:kind/:id` |
| Labels under the reader's access | `deskrecords.Labeler` over `recordsummaryports.RecordSummarizer` |
| Triage in the room | `/desk/triage` renders `TriageRoom` (`routes/inbox/page.tsx`) |

## The room and Nova's margin

`.dsk` is a grid of rail, canvas and, while it is open, a 380px margin (`dk-w-mg-open`). The margin
holds one conversation (`DeskThread`, compact) or, before one exists, the Desk's box. Its state is
`useDeskMarginStore`, in memory only: which thread, the record it is about, the subject the open
page publishes (`setPageSubject`) and the cases opened this session (Open work in the rail). A
record's name is never written to browser storage, and the store is cleared with the session.

- **Asking opens the margin, never a page.** Today's box, the case's box, ⌘J and the top bar's
  **Ask** start a conversation in the margin (`DeskContext.startInMargin`). A question about a
  record carries it as a mention and binds the thread to the record's case
  (`useStartConversation`'s `StartBinding`: title, case subject, page context), so the margin
  reopens the same case conversation next time.
- **The rail opens conversations beside the work.** A plain click on a conversation row shows it in
  the margin; a modified click, or a click while a conversation fills the room, follows the link.
  **Open full width** goes to `/desk/t/:id`, which closes the margin on that thread.
- ⌘J toggles the margin on what the page is about; ⌘\ and Esc fold it.

## The situation line

Up to three of the morning briefing's first figures (section title, first item's value and path),
then **Needs you** (open items across lenses) and **Working**: `WatchtowerCounts.withAgents`, open
items handed to an agent that has not given them back (`agent_run_id` set, status In progress),
counted in the same pass as the queue's other totals.

## Needs you

One row per record, soonest deadline first, made of Watchtower items: the Watchtower already
projects proposals, plans and inbound messages as items, so grouping is a regrouping of one queue,
not a join of three. Items group on the record they are about ([exception-queue.md](exception-queue.md),
"What an item is about"); an item about nothing beyond itself, or about a kind with no permission
resource, is its own row with no record.

- **Two bounded reads, both through `WatchtowerService.List`**, so kind visibility, seen marks and
  cursors stay in one place. The first reads up to 200 open items (Open, Acknowledged, In progress)
  in the lens's kinds, narrowed by `kinds` and `severities` when the address names them (the
  briefing's links do), soonest first, and keeps the first `first` rows (at most 100). The second
  reads every open item about those rows' records in any lens (`AboutIDs`), so a row in the
  Decisions lens still shows its Watchtower condition and its email.
- **Lens counts** sum `WatchtowerCounts.ByKind` by lens.
- **Labels** come from the record summarizer, one batch per kind, and only for kinds the reader may
  read (`deskrecords.Labeler`); a record they may not read is counted, never named. A readable
  record with no label is treated as no record: it names nothing and opens no case.
- **The page** is two panes: lanes of rows on the left, the item on the right with its condition,
  the agent on it or the one that could take it, the message and the decision, and a sticky bar.
  `useWatchtowerActions` owns J/K/E/S/H, done, snooze and hand-off; the page adds 1–4 for the lens,
  Enter for the case and A/D for a decision, and refuses E and S on a decision.
- **Deciding in place** (`useItemDecision`) reads the item's proposal, or every step of its plan,
  and approves with `decideAgentProposal` / `decideAgentPlan` as the Decisions queue does, carrying
  the digest of every preview the server asks to be shown (`ReviewedChange`/`ReviewedPlan` with
  the approver scope). Nothing reads as done until the server answers; a refusal is shown in the
  server's words. Decline asks why and sends it as the decision's note.
- `/desk/watchtower` is gone: the Watchtower lens is it, and the old address redirects there.
  Links to one item are `/desk/needs?item=<id>`.

## Nova handled

What agents changed, read from `agent_proposals` itself rather than the AI audit trail: every write
an agent makes is a proposal, and one that ran on its own carries `AutonomyTier = AutoExecute`. The
read orders by `COALESCE(executed_at, simulated_at)` over `idx_agent_proposals_settled` and folds
consecutive writes of one tool in one run, or one plan's steps, into one line. The outcome is never
more than the record says: previewed (`simulated_at`), failed (`execution_error` or
`ExecutionFailed`), ran on its own (`AutoExecute`), else approved, with the decider read in one batch
from `agent_decisions` or the plan. Titles use the write's own `ExecutionResult` ("Filed 11 rate
confirmations") or the tool in words, never the tool's name. It needs read access to agent
proposals and reaches back at most 31 days.

`deskActivityEntry(id)` reads one line by the id of any write it folds: the run's settled proposals
(`ListByRun`, bounded by the run), folded the same way, the line that holds the id, every write it
folds with its record, and how often this agent made this write in the last 30 days
(`CountExecutedTool`). `/desk/handled/:id` draws it: the rule it ran under or who approved it, its
record, and what it changed.

## A record's case

`/desk/r/:kind/:id` reads `deskAttention` with `recordId` and draws the record as one decision: the
head with the soonest deadline, what the agent found (the decision's rationale) or what the
Watchtower raised, with numbered citations to an exhibit per open item down the side; a trust grid
(why it matters, what happens if nothing changes, what the agent proposes and why it asks); the
plan (`CasePlan`, the same `useItemDecision` as Needs you, ⌘Enter to approve, D to decline); the
record's registry card (`InPlaceRecord`); and a docked box whose questions open in the margin.
"What could happen", the prototype's alternative options, needs agents to draft alternatives,
which they do not yet.

## Keys

One map in `keybinds.config.ts`. ⌘K is the command palette everywhere, mounted in the Desk too; its
**Conversations** scope is Desk search (`assistant.deskSearch`). ⌘J toggles Nova's margin (in a
conversation it focuses its box). ⌘\ folds the margin, or toggles a conversation's workspace. G then
D, N, W or T goes to Today, Needs you, or Needs you on the Watchtower or Triage lens
(`useGoToChords`). A page's single keys fire only when no earlier handler took the key
(`defaultPrevented`); Needs you listens in the capture phase so a decision is never cleared by E or S.

## Verified outcomes

After an approved change runs, the follow-up turn tells the agent to set a wait for the moment a
change is meant to produce something (a delivery inside a new window, a document arriving), say
when it will look, and speak only if it did not happen (`decisionnotes.verifyOutcome`). The Desk
Bench reschedule scenario holds it (`writes.yaml`).
