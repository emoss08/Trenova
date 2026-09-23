---
path: /accounting/reconciliation/work-queue
aliases: [reconciliation queue, bank exceptions queue, receipt exceptions, reconciliation tasks]
related:
  - /accounting/reconciliation/bank-receipts
  - /accounting/reconciliation/summary
  - /accounting/reconciliation/import-batches
---

## What it's for
The bank receipt work queue holds one work item for every bank receipt that became an exception
on import, so someone owns it until it is settled. Each item moves from **Open** to **Assigned**
to **In review**, and ends **Resolved** or **Dismissed**. Figures at the top count **Open**,
**Assigned**, **In review** and **Exceptions**.

The list shows the items still being worked (open, assigned or in review), newest first, each
marked assigned or unassigned. Selecting an item shows the **Bank receipt info** (date, amount,
reference, memo, status and exception reason), the **Assignment**, and the **Actions** available
at its current step.
Accounting staff use it to divide up and close out reconciliation exceptions.

## Tasks

### Take and review a work item
Keywords: assign to me, claim exception, start review
1. Open [Bank receipt work queue](/accounting/reconciliation/work-queue) and select an **Open**
   item.
2. Select **Assign to me**.
3. When you are ready to work it, select **Start review**.

### Resolve or dismiss a work item
Keywords: close exception, mark false positive, external follow-up
1. Open [Bank receipt work queue](/accounting/reconciliation/work-queue) and select an item that
   is **In review**.
2. To resolve it, select **Resolve**, pick a resolution type (**Matched to payment**,
   **Marked false positive**, **Requires external follow-up** or **Superseded**), add notes, and
   select **Confirm resolution**.
3. To dismiss it instead, select **Dismiss**, give the reason, and select **Confirm dismiss**.

## Notes
- Opening the page needs read access to bank receipt work items; assigning, reviewing, resolving
  and dismissing need update access.
- Resolving or dismissing a work item does not match the bank receipt. To match it to a customer
  payment, use [Bank receipt reconciliation](/accounting/reconciliation/bank-receipts); matching
  there resolves the work item automatically.
- Resolved and dismissed items leave the list and have no further actions.
