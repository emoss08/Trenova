---
path: /billing/factoring
aliases: [factoring queue, ready to factor, factoring readiness, sell invoices, schedule of accounts, factored invoices, NOA, notice of assignment, missing POD for factoring]
related:
  - /billing/configuration-files/factoring-companies
  - /billing/configuration-files/customers
  - /billing/invoices
---

## What it's for
Factoring lists every posted invoice assigned to a factor and says whether it is ready to sell.
An invoice is ready when it is posted and open with no open dispute, its customer has acknowledged a
notice of assignment for the factor, the factor has an active agreement, the shipment passes billing
readiness, and the invoice carries every document the agreement asks for.

The strip at the top totals the open balance of invoices that are **Ready**, **Not ready**, and
**Ready over a day** (ready for more than a day without being submitted). The **Ready** and **Not ready**
tabs list those invoices with their customer, factor and open balance. Open an invoice to see each
check it was given, what failed, and the action that clears it.

## Tasks

### See which invoices are ready to factor
Keywords: ready to sell, factoring queue, invoices ready for the factor
1. Open [Factoring](/billing/factoring).
2. Select the **Ready** tab, or the **Ready** figure in the strip. **Ready since** shows when each
   invoice became ready.

### Find out why an invoice is not ready
Keywords: factoring blocked, failed checks, not ready to factor, factoring readiness checks
1. Open [Factoring](/billing/factoring) and select the **Not ready** tab.
2. Select the invoice's row. Its **Checks** list the failed checks first, each with what clears it.
3. Follow the check's link: **Open the factor's agreements** or **Open the factor's notices** for a
   missing agreement or notice, **Open the disputes** for an open dispute, and **Open the shipment**
   for a shipment that fails billing readiness.

### Ask for a missing document
Keywords: request POD, missing proof of delivery, missing BOL for factoring, request paperwork
1. Open [Factoring](/billing/factoring), select the **Not ready** tab and select the invoice's row.
2. On the check for the missing document, select **Request documents**.
3. Optionally enter a **Recipient** and a **Message**, then select **Send request**. With no
   recipient, the driver is asked first, then the carrier, then the customer's invoice contacts.

### Keep one invoice out of factoring
Keywords: exclude invoice from factoring, collect it ourselves, factoring override, do not factor
1. Open [Invoices](/billing/invoices) and select the invoice.
2. In the **Factoring** section on the **Overview** tab, change **Factoring** to
   **Keep out of factoring**.
3. Enter the **Reason**, then select **Save**. Change it back to **Follow the customer** to return
   the invoice to its customer's factor.

## Notes
Opening the page needs read access to invoices. Requesting documents needs permission to create
customer communications, and keeping an invoice out of factoring needs update access to invoices.

Readiness is checked again whenever a document, the invoice, a dispute, the agreement or the notice
changes, so a fixed check clears on its own. Voided, paid and credit memo invoices, and invoices kept
out of factoring, are not listed.

Agreements and notices of assignment are kept on each factor; see
[Factoring companies](/billing/configuration-files/factoring-companies).
