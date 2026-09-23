---
path: /desk/decisions
aliases: [agent approvals, AI proposals, approval queue, pending changes, human in the loop, review agent actions]
related:
  - /desk
  - /desk/watchtower
  - /admin/agent-control
---

## What it's for
Decisions is the queue of changes AI agents have proposed and that need a person to approve before they run, such as updating a record or a multi-step plan. Each proposal shows the agent, what it would change (**Exactly as proposed**), **Why** it wants to, how sure it is, the **Evidence** behind it, and for a plan the **Steps, in the order they run**. Changes that cannot be undone are marked **Permanent**. Dispatchers, billing staff and managers work this queue so agents can act on the operation safely.

## Tasks

### Approve or reject a proposed change
Keywords: approve AI action, deny agent change, review proposal
1. Open [Decisions](/desk/decisions) and select a proposal (or use j and k to move through the list).
2. Read **Why**, **Evidence** and **Exactly as proposed**.
3. Select **Approve** (a) or **Reject** (r). Enter a **Reason** (required when rejecting, optional when approving) and confirm with **Approve and run** or **Reject**.

### Change the values before approving
Keywords: edit proposal, adjust agent change, modify and approve
1. Open [Decisions](/desk/decisions) and select the proposal.
2. Select **Modify** (m), adjust the values, optionally add a reason, and confirm. The change runs with your values.

### Decide several proposals at once
Keywords: bulk approve, batch reject, approve all
1. Open [Decisions](/desk/decisions).
2. Mark proposals with their checkbox (or x). A batch holds one kind of change at a time, and a plan is always decided on its own.
3. Select **Approve all** or **Reject all**, enter one **Reason** for all of them (required when rejecting), and confirm. Approved changes run one after another; one that cannot run is reported on its own.

### Narrow the queue
Keywords: filter by agent, filter by change type
1. Open [Decisions](/desk/decisions).
2. Choose an agent (or **All agents**) and a kind of change (or **All changes**, or **Plans**). Select **Clear** to reset.

### See where a proposal came from
1. Open [Decisions](/desk/decisions) and select the proposal.
2. Select **Open the conversation** or **Open the run** to see the context the agent was working in.

## Notes
Needs read access to the assistant and to agent proposals. Approving, rejecting and modifying need update permission on agent proposals. A rejected change never runs.
