---
path: /carrier-settlements/invoice-matching
aliases: [carrier invoice reconciliation, freight bill audit, match carrier bills, EDI 210, carrier invoice variance, three-way match, freight audit]
related:
  - /carrier-settlements/workspace
  - /carrier-settlements/cost-events
  - /dispatch/carriers
  - /admin/carrier-settlement-control
---

## What it's for
Carrier invoice matching is where accounts payable staff check each inbound carrier freight invoice (from EDI 210 or a scanned document) against the buy rate negotiated on the carrier assignment. An invoice is linked to a carrier, paired with the carrier assignment by pro number or shipment reference to form a match, and the match shows the invoice total beside the expected total (line haul, fuel surcharge and accessorials) and the variance between them.

The strip at the top counts **Invoices needing attention**, **Variance matches**, **Suggested matches** and **Resolved**, and a line above it shows whether auto-match and auto-accept within tolerance are on.

## Tasks

### Link an invoice to a carrier and create a match
Keywords: match carrier invoice, pair invoice with load, reconcile freight bill
1. Open [Invoice matching](/carrier-settlements/invoice-matching) and stay on the carrier invoices list.
2. Use **Needs attention** or **All**, or search with **Search invoice, pro number, carrier...**, and select the invoice.
3. Under **Carrier link**, select **Suggest carrier** to look the carrier up by SCAC and DOT number, or **Link carrier**, choose the **Carrier**, and select **Link carrier**.
4. Select **Create match**. The match pairs the invoice with the carrier assignment found by pro number or shipment reference.

### Resolve a match
Keywords: accept carrier invoice, approve freight bill, variance, pay billed amount
1. Open [Invoice matching](/carrier-settlements/invoice-matching) and switch to the matches list.
2. Filter with **Open** or a status (**Suggested**, **Matched**, **Variance**, **Resolved**, **Rejected**), and by how it was created (**Auto** or **Manual**).
3. Select the match and compare **Carrier invoice** with **Negotiated buy rate** and the variance.
4. Select **Accept** to reconcile the invoice without changing the accrued cost. For a match in **Variance**, select the accept-with-variance button, which accrues an adjustment cost event so the carrier is paid the billed amount.

### Reject a match
Keywords: dispute carrier invoice, wrong load, deny freight bill
1. Open the match on [Invoice matching](/carrier-settlements/invoice-matching).
2. Select **Reject**, enter the reason (required), and select **Reject**.

### Change the matching automation
Keywords: auto-match, auto-accept, variance tolerance
1. Open [Invoice matching](/carrier-settlements/invoice-matching) and select **Automation settings**.
2. Adjust the settings on [Carrier settlement control](/admin/carrier-settlement-control).

## Notes
Needs read access to carrier invoice matches, and the page is only available when the organization has the Brokerage capability.

A match marked **Auto-matched** was created by the inbound EDI 210 auto-match sweep; **Auto-accepted** means it resolved on its own because the variance was within tolerance.
