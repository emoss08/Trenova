---
path: /desk/decisions
aliases: [agent approvals, AI proposals, approval queue, pending changes, human in the loop, review agent actions, preview agent change, what will this change]
related:
  - /desk
  - /desk/watchtower
  - /admin/agent-control
---

## What it's for
Decisions is the queue of changes AI agents have proposed and that need a person to approve before they run, such as updating a record or a multi-step plan. Each proposal leads with **What changes**: every record the change would touch, what happens to it, and each value before and after, worked out by the same code that would make the change and read from your records as they are now. A message the change would send is shown as it would go out, with the recipients it would really reach, and amounts it would move are shown line by line with the totals before and after. Below that are **Why** the agent wants it, how sure it is, the **Evidence** behind it, and under **Details** the arguments **Exactly as proposed**. A plan shows **What changes, in the order it runs**, step by step. Changes that cannot be undone are marked **Permanent**. Dispatchers, billing staff and managers work this queue so agents can act on the operation safely.

## Tasks

### Review what a change would do before approving it
Keywords: preview agent change, what will this change, before and after, check proposed values
1. Open [Decisions](/desk/decisions) and select a proposal (or use j and k to move through the list; the next row is read ahead so it is ready when you reach it).
2. Read **What changes**. The old value is struck through with the new one after it; a value that has moved since the agent proposed the change says what it was then ("Was … when proposed").
3. A value, record or amount you are not allowed to see reads **Hidden by your data access**, and the number of hidden parts is shown under the change. You can still approve it; the decision records that parts were hidden from you.
4. If a record the change depends on was edited or removed after the agent proposed it, the change says **Changed since it was proposed** and can only be rejected; ask the agent again for a fresh proposal.

### Approve or reject a proposed change
Keywords: approve AI action, deny agent change, review proposal
1. Open [Decisions](/desk/decisions) and select a proposal.
2. Read **What changes**, **Why** and **Evidence**. **Approve** stays off until what the change would do has loaded.
3. Select **Approve** (a) or **Reject** (r). The dialog shows what changes again; enter a **Reason** (required when rejecting, optional when approving) and confirm with **Approve and run** or **Reject**.
4. If what the change would do moved while you were reading it, nothing is recorded: the dialog shows the change as it is now and says it looks different. Read it again and confirm.

### Change the values before approving
Keywords: edit proposal, adjust agent change, modify and approve
1. Open [Decisions](/desk/decisions) and select the proposal.
2. Select **Modify** (m) and adjust the values. The record the change is about stays as proposed and cannot be edited; a change can alter what is done to that record, never which record it is.
3. As you type, the dialog shows what your values would do. Values the change would refuse say why and cannot be approved.
4. Optionally add a reason and confirm with **Approve with changes**. The change runs with the values you saw previewed.

### Decide several proposals at once
Keywords: bulk approve, batch reject, approve all
1. Open [Decisions](/desk/decisions).
2. Mark proposals with their checkbox (or x). A batch holds one kind of change at a time, and a plan is always decided on its own.
3. Select **Approve all** or **Reject all**, enter one **Reason** for all of them (required when rejecting), and confirm. Approved changes run one after another; one that cannot run, or whose preview changed since you read it, is reported on its own.

### Narrow the queue
Keywords: filter by agent, filter by change type
1. Open [Decisions](/desk/decisions).
2. Choose an agent (or **All agents**) and a kind of change (or **All changes**, or **Plans**). Select **Clear** to reset.

### See where a proposal came from
1. Open [Decisions](/desk/decisions) and select the proposal.
2. Select **Open the conversation** or **Open the run** to see the context the agent was working in.

## Notes
Needs read access to the assistant and to agent proposals. Approving, rejecting and modifying need update permission on agent proposals. A rejected change never runs.

An approval records exactly what you were shown. In a batch, a proposal you never opened is recorded as approved without its preview having been reviewed, and the confirmation says how many of those there are. Some changes cannot say exactly what they would do; they show the values they would run with under a warning instead. If what a change would do cannot be loaded at all, you can still approve it, and the decision is recorded as made without a preview.

In a plan, a step that changes a record an earlier step also changes is shown as that earlier step would leave it, with a note naming the step.
