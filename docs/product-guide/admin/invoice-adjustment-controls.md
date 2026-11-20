---
path: /admin/invoice-adjustment-controls
aliases: [credit memo rules, rebill policy, write-off approval, adjustment approval, credit and rebill settings]
related:
  - /billing/invoices
  - /billing/adjustment-batches
  - /billing/pending-approvals
  - /admin/billing-controls
---

## What it's for
Invoice adjustment controls set the organization's rules for credits, rebills and write-offs on
invoices. It is one settings form, saved once, in tabs: **Eligibility** (which invoices may be
adjusted and which accounting period an adjustment books in), **Approval** (whether a reason is
required, when adjustments and write-offs wait for approval, and whether someone else must
approve) and **Credits & rebills** (unapplied customer credit, review of replacement invoices and
what customers see after an invoice is replaced).

Billing and finance managers use it to decide how much control to put around changing invoices that
have already gone out. When a save is refused, the tab holding the setting to fix opens and is
marked.

## Tasks

### Decide which invoices can be adjusted
Keywords: adjust paid invoice, disputed invoice, partially paid
1. Open [Invoice adjustment controls](/admin/invoice-adjustment-controls).
2. On **Eligibility**, under **Invoices that can be adjusted**, set **Partly paid invoices**,
   **Paid invoices** and **Disputed invoices** to **Not allowed** or **With approval**. Adjusting
   an invoice that has been paid or disputed always needs approval.
3. Under **Accounting date**, set **Book adjustments on** to **Invoice date while its period is
   open** or **Next open period**. With the invoice date, also set **When the period is closed**:
   **Not allowed**, **Reopen the period first** or **Next open period, with approval**.
4. Select **Save changes**.

### Require approval for adjustments and write-offs
Keywords: approval threshold, write-off limit, adjustment approval
1. Open [Invoice adjustment controls](/admin/invoice-adjustment-controls).
2. On **Approval**, set **Approval** to **Never**, **Always** or **From an amount**. With **From
   an amount**, enter the **Approval amount**: adjustments of that amount or more wait for
   approval. **Never** cannot be chosen while closed-period adjustments post in the next open
   period with approval.
3. Under **Write-offs**, set **Write-offs** to **Not allowed**, **Always with approval** or
   **Approval from an amount**, and enter its **Approval amount** if asked.
4. Turn on **Require a separate approver** so the person who submits an adjustment or credit memo
   cannot approve or post it themselves.
5. Select **Save changes**.

### Require a reason on every adjustment
Keywords: adjustment reason, justification
1. Open [Invoice adjustment controls](/admin/invoice-adjustment-controls).
2. On **Approval**, set **Reason** to **Required**.
3. Select **Save changes**.

### Control customer credits and replaced invoices
Keywords: unapplied credit, over-credit, show old invoice to customer, rerate tolerance
1. Open [Invoice adjustment controls](/admin/invoice-adjustment-controls).
2. On **Credits & rebills**, set **Unapplied credit**. When it is **Allowed**, also set **Credit
   beyond what is owed** to **Not allowed** or **With approval**.
3. Under **Rebills**, set **Review replacement invoices** to **Never**, **When the amount
   changes** or **Always**. With **When the amount changes**, set the **Rerate tolerance**: a
   rerated invoice that differs from the original by more than that percentage is reviewed.
4. Set **Replaced invoices** to **Current invoice only** or **Current and replaced invoices**.
   With **Current invoice only**, an invoice a rebill replaced cannot be emailed, its emailed
   document links stop working, and a statement leaves it out, with the credit memo that
   reversed it, when both fall in the statement's period. Your team still sees every invoice.
5. Select **Save changes**.

## Notes
Opening the page needs read access to invoice adjustment control; saving needs update access.
Adjustments that need approval wait on [Pending approvals](/billing/pending-approvals).
