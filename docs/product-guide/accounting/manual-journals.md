---
path: /accounting/manual-journals
aliases: [journal entry, JE, manual JE, adjusting entry, accrual, GL adjustment, general journal]
related:
  - /accounting/journal-reversals
  - /accounting/reports/trial-balance
  - /accounting/configuration-files/fiscal-years
---

## What it's for
Manual journals are general ledger entries that people key in by hand, such as accruals,
reclassifications and other adjustments that do not come from billing or payments. The page lists
every manual journal with its **Request #**, **Status**, **Description**, **Accounting date**,
**Total debit** and **Total credit**.

A manual journal moves through a review flow before it reaches the ledger: it starts as a
**Draft**, is submitted for approval (pending approval), is **Approved** or **Rejected**, and an
approved journal is then posted to the general ledger (**Posted**). Accountants prepare the
drafts; approvers review and post them.

## Tasks

### Create a manual journal draft
Keywords: new journal entry, add JE, accrual, book an adjustment
1. Open [Manual journals](/accounting/manual-journals).
2. Select the New button above the table. The **New manual journal** panel opens.
3. Under **Journal details**, fill in **Description** and **Accounting date**. Add a **Reason** for
   approvers if needed, check **Currency**, and optionally pick a **Fiscal year** and
   **Fiscal period**.
4. Under **Line items**, pick a **GL account** for each line and enter a **Debit** or **Credit**
   amount, with an optional **Memo**. Select **Add line** for more lines.
5. Make sure total debits equal total credits; the summary under the lines shows **Balanced** when
   they do.
6. Select **Create draft**.

### Edit and submit a draft for approval
Keywords: send for approval, submit journal
1. Open [Manual journals](/accounting/manual-journals).
2. Select the journal's row to open it.
3. Change any fields or lines, then select **Save draft**.
4. Select **Submit** to send the journal for approval. Its status becomes pending approval.

### Approve or reject a journal
Keywords: review journal, sign off, approval
1. Open [Manual journals](/accounting/manual-journals).
2. Select the row of a journal whose **Status** is pending approval.
3. Under **Actions**, select **Approve**, or select **Reject**, type the rejection reason and
   select **Confirm reject**.

### Post an approved journal to the general ledger
Keywords: post JE, post to ledger, book the entry
1. Open [Manual journals](/accounting/manual-journals).
2. Select an **Approved** journal's row.
3. Under **Actions**, select **Post to GL**. The journal's status becomes **Posted**.

### Cancel a journal
Keywords: void journal, withdraw journal, delete draft
1. Open [Manual journals](/accounting/manual-journals).
2. Select the journal's row. Only a draft, pending approval or approved journal can be cancelled.
3. Select **Cancel journal**, type the reason, then select **Confirm cancel**.

## Notes
Opening the page needs read access to manual journals.

The **Accounting date** must fall within an open fiscal period. Only a draft can be edited; once
submitted, the journal's fields are read-only. A posted journal cannot be cancelled or edited; to
undo its effect on the ledger, request a reversal on [Journal reversals](/accounting/journal-reversals).
