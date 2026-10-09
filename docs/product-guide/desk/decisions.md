---
path: /desk/decisions
aliases: [agent approvals, AI proposals, approval queue, pending approvals, pending changes, proposed changes, agent proposals, approve agent change, decline agent change, agent plan approval, dry run, human in the loop, review agent actions, preview agent change, what will this change]
related:
  - /desk
  - /desk/watchtower
  - /admin/agent-control
---

## What it's for
Decisions is the queue of changes AI agents have proposed and that wait on a person before they run, such as updating shipments, posting invoices, sending a message, or a plan of several steps. Open it from **Decisions** in the [Desk](/desk) sidebar, which shows how many are waiting. Dispatchers, billing staff and managers work it so agents only change the operation with a person's say-so.

The **Queue** on the left lists every waiting proposal, newest first, grouped by the kind of change, with how many of today's are decided. The middle shows one proposal at a time: which agent wants it and when, a one-line title with how many records it touches, the agent's reason, and what it would change. That is worked out by the same code that would make the change, read from your records as they are now. A **Dry run** line says whether it would go through as shown. Changes are marked **Reversible** or **Can't be undone**. The bar underneath approves or declines it.

## Tasks

### Review what a change would do
Keywords: preview agent change, before and after, dry run, check proposed values, would it go through
1. Open [Decisions](/desk/decisions) and select a proposal in the **Queue**. A dot marks proposals you have not opened yet.
2. Read the change. When every record gets the same value, it is one line per record with the old value struck through and the new one after it. The first five are shown, then how many more there are. Otherwise each record is listed with every value that changes. A message is shown as it would go out, with its recipients and subject. Money is shown line by line with the total before and after and the difference.
3. A value that moved since the agent proposed the change says "Was … when proposed". A value that may still move before the change runs says so.
4. Read the **Dry run** line. It says the change would go through as shown, or how many records would go through and how many would be refused and why, or how many records changed since the agent drafted it.
5. Select **Arguments the agent sent** to see exactly what the agent asked for.

### Approve a proposed change
Keywords: approve AI action, accept agent change, approve proposal
1. Open [Decisions](/desk/decisions) and select the proposal.
2. Read what it changes. **Approve** stays off until that has loaded.
3. Select **Approve** (or press a). The change is approved at once and runs. The next proposal in the queue comes up.
4. If some records would be refused, the button names how many go through and how many are skipped. Approving runs the rest.
5. If the change moved while you were reading it, nothing is recorded. A notice says it changed since you opened it, and the change is shown again as it is now. Read it again and approve.
6. If the change was approved but could not be carried out, a message says it didn't go through and why, and it is marked that way under **Decided today**.

### Decline a proposed change
Keywords: reject agent change, deny proposal, turn down, decline reason
1. Open [Decisions](/desk/decisions) and select the proposal.
2. Select **Decline** (or press d or r).
3. In **Decline this change?**, enter a **Reason**. It is required, and both the agent and whoever looks after the agent see it.
4. Select **Decline** to confirm, or **Cancel**. A declined change never runs.

### Change the values before approving
Keywords: edit proposal, adjust agent change, modify and approve, approve with changes
1. Open [Decisions](/desk/decisions) and select the proposal.
2. Select **Change values** (or press m). It is offered only for a single change that has values you can edit, not for a plan or a group decided together.
3. Adjust the values. A value marked "Stays as proposed" cannot be changed: you can change what is done to that record, never which record it is.
4. Under the values, the dialog shows what your values would do. Values that would not go through say why, and the change cannot be approved with them.
5. Optionally add a **Reason**, then select **Approve with changes**. It stays off while **Nothing changed yet** is shown. The change runs with the values you saw previewed.

