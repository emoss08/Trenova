---
path: /accounting/reconciliation/summary
aliases: [bank reconciliation dashboard, reconciliation overview, match rate, exception aging]
related:
  - /accounting/reconciliation/bank-receipts
  - /accounting/reconciliation/work-queue
  - /accounting/reconciliation/import-batches
---

## What it's for
The reconciliation summary is a read-only overview of bank receipt reconciliation. It shows how
many receipts are **Imported**, **Matched** and in **Exceptions** with their amounts, and the
**Match rate** (matched receipts as a share of imported ones). **Exception aging** counts
exceptions as **Current**, **1-3 Days**, **4-7 Days** and **7+ Days** old, and **Work items**
counts the **Active**, **Assigned** and **In review** items in the work queue.

Accounting managers use it to see at a glance whether deposits are being matched and how long
exceptions have been waiting.

## Tasks

### Check reconciliation status
Keywords: how many exceptions, unmatched deposits, reconciliation progress
1. Open [Reconciliation summary](/accounting/reconciliation/summary).
2. Read the totals at the top and the **Exception aging** table.
3. Select **Bank receipts** to match receipts, or **Work queue** to work the open items.

### Start reconciling for the first time
Keywords: no receipts yet, import deposits
1. Open [Reconciliation summary](/accounting/reconciliation/summary). Until anything has been
   imported it shows **Nothing to reconcile yet**.
2. Select **Import a batch** to go to [Import batches](/accounting/reconciliation/import-batches).

## Notes
Opening the page needs read access to bank receipts. Nothing can be changed here.
