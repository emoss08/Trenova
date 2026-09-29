---
path: /accounting/sync/drift
aliases: [QuickBooks drift, invoice changed in QuickBooks, document deleted in QuickBooks, books differ, reconcile QuickBooks, reconciliation differences, QuickBooks does not match, fix QuickBooks mismatch, customer balance differs, sync mismatch, account balance differs, trial balance does not match QuickBooks, Xero drift, invoice changed in Xero, invoice voided in Xero, Xero does not match]
related:
  - /accounting/sync
  - /accounting/sync/inbound
  - /admin/integrations
---

## What it's for
Drift findings lists the documents Trenova sent to the accounting system (QuickBooks Online or
Xero) that differ there now: a total someone edited, a document deleted or voided in the books, a
document voided in Trenova but still live in the books, or a customer's open balance that no
longer adds up to the same figure on both sides. Each finding shows both values, when the document
was changed in the books and, for QuickBooks Online, who changed it, and whether an amount
difference is within the reconciliation tolerance.

Trenova compares every synced document dated in an open or locked period each night, and compares
a document again within minutes when the accounting system reports it changed. A finding closes on its own
once both sides agree again. A document with a change still on its way to the books is left out
until that change is sent.

When a QuickBooks Online connection sends journal entries, the check also compares each QuickBooks account's
balance with the Trenova accounts mapped to it. A **Different account balance** finding lists
those Trenova accounts and their balances. It cannot be pushed or adjusted as a whole: find the
entry made in QuickBooks by hand, or the one missing from it, and correct that entry. Accounts are
not compared while journal entries are still waiting to be sent, or when Trenova and QuickBooks
keep different home currencies.

The strip at the top counts **Open differences**, **Different amounts**, documents **Deleted or
voided** in the books, and those **Settled** in the last seven days; selecting a figure filters the
table to it.

## Tasks

### Send Trenova's value to the accounting system
Keywords: push to QuickBooks, overwrite QuickBooks, push to Xero, restore deleted invoice, send again
1. Open [Drift findings](/accounting/sync/drift) and open the finding.
2. Select **Push Trenova's value**. The preview says what is sent: an update of the total, a void,
   or a new copy of a document deleted or voided in the books.
3. Select **Push Trenova's value** again to confirm. The finding closes once the next check finds
   the books match Trenova.

### Match Trenova to the accounting system
Keywords: adjust Trenova, credit memo to match, void to match, reverse payment deleted in QuickBooks, reverse payment deleted in Xero
1. Open [Drift findings](/accounting/sync/drift) and open the finding.
2. Select **Adjust Trenova**. The preview says what is posted: a credit memo or a debit memo
   against the invoice for the difference, a void of an invoice gone from the books, or the
   reversal of a payment voided there.
3. Select **Adjust Trenova** again to confirm. What is posted is not sent back to the
   accounting system, because the books already have it.

### Dismiss a difference
Keywords: ignore difference, rounding, accept difference, keep both
1. Open [Drift findings](/accounting/sync/drift) and open the finding.
2. Select **Dismiss**, say why both sides stay as they are, then select **Dismiss difference**.

### Check the accounting system now
Keywords: reconcile now, compare now, run drift check
1. Open [Drift findings](/accounting/sync/drift).
2. Select **Check now**. The check runs in the background; the page shows when it last finished.

## Notes
Viewing the page needs read access to accounting sync; fixing and dismissing need update access.
Adjusting Trenova also needs permission to change invoices, or customer payments for a reversal.
Adjust Trenova is offered for invoices and debit memos, and for customer payments voided or
deleted in the books; settlements, credit memos and customer balances offer only Push Trenova's
value or Dismiss, and account balances offer only Dismiss. An invoice with payments or credits applied cannot be voided to match until they
are unapplied. Agents may dismiss only an amount difference within the reconciliation tolerance.
