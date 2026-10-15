---
path: /billing/factoring
aliases: [factoring queue, ready to factor, factoring readiness, sell invoices, schedule of accounts, factored invoices, NOA, notice of assignment, missing POD for factoring, submit to factor, factoring schedule, failed schedule, factor remittance, funding report, reserve report, remittance import, remittance exceptions, apply remittance]
covers:
  - /billing/factoring/submissions/:id
  - /billing/factoring/remittances/:id
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

The strip at the top totals the open balance of invoices that are **Ready**, **Not ready**,
**Submitted** (on a schedule with the factor), and **Ready over a day** (ready for more than a day
without being submitted). The **Ready**, **Not ready** and **Submitted** tabs list those invoices
with their customer, factor and open balance. Open an invoice to see each check it was given, what
failed, and the action that clears it.

Ready invoices are sold by sending them to their factor on a schedule of accounts: one PDF listing
each invoice with the funding expected under the agreement, the same rows as a CSV, and one packet
per invoice with its supporting documents. The **Schedules** tab lists every schedule sent, and each
opens on its own page with its delivery, documents and invoices.

What the factor pays back arrives as a remittance: its funding, rejection, reserve release or
chargeback report as a CSV, an Excel workbook or a PDF. The **Remittances** tab lists every one,
whether it was imported by hand or read from an email the factor sent to your inbox. Each line is
matched to an invoice on one of the factor's schedules; a line that matched on its own waits to be
booked, and the rest wait for someone to resolve them. Nothing reaches the ledger until the
remittance is applied.

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

### Send ready invoices to the factor
Keywords: submit to factor, sell invoices, send schedule of accounts, factor invoices, schedule of accounts
1. Open [Factoring](/billing/factoring) and select the **Ready** tab.
2. Select the invoices to send, then select **Submit to factor**.
3. Check the schedule: the **Face value**, **Advance**, **Fee** and **Reserve** expected, where it
   is sent, and each invoice. A schedule goes to one factor; when the selection spans several, pick
   the **Factor** to send to now.
4. Select **Send**. The schedule opens on its own page while it is built and delivered, and its
   invoices move to the **Submitted** tab.

### Retry or cancel a schedule that failed to send
Keywords: factoring schedule failed, resend schedule, factor delivery failed, release invoices from schedule
1. Open [Factoring](/billing/factoring), select the **Schedules** tab and open the schedule marked
   **Failed to send**.
2. Read why delivery failed at the top of the page. Fix the factor's delivery settings first if
   they are wrong; see [Factoring companies](/billing/configuration-files/factoring-companies).
3. Select **Retry** to send it again with the factor's current settings, or **Cancel schedule**,
   enter the **Reason** and confirm to put its invoices back on the **Ready** tab.

### Import a factor's remittance
Keywords: upload remittance, funding report, reserve report, factor statement, import factor file
1. Open [Factoring](/billing/factoring), select the **Remittances** tab and select
   **Import remittance**.
2. Choose the **Factor**, what the file is a **Report of** when it has no column saying so, and
   optionally the date it was **Remitted on**.
3. Drop the file or select it. The remittance opens on its own page once it is read; a PDF is read
   in the background and its lines appear when it is done.
4. If the page asks you to **Map the columns**, choose the column that holds the invoice number or
   the factor's reference and the one that holds the amount, then select
   **Read with this mapping**. The factor's next file with the same columns is read on its own.

### Resolve remittance lines that did not match
Keywords: remittance exceptions, unmatched remittance lines, match remittance line, ignore remittance line
1. Open the remittance from the **Remittances** tab.
2. Under **To resolve**, select **Use** beside a suggested invoice, or **Pick invoice** to choose
   another open invoice on the factor's schedules.
3. Change what a line records when the factor's file got it wrong, or select **Ignore**, enter the
   **Reason** and confirm for a fee or adjustment you book by hand. **Restore** brings an ignored
   line back, and **Unmatch** under **Matched** undoes a match.
4. Select **Match again** to rematch every open line against the schedules as they are now.

### Book a remittance to the ledger
Keywords: apply remittance, book factor funding, post remittance, record reserve release from remittance
1. Open the remittance and select **Apply**.
2. Check what is booked: the invoices funded, rejected, released and charged back, the advance,
   fee and reserve, and each line's journal. Lines still to resolve stay open.
3. Select **Apply**. The remittance is marked **Applied**, or **Partly applied** while lines are
   left; resolve them and apply again to book the rest.

### Discard a remittance
Keywords: delete remittance, duplicate remittance, wrong remittance file
1. Open the remittance and select **Discard**.
2. Enter the **Reason** and confirm. A discarded remittance can no longer be booked; lines already
   booked stay in the ledger.

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
Sending invoices to a factor needs permission to submit factoring schedules; retrying a failed
schedule needs update access to them, and cancelling one needs permission to cancel them. Opening a
schedule needs read access to factoring schedules.

A schedule carries at most 200 invoices, all for one factor, and the factor needs schedule delivery
set up first. A failed delivery is retried on its own before the schedule is marked **Failed to send**,
and Watchtower raises it as an exception. An invoice on a schedule stays off the other tabs until the
schedule is cancelled.

Readiness is checked again whenever a document, the invoice, a dispute, the agreement or the notice
changes, so a fixed check clears on its own. Voided, paid and credit memo invoices, and invoices kept
out of factoring, are not listed.

Opening a remittance needs read access to factor remittances; importing one needs permission to
create them, resolving lines and matching again need update access, applying needs permission to
import them, and discarding needs permission to cancel them. A remittance holds at most 2,000 lines
and a file at most 32 MiB. An email the inbox classifies as a factor remittance is read into a
remittance for the factor whose address or domain sent it, and the message links to it.

Agreements and notices of assignment are kept on each factor; see
[Factoring companies](/billing/configuration-files/factoring-companies).
