---
path: /admin/hold-reasons
aliases: [shipment holds, hold codes, hold types, on hold reasons]
related:
  - /shipment-management/shipments
  - /admin/shipment-controls
  - /admin/service-failure-reason-codes
---

## What it's for
Hold reasons are the reasons people choose when putting a shipment on hold. Each reason has a
**Hold type** (Operational, Compliance, Customer or Finance), a **Reason code**, a **Display name**,
a **Default severity** and gating rules that decide what the hold blocks by default: dispatch,
delivery and/or billing.

The table lists each reason with its **Active**, **Type**, **Code**, **Label**, **Description**,
**Default severity**, **Blocks dispatch**, **Blocks delivery** and **Blocks billing** columns.
Administrators maintain the list so holds are applied and reported consistently.

## Tasks

### Add a hold reason
Keywords: new hold code, create hold reason
1. Open [Hold reasons](/admin/hold-reasons).
2. Select **New hold reason**.
3. Choose the **Hold type**, enter a **Reason code** (for example ELD_OOS) and a **Display name**,
   and optionally **Details**.
4. Choose the **Default severity**: **Informational**, **Advisory** or **Blocking**.
5. Under **Gating rules**, turn on **Block dispatch**, **Block delivery** and/or **Block billing**,
   and **Visible to customer** if customers may see this reason.
6. Select **Save**.

### Edit or retire a hold reason
Keywords: change hold reason, deactivate hold reason
1. Open [Hold reasons](/admin/hold-reasons).
2. Select the row to open it.
3. Change the fields, or turn off **Active** so the reason is no longer offered.
4. Select **Save**.

## Notes
Viewing the page needs read access to hold reasons; **New hold reason** appears only for people who
can create them, and editing needs update access. The severity and gating rules are defaults applied
when someone picks the reason; they can be adjusted on each hold.
