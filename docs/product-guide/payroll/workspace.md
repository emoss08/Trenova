---
path: /payroll/workspace
aliases: [settlement workspace, driver pay, run payroll, pay period, driver settlements, process settlements]
related:
  - /payroll/settlements
  - /payroll/settlement-batches
  - /payroll/pay-events
  - /payroll/deductions
  - /payroll/earnings
  - /payroll/advances
---

## What it's for
The settlement workspace is where payroll staff run the current driver pay period from one screen. The top strip shows the pay period and pay date and counts what is **In pipeline**, what **Needs review**, what is **Posted / paid**, the **Period net pay**, **Unsettled pay** not yet on a settlement, and pay events **On hold**.

Below it, the queue on the left lists every settlement in the period, the middle pane shows the selected settlement's earnings, deductions and history with the actions for its current status, and the right rail shows the selected driver's unsettled pay, recurring earnings, recurring deductions, advances and escrow.

## Tasks

### Generate the period's settlements
Keywords: build settlements, run payroll, create settlements
1. Open [Workspace](/payroll/workspace).
2. Select **Generate settlements**. It builds one draft settlement per driver from the unsettled pay events, pulling in recurring earnings, recurring deductions, advance recoveries and escrow.
3. Run it again after more pay accrues. Drivers who already have a settlement for the period are skipped, and their new accruals attach to their open draft automatically.

### Review and approve a settlement
Keywords: approve driver pay, submit settlement, post settlement
1. Open [Workspace](/payroll/workspace).
2. Find the driver in the queue with **Search driver or number**, or narrow it with the chips (**All**, **Needs review**, **Draft**, **Pending**, **Approved**, **Posted**, **Paid**).
3. Select the settlement. If it shows **Review required before approval**, read the listed exceptions first.
4. On a draft, select **Add adjustment** to add or deduct pay, **Recalculate** to rebuild it from current pay events, or remove a pay event line to return it to the unsettled pool.
5. Select **Submit for approval**, then **Approve** (or **Reject** with a reason to send it back to draft).
6. Select **Post to GL**, then **Mark paid** and choose the **Payment method** and an optional **Reference**.

### Act on several settlements at once
Keywords: bulk approve, bulk post, bulk mark paid
1. Open [Workspace](/payroll/workspace).
2. Tick the settlements in the queue, or tick the box above the list to select all visible ones.
3. Use **Submit**, **Approve**, **Post** or Mark Paid in the bar at the bottom of the queue. Each button shows how many selected settlements are in a status it applies to; the rest are skipped.

### Pay a driver off-cycle
Keywords: instant pay, pay now, early pay, settle one driver
1. Open [Workspace](/payroll/workspace).
2. Select **Pay now**, choose the **Driver**, and tick the loads to pay.
3. Decide whether to **Apply recurring deductions, escrow, and advance recovery**, choose the **Payment method**, and optionally enter a **Payment reference**.
4. Select **Pay now** to build, approve, post and mark the settlement paid in one pass.
5. To create an off-cycle draft instead, select the **Unsettled pay** tile and use **Settle now** next to the driver, or the generate-for-all button at the bottom of that dialog for everyone listed.

### Hold or attach a driver's unsettled pay
Keywords: hold pay event, defer pay, release hold
1. Open [Workspace](/payroll/workspace) and select a settlement for the driver.
2. In the **Unsettled pay** section of the right rail, select **Add to settlement** to attach a pay event to the selected draft.
3. Select **Hold** to defer a pay event, enter the reason, and select **Hold pay**. Held pay is skipped by settlement generation until you select **Release hold**.

### Void a settlement
1. Open [Workspace](/payroll/workspace) and select the settlement.
2. Select **Void**, enter the reason, and confirm with **Void**.

## Notes
Needs read access to driver settlements, and the page is only available when the organization has the Asset operations capability.

Voiding releases the settlement's pay events back to the unsettled pool and reverses any GL postings. Paid and voided settlements cannot be voided.

**Add adjustment** takes dollars, not cents: a positive amount adds pay and a negative amount deducts it.

To create or edit recurring earnings, recurring deductions or advances, use **Manage** in the right rail, which opens [Recurring earnings](/payroll/earnings), [Recurring deductions](/payroll/deductions) or [Pay advances](/payroll/advances). **View ledger** in the **Escrow** section opens [Escrow accounts](/payroll/escrow-accounts).
