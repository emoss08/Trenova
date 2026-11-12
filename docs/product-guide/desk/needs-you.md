---
path: /desk/needs
aliases: [needs you, what needs me, everything waiting, one list, lenses, watchtower, watch tower, watchtower lens, decisions lens, unified queue, alerts feed, exceptions feed, exception feed, exception queue, things to look at, operational alerts, AI monitoring, failed agent runs, agent exceptions, quarantined EDI, deadlines, approve in place]
related:
  - /desk
  - /desk/decisions
  - /desk/triage
  - /shipment-management/service-failures
---

## What it's for
Needs you is everything waiting on people across the operation, in one list with one row per record: a shipment's late warning from the Watchtower, the customer's email about it in Triage and the change an agent drafted to recover it in Decisions are one row. Open it from **Needs you** in the [Desk](/desk) rail, or press G then N.

The list down the left is grouped by how long is left before each row's soonest deadline: **Act now** (four hours or less, or already past), **Today**, **Coming up** and **For your information** (no deadline). Each row has a square coloured by its worst severity, the title of what is most urgent, the record's number and what it is, and under it one line for every signal about the record with the queue that raised it. On the right it shows how long is left (red inside four hours, amber inside a day) and where the work stands: **Needs your approval**, **Looking into it** while an agent works it, or **Didn't go through**. A blue dot marks a row you have not opened yet. Above the list, the bar shows how many items you have cleared today out of all of them.

**Everything**, **Watchtower** and **Decisions** above the list are lenses, not separate places: pick one (or press 1 to 3) to see only the rows with something in that queue. Each row still shows every signal about its record, a customer's email included. The Decisions lens links to [Decisions](/desk/decisions) for approving several changes at once. Mail has no lens here: a message shows on the row of the record it is about, and it is read, answered and filed in [Triage](/desk/triage), the one place mail is worked. Select **Open in Triage** on a message to go to it; answering, filing or ignoring it there clears it from this list.

The right side is the item in front of you: what it is and the record it is about, how long is left, the **Condition** the Watchtower raised (what is at stake, the facts and how it got here), the agent working it with its steps, or the agent that could take it, the **Message** Triage read, and for a change an agent wants to make, the **Decision**: what it would change, why the agent proposes it, whether it can be undone and why it asks. The bar at the bottom acts on it.

You only see the kinds of items whose records you are allowed to open, and a record you may not read is counted, never named. An item leaves the list when you mark it done, snooze it or decide it, or when the problem behind it clears. The numbers in the rail and in the bar across the top of the Desk are how many items wait in each lens.

## Tasks

### Work what needs you
Keywords: next item, keyboard, check alerts, what went wrong, exceptions today, open alerts
1. Open [Needs you](/desk/needs) from the rail, or press G then N.
2. Select a row, or press J (or the down arrow) for the next one and K (or the up arrow) for the previous one. Opening an item marks it seen, so its blue dot goes away.
3. Pick a lens above the list, or press 1 to 3, to see only one queue's rows.

### Mark an item done or snooze it
Keywords: clear alert, resolve, close alert, finished, remove from list, later, put aside, remind me tomorrow
1. Select the item in [Needs you](/desk/needs).
2. Select **Done**, or press E. It leaves the list for everyone and the next row takes its place; it comes back if the problem returns.
3. Select **Snooze**, or press S, to put it away until eight tomorrow morning in your time zone.
4. If the item changed while you had it open, it comes back as it is now. Look at it again before you clear it.

### Hand an item to an agent
Keywords: delegate to AI, let the agent handle it, assign to agent
1. Select the item in [Needs you](/desk/needs). When an agent can take it, the page names it.
2. Select **Hand off**, or press H, to give it to that agent. To choose another agent, select the arrow beside **Hand off** and pick one.
3. The agent starts at once. The row now says **Looking into it**, and the item shows the agent with a bar of its steps: reading the record, working out what to do and asking you to approve.

### Approve or decline a change in place
Keywords: approve agent change, decline agent change, review agent work, agent needs me
1. Pick the **Decisions** lens, or select a row that says **Needs your approval**.
2. Read the **Decision**: what it would change, why the agent proposes it and whether it can be undone. A change that has to be shown before it can be approved draws its preview under it.
3. Select **Approve**, or press A. Nothing is marked done until the server confirms it; if it does not go through, the page says why and that nothing changed.
4. Select **Decline**, or press D, and say why. The agent reads your reason in its conversation, and so does whoever looks after it.

### Open the case behind a row
Keywords: open case, open record, go to the shipment, view the source
1. Select the row in [Needs you](/desk/needs).
2. Select **Open the case**, or press Enter, to see everything open about the record as one decision.

### Ask Nova about an item
Keywords: investigate alert, explain this exception, ask AI about alert
1. Select the item in [Needs you](/desk/needs).
2. Select **Ask** at the top of the Desk, or press ⌘J (Ctrl+J). Nova's margin opens beside the list with the record attached to your question.

## Notes
- You need read access to the assistant to open the Desk, and read access to Watchtower to open this page. Each kind of item shows only if you can read that kind of record.
- Marking items done, snoozing them and handing them off need update access to Watchtower; approving and declining need update access to agent proposals.
- A deadline the record reports itself, such as a dock closing or a certificate lapsing, is the item's deadline. Otherwise it comes from the due times for each severity in dispatch controls.
- When nothing is left, the page says **Nothing here needs you**.
