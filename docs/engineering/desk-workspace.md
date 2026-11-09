# The Desk as the operation's room

The Desk is one room: Today, Needs you, a record's case, conversations and Triage, under one rail and
one strip. This page is how the parts that join them work. The design and its review live in the
git-ignored `design-plans/desk-workspace/`.

| Part | Where |
|---|---|
| Needs you, a record's row and its case's signals | `deskattentionservice`, `deskAttention` (GraphQL), `routes/desk/_components/attention/` |
| Nova handled | `deskactivityservice`, `deskActivity`, `routes/desk/_components/today/` |
| A record's case and Nova's margin | `routes/desk/_components/record-case/`, `/desk/r/:kind/:id` |
| Labels under the reader's access | `deskrecords.Labeler` over `recordsummaryports.RecordSummarizer` |
| Triage in the room | `/desk/triage` renders `TriageRoom` (`routes/inbox/page.tsx`) |

## Needs you

One row per record, soonest deadline first, made of Watchtower items: the Watchtower already
projects proposals, plans and inbound messages as items, so grouping is a regrouping of one queue,
not a join of three. Items group on the record they are about ([exception-queue.md](exception-queue.md),
"What an item is about"); an item about nothing beyond itself is its own row.

- **Two bounded reads, both through `WatchtowerService.List`**, so kind visibility, seen marks and
  cursors stay in one place. The first reads up to 200 open items (Open, Acknowledged, In progress)
  in the lens's kinds, soonest first, and keeps the first `first` rows (at most 100). The second
  reads every open item about those rows' records in any lens (`AboutIDs`), so a row in the
  Decisions lens still shows its Watchtower condition and its email.
- **Lens counts** sum `WatchtowerCounts.ByKind` by lens; they count items, which is what the rail's
  numbers have always counted.
- **Labels** come from the board's record summarizer, one batch per kind, and only for kinds the
  reader may read (`deskrecords.Labeler`); a record they may not read is counted, never named.
- **The client** reuses the Watchtower's machinery: `useWatchtowerActions` over the rows' items, the
  item card, and `WatchtowerItemBar` (shared with the Watchtower page; a decision opens on
  Decisions with **Review**, a message offers **Open in Triage**). `lib/desk-attention-cache.ts`
  takes an item out of every cached list when it leaves; the Watchtower realtime patcher and
  proposal and plan events refetch `deskAttention`.

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

## A record's case

`/desk/r/:kind/:id` draws the record with its registry card (`InPlaceRecord`) and beside it either
what is open about it (`deskAttention` with `recordId`, the Watchtower card and bar) or Nova. Nova's
margin is the Desk's own thread (`DeskThread`, compact): the person's case conversation about the
record when the thread list holds one, else a new one started on the first question with the record
as its subject (shipments, invoices and disputes; any other kind starts an unbound thread) and the
record attached as the page context of every message. ⌘J turns the margin to Nova.

## Keys

One map in `keybinds.config.ts`. ⌘K is the command palette everywhere, mounted in the Desk too; its
**Conversations** scope is Desk search (`assistant.deskSearch`), and on the Desk **All** lists the
top conversations as well. ⌘J asks Nova (the box in front of the person, the case's margin, or
Today). ⌘\ opens the workspace on everything a conversation made or folds it. G then D, N, W or T
goes to Today, Needs you, Watchtower or Triage (`useGoToChords`). A queue's single keys fire only
when no earlier handler took the key (`defaultPrevented`), so a chord's second key never also acts.

## Verified outcomes

After an approved change runs, the follow-up turn tells the agent to set a wait for the moment a
change is meant to produce something (a delivery inside a new window, a document arriving), say
when it will look, and speak only if it did not happen (`decisionnotes.verifyOutcome`). The Desk
Bench reschedule scenario holds it (`writes.yaml`).
