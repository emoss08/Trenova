---
path: /accounting/reconciliation/bank-receipts
aliases: [bank reconciliation, deposit matching, match deposits, unmatched receipts, cash reconciliation]
related:
  - /accounting/reconciliation/work-queue
  - /accounting/reconciliation/import-batches
  - /accounting/reconciliation/summary
  - /accounting/ar/payments
---

## What it's for
Bank receipt reconciliation is where imported bank deposits are matched to the customer payments
they paid for. The list on the left holds the receipts that could not be matched automatically
when they were imported (exceptions), newest first. Selecting one shows its **Receipt details**,
the **Exception reason**, and **Match suggestions**: posted customer payments that could be the
same money, each with a **Score** and a **Reason**.

Figures at the top count receipts that are **Imported**, **Matched** and in **Exceptions**, plus
**Active work items**. Accounting staff use the page to clear unmatched deposits.

## Tasks

### Match a bank receipt to a customer payment
Keywords: reconcile deposit, clear exception, match receipt
1. Open [Bank receipt reconciliation](/accounting/reconciliation/bank-receipts).
2. Select the receipt in the list.
3. Check the **Exception reason** and look through **Match suggestions**.
4. Select **Match** on the right payment. The receipt shows **Matched payment** with the
   payment ID and time.

### Handle a receipt with no suitable payment
Keywords: no match suggestions, missing payment, deposit without payment
1. Open [Bank receipt reconciliation](/accounting/reconciliation/bank-receipts) and select the
   receipt. If it shows **No match suggestions available.**, the payment may not be posted yet.
2. Post the payment on [Customer payments](/accounting/ar/payments) for the same amount, then
   come back and match it.
3. Or work the receipt from the [Work queue](/accounting/reconciliation/work-queue), where it can
   be resolved or dismissed with a note.

## Notes
- Opening the page needs read access to bank receipts; matching needs update access to bank
  receipts.
- Only a posted customer payment for exactly the receipt's amount can be matched, and a receipt
  that is already matched cannot be matched again.
- Matching a receipt also resolves its open item in the [Work queue](/accounting/reconciliation/work-queue).
- Receipts arrive here from [Import batches](/accounting/reconciliation/import-batches).
