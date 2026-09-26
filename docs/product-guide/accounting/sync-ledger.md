---
path: /accounting/sync
aliases: [QuickBooks sync, sync log, sync errors, sync history, failed to sync, QuickBooks not updating, invoice not in QuickBooks, accounting sync ledger, outbox, settlement not in QuickBooks, carrier bill, bill payment, vendor credit]
related:
  - /accounting/sync/mappings
  - /accounting/sync/inbound
  - /accounting/sync/drift
  - /admin/integrations
---

## What it's for
The sync ledger lists every document Trenova sends to QuickBooks Online and what happened to
each one: invoices, credit and debit memos, customer payments, credit applications and customer
changes, and on the payables side carrier settlements, owner-operator settlements, their payments,
and changes to carriers and drivers. Documents are sent on their own as they are posted; the ledger is where you see what
went through, what is waiting, and fix what did not.

The strip at the top counts **Synced**, **In progress**, **Waiting for release** and **Needs
attention** records; selecting a figure filters the table to it. Records held or failed for the
same reason are grouped under **Needs attention**, so one fix clears them all.

A posted carrier settlement is sent as a bill to the carrier's vendor, or as a vendor credit when
it nets below zero; its lines follow the settlement's journal entry, one per GL account. Marking it
paid sends a bill payment from the cash account. Voiding a posted settlement removes its bill.
Owner-operator settlements work the same way once **Send owner-operator settlements** is on for
the connection, each driver becoming a 1099 vendor; company driver pay is never sent.

## Tasks

### Fix a document that did not reach QuickBooks
Keywords: sync failed, held document, blocked invoice, missing mapping, closed period
1. Open [Sync ledger](/accounting/sync) and select **Needs attention** in the strip.
2. Open the record. Its reason says what to do, such as mapping a charge code to a QuickBooks item
   on the [Mappings](/accounting/sync/mappings) page.
3. Fix what the reason names, then select **Retry now**. To retry every record held for the same
   reason, select **Retry these** beside the reason under **Needs attention**.

### Fix a settlement that did not reach QuickBooks
Keywords: carrier bill held, settlement not synced, map GL account, vendor missing
1. Open [Sync ledger](/accounting/sync) and open the settlement's record.
2. When its reason names a GL account, map that account to a QuickBooks account on the
   [Mappings](/accounting/sync/mappings) page (filter by **GL accounts**). When it names a carrier
   or an owner-operator, map them to the existing QuickBooks vendor, or let Trenova create one.
3. Select **Retry now**. A settlement's payment waits for its bill and follows once the bill is
   in QuickBooks.

### Send a document dated in closed books
Keywords: closed period, books closed, re-date, first open day, closing date
1. Open [Sync ledger](/accounting/sync) and open the record held for a closed period.
2. Select **Send on first open day**. The document goes out dated on the day after the closing
   date in QuickBooks, and its note in QuickBooks keeps Trenova's date. This is offered only while
   Trenova's closed-period policy is to post to the next open period; otherwise reopen the period
   in QuickBooks, then select **Retry now**.

### Release documents waiting for approval
Keywords: approve sync, send held documents, manual sync
1. Open [Sync ledger](/accounting/sync) and select **Waiting for release** in the strip.
2. Open a record and select **Release** to send it, or use the release button in the page header
   to send all of them.

### Skip a document
Keywords: do not send, already entered by hand, ignore sync error
1. Open [Sync ledger](/accounting/sync) and open the record.
2. Select **Skip**, say why it is not sent, then select **Skip document**.

### Pause or resume sending
Keywords: stop syncing, month-end close, hold QuickBooks sync
1. Open [Sync ledger](/accounting/sync) and select **Pause sending**. Add a reason if you like,
   then select **Pause sending** again to confirm.
2. Posted documents keep queueing while sending is paused. Select **Resume sending** to send them
   in order.

### Send documents posted before sync was turned on
Keywords: backfill, historical invoices, send old documents
1. Open [Sync ledger](/accounting/sync) and select **Backfill**.
2. Choose the range and, optionally, the kinds of documents, then select **Start backfill**.
   Documents already queued are left alone, and one backfill runs at a time.

## Notes
Viewing the ledger needs read access to accounting sync; retrying, releasing, skipping, pausing
and resuming need update access; a backfill needs manage access to the accounting integration.
A document that fails for a temporary reason is retried on its own, backing off from 30 seconds
to 6 hours, and is marked failed after 8 tries. Invoices, customer payments, customers, carrier and
driver settlements, carriers and workers show where they stand in QuickBooks Online on their own
pages, with a link back to this ledger.
