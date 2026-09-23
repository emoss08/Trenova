---
path: /billing/pending-approvals
aliases: [adjustment approvals, credit approvals, finance approval, approve credit memo, approve void]
related:
  - /billing/invoices
  - /billing/reconciliation-exceptions
  - /billing/adjustment-batches
---

## What it's for
Pending approvals lists invoice adjustments that the organization's adjustment policy holds for finance review before they change any money: credits, credit-and-rebills, full reversals (including voids of posted invoices) and write-offs. A strip across the top counts pending approvals, reconciliation exceptions, write-offs and batch failures.

Selecting an adjustment shows its financial impact (credit, rebill and net change), the reason and policy that required approval, who requested it and when, links to the invoices it touches, and the line-by-line charge detail. Finance approvers use it to approve or reject each request.

## Tasks

### Approve an adjustment
Keywords: approve credit, approve rebill, approve reversal, approve void
1. Open [Pending approvals](/billing/pending-approvals).
2. Select an adjustment in the list.
3. Review the **Financial impact**, the reason, and the **Charge detail** lines. Use the links under **Linked artifacts**, such as **Original invoice**, to open the invoices involved.
4. Under **Decision**, select **Approve**.

### Reject an adjustment
Keywords: decline credit, deny adjustment
1. Open [Pending approvals](/billing/pending-approvals) and select the adjustment.
2. Under **Decision**, select **Reject**.
3. Enter a **Rejection reason** and select **Confirm rejection**.

### Find a pending adjustment
Keywords: search approvals, filter adjustment type
1. Open [Pending approvals](/billing/pending-approvals).
2. Search by invoice, customer or reason in the box above the list.
3. Use the type filter (**All adjustment types** by default) to show only credits, credit-and-rebills, full reversals or write-offs.

## Notes
Opening the page needs read access to invoices. Adjustments arrive here when someone submits one from [Invoices](/billing/invoices) (with **Adjust invoice** or **Void invoice**) and the policy requires approval; approving it carries out the adjustment, while a rejected one makes no financial change.
