# Watchtower redesign: notes

What the Desk v2 Watchtower handoff (`design_handoff_watchtower/`) became in the product. Where a
server rule or a product rule made the design impossible as drawn, the rule was kept and the
difference is recorded here. The engineering reference is
[engineering/exception-queue.md](engineering/exception-queue.md).

## Rules kept

### Hand off needs a record an agent can work on
The design offers **Hand off** on every unowned item. The server refuses to hand off an item
with no agent subject ("This item is not about a record an agent can work on"): an agent run is
always about a record. The split button is drawn only on items with a subject. On the others
the bar has Snooze and Done.

### Every item has a due time unless someone chose otherwise
The design's "For your information" lane holds items with no deadline. The exception queue
gives every new or reopened item a due time from the organization's per-severity minutes in
dispatch controls, so an item only has no due time when a person cleared it
(`setWatchtowerItemDue` with no time). That rule is kept. What changed is where a due time
comes from: a source that reports a real deadline (a dock closing, a certificate lapsing, free
time running out) now sets it, with its label, and only a person can move it. The lane fills
with items a person cleared, and in development with the seeded ones.

### Agent progress is the run's real progress
The design gives each hand-off its own list of four steps ("Read the certificate on file",
"Checked loads tendered…") that tick over every few seconds. A background agent run does not
plan named steps; what the server knows is the run's status. Progress is worked out from it:
reading the record (Pending, GatheringContext), working out what to do (Diagnosing), and
asking you to approve (AwaitingDecision). The bar therefore has three segments. A run that
finishes is **Done** with its summary as the note. The design's "finishes its last step →
needs you, Ready for you" maps to a run that ends with a proposal waiting on a person, which is
**needs you**. A run that fails also needs you, says why, and offers **See the run**.

### Review & approve goes through Decisions
In the prototype, **Review & approve** clears the item. Approving an agent's change sends the
digest of the preview the person saw, so a record that moved since is caught rather than
overwritten. That check lives in Decisions. The button opens Decisions on the run's waiting
change (`/desk/decisions?run=…`, which Decisions now honours), and the item is cleared with
Done once the work is through.

### Escalation does not move a source's deadline
Escalation used to push an overdue item's due time forward by the escalation interval. A
deadline the record reports does not move: the dock closes when it closes. So escalation
leaves it in place, which keeps "Detention is accruing · 51m running" true. Such an item
escalates again only after the escalation interval since the last escalation. Due times from
the policy or a person escalate as before.

### Snooze has a time
The design's Snooze takes no time. The server needs one (at most 30 days). S and the button
snooze until eight tomorrow morning in the reader's time zone, the same as Triage's S.

### Done is a resolution
Done resolves the exception with the code `Fixed`, so the history and the resolution reports
say what happened.

### No toast, and so no Undo
The handoff asks for an Undo toast after Done and Snooze. The product's rule is that an action
the person watches happen takes no toast, and the card leaving is that confirmation, so Done
and Snooze say nothing. Undo lived only in that toast, so it is gone with it. A resolved item
can still be reopened (`reopenWatchtowerItem`, and the agents' `exception_reopen`), and a
snoozed one woken through the API (`wakeWatchtowerItem`, which has no agent tool yet and is
`pending` in write coverage).

### Motion is transform and opacity only
The prototype animates the progress bar's width and the rings' stroke offset. The progress bar
now scales on `transform`. The deadline clock and the agent rings move a step at a time (once
a second, and whenever the run moves) with no transition. On a four-hour window a one-second
step is not visible as a jump. The card's exits and the breathing step are transform and
opacity, and reduced motion cuts all of them; under reduced motion the card leaves at once.

## What the old screen did that this one does not

The old feed's filter chips (severity and kind), its All / Mine / Unassigned / Overdue views,
the exception sheet (assign, acknowledge, start, change the due time, comment, history),
Dismiss, and the thumbs-up rating on agent-raised rows are not in the design and are gone from
the Desk. Every one of those writes is still in the GraphQL API and, where it was, in the agent
tools (`exception_assign`, `exception_set_due`, `exception_add_comment`, …), so an agent asked
from **Ask about this** can still do them. The links that used to open the feed filtered
(`?severities=`, `?kinds=`, `?item=`) now open the queue on the first item they name.

## Choices worth knowing

- **Where things live.** The record an item is about, its facts, what is at stake and its trail
  are stored on the item by the source that raised it, so the queue renders with no extra read
  per row. Dataloaders serve what is not the item's own: the agent run behind `agentWork`
  (`AgentRunByID`), the agent definitions (`AgentDefinitionByID`), the person who handed it off
  (`UserByID`) and the suggested agent (`SuggestedAgentByEventKind`).
- **Seen is per item.** Opening an item marks it seen for that reader
  (`watchtower_item_reads`). The cursor the old feed moved on every visit still counts:
  anything before it is seen too.
- **Counts follow the queue.** The rail's number, the Today page's "Critical in Watchtower"
  link and the briefing's Watchtower figures count open, not-snoozed items (`active`,
  `activeCritical`), so they match what the queue shows. The Today page had no Watchtower link
  before; it now has one under the ask box when anything is critical, and it lands on the first
  critical item.
- **Words from the server.** Kind names ("ETA at risk") and record nouns ("Shipment") come from
  the server in English, as the kind labels did before. Everything the client writes goes
  through `useT()`.
- **The "At stake" label** is small uppercase, as drawn. The Desk already labels its dry-run
  and plan callouts this way. That is a Desk convention outside the app shell's sentence-case
  rule.
- **The hand-off menu** is the shared dropdown menu, which floats inverted (dark in light mode)
  and fully opaque, as every menu in the product does, rather than the prototype's raised
  surface.
- **Migrations.** There is one migration set, applied to the development and test databases
  alike (`20261231009800_watchtower_queue`, up and down).
- **A bug found on the way.** `stringutils.Ellipsize` returned one character more than its limit
  once it added the ellipsis, so a 200-character title column refused an ellipsized title. It now
  stays within the limit.
