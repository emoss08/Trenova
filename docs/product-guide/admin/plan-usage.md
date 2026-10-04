---
path: /admin/plan-usage
aliases: [free demo, trial, plan limits, usage limits, quota, subscription, read-only workspace, upgrade, billing plan]
related:
  - /admin/organization-settings
  - /onboarding
---

## What it's for
Plan & usage shows which Trenova Cloud plan the organization is on, where its free demo trial
stands, and how much of each limit it has used. It appears in Organization settings only on
Trenova Cloud; a self-hosted installation has no plan limits.

The strip at the top shows the **Plan**, its **Status** and, during the trial, the date the
**Trial ends** with the days left. Once the trial is over it shows the date the workspace is
**Deleted on**. The **Usage** panel lists every limit with a bar: records the organization holds
(shipments, customers, locations, workers, tractors, trailers, user seats, documents and document
storage), the largest single upload, and the assistant messages and AI spend that reset each
month. On the free demo a second panel, **Not included in the free demo**, lists what the demo
leaves out, such as outbound email, integrations, API keys and agent automation.

## Tasks

### Check how much of the free demo is used
Keywords: how many shipments left, usage, limit, quota, remaining
1. Open [Plan & usage](/admin/plan-usage).
2. Read the **Usage** panel. Each row shows what is used against the limit; a bar turns amber near
   the limit and red when it is reached.
3. Select **Refresh** to read the latest figures.

### Free room under a limit
Keywords: limit reached, cannot create, delete to free space
1. Open [Plan & usage](/admin/plan-usage) and find the limit that was reached.
2. Delete records of that kind that you no longer need, for example sample customers or
   locations. Limits on records count what the organization holds now, so each deletion frees a
   slot. Shipments are the exception: using up the shipment limit ends the trial, so deleting
   shipments afterwards does not reopen it.
3. Monthly limits such as **Assistant messages** and **AI spend** reset at the start of the next
   month instead.

### See when the trial ends
Keywords: trial days left, demo expiry, when does my trial end
1. Open [Plan & usage](/admin/plan-usage).
2. Read **Trial ends** in the strip at the top. The banner above every page also counts the days
   down. The trial ends on that date or as soon as every shipment in the plan has been used,
   whichever comes first.

## Notes
Opening the page needs read access to the organization.

When an action would go over a limit, or uses something the free demo leaves out, a dialog
explains which limit was reached; **View plan & usage** in that dialog opens this page. The free demo
trial lasts 7 days, and ends early once the organization has used its whole shipment allowance.
After the trial ends the workspace is read-only: everything can be opened and exported, but nothing can be
created or changed, and the workspace is deleted on the date shown. Paid plans with higher limits
are coming soon.
