---
path: /desk/needs
aliases: [needs you, what needs me, everything waiting, one list, lenses, watchtower, watch tower, watchtower lens, decisions lens, unified queue, alerts feed, exceptions feed, exception feed, exception queue, things to look at, operational alerts, AI monitoring, failed agent runs, agent exceptions, quarantined EDI, deadlines, approve in place, agent approvals, AI proposals, approval queue, pending approvals, pending changes, proposed changes, agent proposals, approve agent change, decline agent change, agent plan approval, dry run, human in the loop, review agent actions, preview agent change, what will this change]
related:
  - /desk
  - /admin/agent-control
  - /desk/triage
  - /shipment-management/service-failures
---

## What it's for
Needs you is everything waiting on people across the operation, in one list with one row per record: a shipment's late warning from the Watchtower, the customer's email about it in Triage and the change an agent drafted to recover it in Decisions are one row. Open it from **Needs you** in the [Desk](/desk) rail, or press G then N.

The list down the left is grouped by how long is left before each row's soonest deadline: **Act now** (four hours or less, or already past), **Today**, **Coming up** and **For your information** (no deadline). Each row has a square coloured by its worst severity, the title of what is most urgent, the record's number and what it is, and under it one line for every signal about the record with the queue that raised it. On the right it shows how long is left (red inside four hours, amber inside a day) and where the work stands: **Needs your approval**, **Looking into it** while an agent works it, or **Didn't go through**. A blue dot marks a row you have not opened yet. Above the list, the bar shows how many items you have cleared today out of all of them.

**Everything**, **Watchtower** and **Decisions** above the list are lenses, not separate places: pick one (or press 1 to 3) to see only the rows with something in that queue. Each row still shows every signal about its record, a customer's email included. Mail has no lens here: a message shows on the row of the record it is about, and it is read, answered and filed in [Triage](/desk/triage), the one place mail is worked. Select **Open in Triage** on a message to go to it; answering, filing or ignoring it there clears it from this list.

The **Decisions** lens is where changes AI agents have proposed wait on a person before they run, such as updating shipments, posting invoices, sending a message, or a plan of several steps. It is the one place they are decided, one at a time or several together. Its list is grouped by the kind of change instead of by deadline, newest first, and the bar above it shows how many of today's are decided. Each row is named for what is its own: who a message goes to, the record a change touches, or how many records. A dot on the right marks a change with something to look at before you decide (amber) or one that can't run as it stands (red). The page beside the list shows one change at a time: which agent wants it and when, the agent's reason, whether it is **Reversible** or **Can't be undone**, anything to know before deciding, and then the change itself, worked out by the same code that would make it and read from your records as they are now. A message is drawn as it would go out. A line under it says whether it would go through as shown. The bar underneath approves or declines that one change. To decide several, tick them in the list. A change that shows on a record's row under **Everything** can be approved there too, and **Approve several at once in Decisions** opens it in this lens.

Under **Everything** and **Watchtower**, the right side is the item in front of you: what it is and the record it is about, how long is left, the **Condition** the Watchtower raised (what is at stake, the facts and how it got here), the agent working it with its steps, or the agent that could take it, the **Message** Triage read, and for a change an agent wants to make, the **Decision**: what it would change, why the agent proposes it, whether it can be undone and why it asks. The bar at the bottom acts on it.

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
1. Under **Everything** or **Watchtower**, select a row that says **Needs your approval**.
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

### Review what a change would do
Keywords: preview agent change, before and after, dry run, check proposed values, would it go through
1. Open [Needs you](/desk/needs), pick the **Decisions** lens (or press 3) and select a proposal in the list. A dot marks proposals you have not opened yet.
2. Read the change. When every record gets the same value, it is one line per record with the old value struck through and the new one after it. The first eight are shown, then how many more there are. Otherwise each record is listed with every value that changes. A message is shown as it would go out, with its recipients and subject. Money is shown line by line with the total before and after and the difference.
3. A value that moved since the agent proposed the change says "Was … when proposed". A value that may still move before the change runs says so.
4. Read the line under the change. It says the change would go through as shown, or how many records would go through and how many would be refused and why, or how many records changed since the agent drafted it.
5. A long reason from the agent is folded to its first lines. Select **Show more** to read all of it.

