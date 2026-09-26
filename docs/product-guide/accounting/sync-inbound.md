---
path: /accounting/sync/inbound
aliases: [QuickBooks payments, payment entered in QuickBooks, customer paid in QuickBooks, bill paid in QuickBooks, receive payment QuickBooks, payments from QuickBooks, inbound payments, two-way sync, invoice paid in books, settlement paid in books]
related:
  - /accounting/sync
  - /admin/integrations
---

## What it's for
Payments from the books lists the payments someone recorded in QuickBooks Online against
invoices and settlements Trenova sent: customer payments that pay invoices, and bill payments that
pay carrier or owner-operator settlements. Trenova reads them every few minutes, and as soon as
QuickBooks tells it something changed. Each one says what it pays and whether it was brought into
Trenova, so the invoice or settlement shows as paid here too.

Payments Trenova sent itself are never read back. A payment waits for a person when it does not
match cleanly, such as one that pays more than is open, pays documents Trenova did not send, pays
a settlement only in part, or falls in a period that is not open; its reason says which. What
happens to the rest follows the choice for payments recorded in QuickBooks Online in the
integration's sync settings.

The strip at the top counts payments **Waiting for you**, **Just read**, and those **Applied** or
**Ignored** in the last seven days; selecting a figure filters the table to it.

## Tasks

### Apply a payment recorded in QuickBooks
Keywords: bring payment into Trenova, mark invoice paid, settlement paid in QuickBooks, receive payment
1. Open [Payments from the books](/accounting/sync/inbound) and select **Waiting for you** in the strip.
2. Open the payment. **Applying it posts** shows each invoice or settlement it pays and any cash
   left on the customer's account as unapplied.
3. Select **Apply in Trenova**. A customer payment is posted and applied to its invoices, with its
   credit memos applied first; a bill payment marks its settlement paid on the day it was paid in
   QuickBooks.

### Ignore a payment
Keywords: already entered, duplicate payment, do not bring in, refund
1. Open [Payments from the books](/accounting/sync/inbound) and open the payment.
2. Select **Ignore**, say why it stays out of Trenova, then select **Ignore payment**.

### Choose what happens to payments recorded in QuickBooks
Keywords: automatic payments, apply automatically, turn off payment sync, inbound payment policy
1. Open [Integrations](/admin/integrations) and open the QuickBooks Online card.
2. Under **Sync settings**, choose what happens to payments recorded in QuickBooks Online:
   **Wait for someone to apply them**, **Apply them automatically**, or **Leave them out of
   Trenova**.
3. Select **Save**.

## Notes
Viewing the page needs read access to accounting sync; applying and ignoring need update access,
and applying also needs permission to post customer payments or to mark the settlement paid.
A payment applied here cannot be voided or changed from Trenova while it pays more than one
document, because QuickBooks holds it as one payment; change it in QuickBooks instead. A payment
voided in QuickBooks before it was applied is marked **Voided in the books**.
