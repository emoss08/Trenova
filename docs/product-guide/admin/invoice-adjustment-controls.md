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
Invoice adjustment controls set the organization's policy for credits, rebills and write-offs on
invoices. The page is one settings form in four cards: **Eligibility policy** (which invoices may be
adjusted and which accounting date adjustments use), **Documentation requirements**, **Approval
policy** (when adjustments and write-offs need approval) and **Credit and visibility** (customer
credit balances and what customers see after an invoice is replaced).

Billing and finance managers use it to decide how much control to put around changing invoices that
have already gone out.

## Tasks

### Decide which invoices can be adjusted
Keywords: adjust paid invoice, disputed invoice, partially paid
1. Open [Invoice adjustment controls](/admin/invoice-adjustment-controls).
2. In **Eligibility policy**, set the **Partially paid invoice adjustment policy**, **Paid invoice
   adjustment policy** and **Disputed invoice adjustment policy** to **Disallow**, **Allow with
   approval** or **Allow without approval**.
3. Set the **Adjustment accounting date policy** and **Closed period adjustment policy**.
4. Set the **Rerate variance tolerance percent** and **Replacement invoice review policy** for
   credit-and-rebill work.
5. Select **Save changes**.

### Require approval for adjustments and write-offs
Keywords: approval threshold, write-off limit, adjustment approval
1. Open [Invoice adjustment controls](/admin/invoice-adjustment-controls).
2. In **Approval policy**, set the **Standard adjustment approval policy** to **None**, **Always**
   or **Amount threshold**. With **Amount threshold**, enter the **Standard adjustment approval
   threshold**.
3. Set the **Write-off approval policy** to **Disallow**, **Always require approval** or **Require
   approval above threshold**, and enter the **Write-off approval threshold** if asked.
4. Select **Save changes**.

### Require a reason on every adjustment
Keywords: adjustment reason, justification
1. Open [Invoice adjustment controls](/admin/invoice-adjustment-controls).
2. In **Documentation requirements**, set **Adjustment reason requirement** to **Required**.
3. Select **Save changes**.

### Control customer credits and superseded invoices
Keywords: unapplied credit, over-credit, show old invoice to customer
1. Open [Invoice adjustment controls](/admin/invoice-adjustment-controls).
2. In **Credit and visibility**, set the **Customer credit balance policy** and **Over-credit
   policy**.
3. Set the **Superseded invoice visibility policy** to decide whether customer-facing views show
   only the current invoice or also the ones it replaced.
4. Select **Save changes**.

## Notes
Opening the page needs read access to invoice adjustment control; saving needs update access.
Adjustments that need approval wait on [Pending approvals](/billing/pending-approvals).