### Approve a proposed change
Keywords: approve AI action, accept agent change, approve proposal
1. Open [Needs you](/desk/needs), pick the **Decisions** lens (or press 3) and select the proposal.
2. Read what it changes. **Approve** stays off until that has loaded.
3. Select **Approve** (or press a). The change is approved at once and runs. The next proposal in the queue comes up.
4. If some records would be refused, the button names how many go through and how many are skipped. Approving runs the rest.
5. If the change moved while you were reading it, nothing is recorded. A notice says it changed since you opened it, and the change is shown again as it is now. Read it again and approve.
6. If the change was approved but could not be carried out, a message says it didn't go through and why, and it is marked that way under **Decided today**.

### Decline a proposed change
Keywords: reject agent change, deny proposal, turn down, decline reason
1. Open [Needs you](/desk/needs), pick the **Decisions** lens (or press 3) and select the proposal.
2. Select **Decline** (or press d or r).
3. In **Decline this change?**, enter a **Reason**. It is required, and both the agent and whoever looks after the agent see it.
4. Select **Decline** to confirm, or **Cancel**. A declined change never runs.

### Change the values before approving
Keywords: edit proposal, adjust agent change, modify and approve, approve with changes
1. Open [Needs you](/desk/needs), pick the **Decisions** lens (or press 3) and select the proposal.
2. Select **Change values** (or press m). It is offered for the open change when it has values you can edit, never for a plan.
3. Adjust the values. A value marked "Stays as proposed" cannot be changed: you can change what is done to that record, never which record it is.
4. Under the values, the dialog shows what your values would do. Values that would not go through say why, and the change cannot be approved with them.
5. Optionally add a **Reason**, then select **Approve with changes**. It stays off while **Nothing changed yet** is shown. The change runs with the values you saw previewed.

### Fix a change that would not go through
Keywords: would be refused, refused values, ask agent to fix, change refused value
1. Open [Needs you](/desk/needs), pick the **Decisions** lens (or press 3) and select the proposal. A change its own rules would refuse has a red dot on its row and lists each reason in a red block above the change. In the one-line-per-record view, records that would be refused are listed first and marked **Refused**.
2. Where you can edit the value a reason names, select the button next to the reason to open the values on it. Then approve with your changes.
3. Or select **Send back to the agent**, which takes the place of **Approve**. This opens the decline dialog with the reasons already written as the **Reason**. Decline it, and the agent learns what to fix and can propose it again.

### Decide several changes together
Keywords: bulk approve, batch decline, approve all like this, decide together, select all, tick changes
1. Open [Needs you](/desk/needs) and pick the **Decisions** lens (or press 3).
2. Point at a row in the list and tick the checkbox that takes the place of its agent's mark, or press x to tick the open change. Once one is ticked, every row shows its checkbox.
3. To tick a whole kind, point at its heading and select **Select all**. When some of the kind have a dot, the heading also offers to select only the ones with nothing flagged. Up to 50 changes can be decided together.
4. The box at the foot of the list shows how many are selected. It says when some have something to look at first, and when some can't be approved as they stand and will be left out. Select **Clear** or press Esc to untick them all.
5. To approve, select the button that reviews and approves them. One review opens: **The same for every one** shows what the changes share once, with a shared message as a template with numbered places, and each row shows what fills each place, the ones with something to look at first. A message worded differently is shown in full on its row. Select **Approve** with the count to approve them, or **Cancel**.
6. To decline, select **Decline** with the count in the box and enter one **Reason** for all of them. A change whose record moved since it was proposed can only be declined.
7. If some could not be approved, a message says how many went through and why the rest did not. Those stay in the queue. Changes you did not tick stay too.

