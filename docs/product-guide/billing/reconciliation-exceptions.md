---
path: /billing/reconciliation-exceptions
aliases: [finance exceptions, adjustment exceptions, unapplied credit, replacement invoice review]
related:
  - /billing/pending-approvals
  - /billing/invoices
  - /billing/adjustment-batches
---

## What it's for
Reconciliation exceptions lists the follow-ups that invoice adjustments leave for finance to settle by hand, such as an unapplied customer credit or a replacement invoice that needs review. Each exception shows the original invoice, customer, reason and amount; a strip across the top counts open exceptions, pending approvals, write-offs and adjustment batches in flight.

Selecting an exception shows its status, amount, the adjustment's kind and status, who requested it and when, the policy source and any finance notes, the credit and rebill lines that created it, and links to every invoice and queue item involved. The page is for investigation: the fix itself happens on the linked invoice or queue item.

## Tasks

### Find open exceptions
Keywords: search exceptions, filter status, unresolved
1. Open [Reconciliation exceptions](/billing/reconciliation-exceptions).
2. Set the status filter to **Open** (or **Resolved** to look back), or leave it on **All statuses**.
3. Search by invoice number, customer or exception reason in the box above the list.

### Trace an exception back to its invoices
Keywords: source invoice, credit memo, rebill, investigate exception
1. Open [Reconciliation exceptions](/billing/reconciliation-exceptions) and select an exception.
2. Review its details and the **Adjustment detail** lines.
3. Under **Linked artifacts**, select **Original invoice**, **Credit memo** or **Replacement invoice** to open that invoice on [Invoices](/billing/invoices), or **Rebill queue item** to open the rebill in the [Billing queue](/billing/queue).

## Notes
Opening the page needs read access to invoices. Exceptions are created by adjustments submitted from [Invoices](/billing/invoices); adjustments still waiting for approval are on [Pending approvals](/billing/pending-approvals).
