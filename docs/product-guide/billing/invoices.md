---
path: /billing/invoices
aliases: [bills, draft invoices, post invoice, send invoice, credit memo, void invoice, invoice dispute]
related:
  - /billing/queue
  - /billing/pending-approvals
  - /billing/reconciliation-exceptions
  - /billing/adjustment-batches
---

## What it's for
Invoices is the billing team's workspace for the invoices created from the billing queue. The list on the left can be searched and filtered; selecting an invoice opens its detail with the bill-to, dates, payment terms and totals, and the tabs **Overview**, **Delivery**, **Charges**, **Payments**, **Disputes**, **Documents** and **Activity**.

Billers use it to check draft invoices and post them, generate and email the invoice PDF, and handle what comes after posting: adjustments and credits, voids, customer disputes and payment applications.

## Tasks

### Find an invoice
Keywords: search invoice, invoice number, filter invoices, disputed invoices
1. Open [Invoices](/billing/invoices).
2. Type an invoice number, PRO number or bill-to name in the search box above the list.
3. Narrow the list with **Invoice status**, **Bill type** and **Invoice scope**, or turn on **Disputed only**.
4. Select an invoice to open it.

### Review and post a draft invoice
Keywords: finalize invoice, post to ledger, approve invoice
1. Open [Invoices](/billing/invoices) and select a draft invoice.
2. Check the **Overview** and **Charges** tabs, and the supporting paperwork on **Documents**.
3. Select **Post invoice** in the invoice header. You can also right-click a draft in the list and choose **Post invoice**.

### Email the invoice to the customer
Keywords: send invoice, invoice PDF, reprint, resend invoice
1. Open [Invoices](/billing/invoices), select the invoice and open the **Delivery** tab.
2. Under **Email delivery**, select Generate PDF (or **Regenerate** to rebuild an existing one). **Reprint** downloads the current PDF.
3. Check the recipients, subject and attachments in the send plan, then select **Send** (or **Resend** if it has already gone out).
4. If sending is disabled, hover the button to see why, for example a missing PDF or a problem with the customer's email setup.

### Credit, rebill or reverse an invoice
Keywords: invoice adjustment, credit memo, rebill, correction, reversal
1. Open [Invoices](/billing/invoices) and select the invoice.
2. Select **Adjust invoice**.
3. Choose **Credit only**, **Credit & rebill** or **Full reversal**; for a rebill, also choose a **Rebill strategy**.
4. Enter a **Reason**, set the credit and rebill amounts per line, and attach any **Supporting documents (optional)**.
5. Select **Preview** to see the totals, warnings and policy implications, then select **Execute**, or **Submit for approval** when the policy requires one.

### Void an invoice
Keywords: cancel invoice, void, retire freight
1. Open [Invoices](/billing/invoices) and select the invoice.
2. Open the **Invoice actions** menu (the three-dot button in the header) and choose **Void invoice**.
3. Enter a **Reason** and pick a **Freight disposition**: **Release freight for rebilling** sends the billing queue items back to Approved, **Do not rebill** cancels them.
4. Select **Void invoice** to confirm.

### Record a customer dispute
Keywords: contested invoice, short pay dispute, rate discrepancy
1. Open [Invoices](/billing/invoices), select the invoice and open the **Disputes** tab.
2. Select **Open dispute**, choose a **Reason**, enter the **Disputed amount** and **Notes**, then select **Open dispute**.
3. When it is settled, select **Resolve** on the dispute and record the **Resolution**, or **Withdraw** to drop it without an outcome.

## Notes
Opening the page needs read access to invoices. Voiding needs permission to cancel invoices, and sending by EDI needs permission to update them.

Posting finalizes the invoice, records it in accounting and marks its billing queue item as posted. Only a posted invoice can be sent by EDI.

An invoice with customer payments or credit memos applied cannot be voided until they are unapplied on the **Payments** tab. Depending on the adjustment policy, a void or adjustment may wait for approval on [Pending approvals](/billing/pending-approvals) before it takes effect.

While a dispute is open, late charges on the invoice pause. To record a payment against an open invoice, use **Record payment** on the **Payments** tab, which opens [Customer payments](/accounting/ar/payments) with the invoice selected.
