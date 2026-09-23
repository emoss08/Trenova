---
path: /accounting/reconciliation/import-batches
aliases: [bank deposits import, bank statement import, deposit batch, bank file, lockbox import]
related:
  - /accounting/reconciliation/bank-receipts
  - /accounting/reconciliation/work-queue
  - /accounting/reconciliation/summary
covers:
  - /accounting/reconciliation/import-batches/:batchId
---

## What it's for
Import batches is where bank receipts enter reconciliation. Each batch is a set of deposits from
one bank, entered line by line, and the list shows its **Reference**, **Source**, **Status**,
**Imported**, **Matched**, **Exceptions**, **Total amount** and **Created** time. Figures at the
top give **Total batches**, **Processing** and **Completed**.

As a batch is saved, each receipt is checked against posted customer payments. A receipt with a
reference number that points to exactly one payment of the same amount is matched on the spot;
the rest become exceptions to work on [Bank receipts](/accounting/reconciliation/bank-receipts)
and the [Work queue](/accounting/reconciliation/work-queue). Accounting staff use this page to
bring deposits in and see how many matched.

## Tasks

### Import a batch of bank receipts
Keywords: enter deposits, add bank receipts, new batch, import bank file
1. Open [Import batches](/accounting/reconciliation/import-batches).
2. Select **Import batch** (or **Import a batch** when there are none yet).
3. Fill in **Source** (the bank) and **Reference**, such as a statement number.
4. Under **Receipt lines**, enter each deposit's **Date**, **Amount ($)**, **Reference #** (the
   check number or transaction ID) and an optional **Memo**. Select **Add line** for more
   deposits, or the bin icon to remove one.
5. Select **Import batch**, or press Ctrl+Enter.

### See what a batch matched
Keywords: batch detail, batch results, unmatched deposits
1. Open [Import batches](/accounting/reconciliation/import-batches).
2. Select the batch row. Its page shows **Imported**, **Matched** and **Exceptions** totals, the
   **Batch info**, and every receipt under **Receipts** with its status.
3. Select **Back to batches** to return to the list.

## Notes
- Opening the page needs read access to bank receipts; importing needs create access to bank
  receipts.
- A batch needs at least one receipt line.
- Receipts with no **Reference #** cannot be matched automatically and always become exceptions.
- Automatic matching only runs when reconciliation is turned on in the organization's accounting
  control settings. When it is off, receipts stay imported and unmatched.
