---
path: /payroll/settlements
aliases: [driver settlement history, past settlements, pay statements, driver pay history, settlement lookup]
related:
  - /payroll/workspace
  - /payroll/settlement-batches
  - /payroll/disputes
  - /payroll/pay-events
---

## What it's for
Settlement history is the read-only record of every driver and owner-operator settlement across all pay periods. Payroll and accounting staff use it to look up a past settlement, see its earnings, deductions, net pay and history, and find settlements from earlier periods that are no longer in the workspace queue.

The table shows **Status**, **Settlement #**, **Driver**, **Type**, **Period end**, **Pay date**, **Loads**, **Gross**, **Deductions** and **Net pay**. Settlements are created in the workspace, not here.

## Tasks

### Look up a settlement
Keywords: find settlement, driver statement, search settlements
1. Open [Settlement history](/payroll/settlements).
2. Use the search box, or **Filter** and **Sort**, to narrow the list by driver, status, period or pay date.
3. Select a row to open the settlement. It shows the gross earnings, deductions, miles and loads, net pay, every line item and a **History** of when it was created, submitted, approved, posted, paid or voided.

### Work on a settlement that is still open
Keywords: edit settlement, process settlement
1. Open [Settlement history](/payroll/settlements) and select the settlement.
2. Select **Open in workspace** to process, adjust or pay it in the [Workspace](/payroll/workspace). Paid and voided settlements are final and have no workspace link.

### Move several settlements forward at once
Keywords: bulk approve, bulk post, bulk mark paid
1. Open [Settlement history](/payroll/settlements).
2. Tick the settlements to act on.
3. In the bar that appears, choose **Lifecycle action** and pick **Submit for approval**, **Approve** or **Post to GL**; settlements not in an eligible status are skipped.
4. To record payment, select **Mark paid**, choose the **Payment method**, optionally enter a **Batch reference**, and confirm with **Mark paid**. Only posted settlements can be marked paid.

## Notes
Needs read access to driver settlements, and the page is only available when the organization has the Asset operations capability.

The first time you open the page, a four-step explainer at the top describes how pay accrues, settlements build, exceptions are reviewed and settlements are posted and paid. Dismiss it with the close button; it stays hidden in that browser.