### Approve or decline a plan
Keywords: multi-step plan, agent plan, plan steps
1. Open [Needs you](/desk/needs), pick the **Decisions** lens (or press 3) and select a plan. Its title shows how many steps it has.
2. Read each step in the order it runs. A step that works on a record an earlier step also changes says which step, and its values are shown as that step would leave them.
3. Select **Approve** or **Decline**. A plan is decided as one, and always on its own. It has no checkbox in the list, and its values cannot be changed. If a step would not go through, **Ask the agent to fix it** declines the plan with the reasons.

### Move through the decisions
Keywords: next proposal, previous proposal, keyboard shortcuts, decisions shortcuts
1. Open [Needs you](/desk/needs) and pick the **Decisions** lens (or press 3).
2. Use **Previous** and **Next** in the bar, or press k and j (or the up and down arrows). The bar shows where you are in the queue.
3. Shortcuts: a approves the open change, d or r declines it, m opens **Change values**, x ticks the open change, Esc unticks every change. Ticked changes are decided from the box at the foot of the list. The keys do nothing while you are typing or a dialog is open.
4. When more are waiting than the queue shows, select **Load more**. The queue also refreshes on its own every minute.

### See what was decided today
Keywords: decided today, approval history, who approved
1. Open [Needs you](/desk/needs) and pick the **Decisions** lens (or press 3).
2. Look under **Decided today** at the bottom of the list. Each change is marked approved, declined, or approved but didn't go through. Hover over one to see who decided it.
3. When nothing is waiting, the page says **Queue clear** and how many were decided today.

### See where a proposal came from
Keywords: agent conversation, agent run, source of proposal
1. Open [Needs you](/desk/needs), pick the **Decisions** lens (or press 3) and select the proposal.
2. Select **See the conversation** if an agent proposed it in a conversation, or **See the run** if it came from an agent's run.

### Decide a change in your own conversation instead
Keywords: approval card, approve in chat, not now, redraft
1. Open the conversation on the [Desk](/desk). A change waiting on you sits on a card above the message box.
2. Select **Approve**, **Review** to read it in full, or **Not now** to put it off. A change that is out of date offers **Redraft** or **Review** instead of **Approve**. A decision made there leaves this queue.

## Notes
- You need read access to the assistant to open the Desk, and read access to Watchtower or to agent proposals to open this page; you see the lenses you may read. Each kind of item shows only if you can read that kind of record.
- Marking items done, snoozing them and handing them off need update access to Watchtower; approving and declining need update access to agent proposals.
- A deadline the record reports itself, such as a dock closing or a certificate lapsing, is the item's deadline. Otherwise it comes from the due times for each severity in dispatch controls.
- When nothing is left, the page says **Nothing here needs you**.

- The Decisions lens needs read access to agent proposals; the other lenses need read access to the Watchtower. Approving, declining and changing values need update permission on agent proposals. Without it, the page says you can read these but not decide them.

An approval records exactly what you were shown, including when you decide several together. If what the changes would do cannot be loaded, a notice says so and offers **Try again**. You can still approve a change that stays inside your organization, and the decision is recorded as made without a preview. A change that moves money, reaches a customer or driver, or goes outside the organization waits until it can be read.

A value, record or amount you are not allowed to see reads **Hidden by your data access**, and the number of hidden parts is shown under the change. You can still approve it.

If a record the change depends on was edited or removed after the agent proposed it, the change is out of date. It may say **Changed since it was proposed**, and **Approve** stays off. It can only be declined. Ask the agent again for a fresh proposal.

Some changes cannot say exactly what they would do. They show the values they would run with under a warning instead. In a plan, a later step on a record an earlier step changed runs against what that step left. When every step of an approved plan changes a different record of the same kind, such as posting several invoices, each step runs whatever happens to the others. Otherwise the first step that fails stops the rest.