### Fix a change that would not go through
Keywords: would be refused, refused values, ask agent to fix, change refused value
1. Open [Decisions](/desk/decisions) and select the proposal. A change its own rules would refuse shows **This would not go through as it stands**, reason by reason. In the one-line-per-record view, records that would be refused are listed first and marked **Refused**.
2. Where you can edit the value a reason names, select the button next to the reason to open the values on it. Then approve with your changes.
3. Or select **Ask the agent to fix it**. This opens the decline dialog with the reasons already written as the **Reason**. Decline it, and the agent learns what to fix and can propose it again.

### Decide every change like this one together
Keywords: bulk approve, batch decline, approve all like this, decide together
1. Open [Decisions](/desk/decisions) and select a proposal that has others of the same kind waiting. A line under it says how many more are waiting.
2. Select the link on that line to decide all of them together (or press x). The line then says how many you are deciding together and how many you have not opened yet. Select **Just this one** (or press x again) to go back.
3. Select **Approve** or **Decline**. Both buttons show the count. Declining asks for one **Reason** for all of them.
4. If some could not be approved, a message says how many went through and why the rest did not. Those stay in the queue.

### Approve or decline a plan
Keywords: multi-step plan, agent plan, plan steps
1. Open [Decisions](/desk/decisions) and select a plan. Its title shows how many steps it has.
2. Read each step in the order it runs. A step that works on a record an earlier step also changes says which step, and its values are shown as that step would leave them.
3. Select **Approve** or **Decline**. A plan is decided as one, and always on its own. It cannot be grouped with other proposals, and its values cannot be changed. If a step would not go through, **Ask the agent to fix it** declines the plan with the reasons.

### Move through the queue
Keywords: next proposal, previous proposal, keyboard shortcuts, decisions shortcuts
1. Open [Decisions](/desk/decisions).
2. Use **Previous** and **Next** in the bar, or press k and j (or the up and down arrows). The bar shows where you are in the queue.
3. Shortcuts: a approves, d or r declines, m opens **Change values**, x decides every change like this one together. They do nothing while you are typing or a dialog is open.
4. When more are waiting than the queue shows, select **Load more**. The queue also refreshes on its own every minute.

### See what was decided today
Keywords: decided today, approval history, who approved
1. Open [Decisions](/desk/decisions).
2. Look under **Decided today** at the bottom of the **Queue**. Each change is marked approved, declined, or approved but didn't go through. Hover over one to see who decided it.
3. When nothing is waiting, the page says **Queue clear** and how many were decided today.

### See where a proposal came from
Keywords: agent conversation, agent run, source of proposal
1. Open [Decisions](/desk/decisions) and select the proposal.
2. Select **See the conversation** if an agent proposed it in a conversation, or **See the run** if it came from an agent's run.

### Decide a change in your own conversation instead
Keywords: approval card, approve in chat, not now, redraft
1. Open the conversation on the [Desk](/desk). A change waiting on you sits on a card above the message box.
2. Select **Approve**, **Review** to read it in full, or **Not now** to put it off. A change that is out of date offers **Redraft** or **Review** instead of **Approve**. A decision made there leaves this queue.

## Notes
You need read access to the assistant and to agent proposals to open this page. Approving, declining and changing values need update permission on agent proposals. Without it, the page says you can read these but not decide them.

An approval records exactly what you were shown. When you decide several together, a proposal you never opened is recorded as approved without its preview having been reviewed. If what a change would do cannot be loaded, a notice says so and offers **Try again**. You can still approve it, and the decision is recorded as made without a preview.

A value, record or amount you are not allowed to see reads **Hidden by your data access**, and the number of hidden parts is shown under the change. You can still approve it.

If a record the change depends on was edited or removed after the agent proposed it, the change is out of date. It may say **Changed since it was proposed**, and **Approve** stays off. It can only be declined. Ask the agent again for a fresh proposal.

Some changes cannot say exactly what they would do. They show the values they would run with under a warning instead. In a plan, a later step on a record an earlier step changed runs against what that step left. When every step of an approved plan changes a different record of the same kind, such as posting several invoices, each step runs whatever happens to the others. Otherwise the first step that fails stops the rest.
