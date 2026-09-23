---
path: /billing/adjustment-batches
aliases: [bulk adjustments, adjustment batches, bulk credit progress, batch failures]
related:
  - /billing/pending-approvals
  - /billing/reconciliation-exceptions
  - /billing/invoices
---

## What it's for
Batch monitor tracks bulk invoice adjustment submissions: how far each batch has got, which items failed and why, and what each item created. The list shows every batch with its submitter and counts of total, done, failed and pending items; a strip across the top counts batches in flight, failed items, approvals pending and write-offs.

Selecting a batch shows its totals, status, submission time, last failure count and idempotency key, and an **Item results** table with each invoice's outcome, failure reason and links to the invoice and adjustment it produced. Finance and billing staff use it to follow up on bulk corrections that did not fully succeed.

## Tasks

### Check how a batch ran
Keywords: batch progress, bulk adjustment status, failed items
1. Open [Batch monitor](/billing/adjustment-batches).
2. Select a batch in the list.
3. Read the totals for succeeded, failed and pending items, then scan **Item results** for the failure reason on each failed invoice.

### Find a batch
Keywords: search batch, filter batch status, idempotency key
1. Open [Batch monitor](/billing/adjustment-batches).
2. Search by batch ID, submitter or idempotency key in the box above the list.
3. Filter by status: **Queued**, **Submitted**, **Running**, **Completed**, **Failed** or **Partial**.

### Open what a batch item created
Keywords: batch invoice, batch adjustment
1. Open [Batch monitor](/billing/adjustment-batches) and select the batch.
2. In **Item results**, select **Invoice** to open the invoice on [Invoices](/billing/invoices), or **Adjustment** to open the adjustment on [Pending approvals](/billing/pending-approvals).

## Notes
Opening the page needs read access to invoices. The page only monitors batches; it does not submit or retry them.
