---
path: /accounting/ar/invoices
aliases: [invoice list, invoice ledger, all invoices, issued invoices, invoice history, billed invoices]
related:
  - /billing/invoices
  - /accounting/ar/open-items
  - /accounting/ar/payments
  - /accounting/ar/aging
---

## What it's for
The invoice register lists every invoice your organization has issued, as a ledger rather than a
work queue. Each row shows **Invoice #**, **Type**, **Status**, **Bill To**, **Shipper**,
**Invoice date**, **Due date**, **Total**, **Open**, **Past due**, **Settlement**, **Dispute**,
**EDI**, **Scope** and **Split**. The **Shipper** column is only filled in when the invoice bills
someone other than the shipper.

Accounting and AR staff use it to look up any invoice, current or historical, check whether it
has been paid, and jump to the invoice or the shipment behind it.

## Tasks

### Find an invoice
Keywords: search invoices, invoice number, look up invoice, unpaid invoices, disputed invoices
1. Open [Invoice register](/accounting/ar/invoices).
2. Type in the search box, or use **Filter** to add conditions on columns such as **Invoice #**,
   **Bill To**, **Status**, **Settlement**, **Dispute** or **Invoice date**.
3. Select a column heading to sort by it, for example **Due date** or **Open**.

### Open an invoice
Keywords: view invoice, invoice detail
1. Open [Invoice register](/accounting/ar/invoices).
2. Select the row or its **Invoice #**. The invoice opens in the
   [Invoices](/billing/invoices) workspace.
3. Or right-click the row and choose **Open in workspace**.

### See the shipment an invoice billed
Keywords: invoice shipment, load for invoice
1. Open [Invoice register](/accounting/ar/invoices).
2. Right-click the row and choose **View shipment**. The shipment opens in a new browser tab.
   The option only appears when the invoice is tied to a shipment.

### Void an invoice
Keywords: cancel invoice, reverse invoice, void
1. Open [Invoice register](/accounting/ar/invoices).
2. Right-click the row and choose **Void**.
3. Enter the **Reason** and choose a **Freight disposition**: **Release freight for rebilling**
   sends the billing queue items back for a fresh invoice; **Do not rebill** cancels them.
4. Select **Void invoice**.

## Notes
- Opening the page needs read access to invoices. The register has no create button; invoices
  are created from the billing queue.
- **Void** is unavailable once any payment has been applied to the invoice, and is hidden on
  invoices that are already voided.
- A draft is voided at once. A posted invoice is voided through a full-reversal credit memo; when
  that reversal needs an approver, the dialog shows **Void requested; awaiting reversal approval**
  and the invoice reads voided only once the reversal is approved.
