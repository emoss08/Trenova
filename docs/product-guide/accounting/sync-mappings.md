---
path: /accounting/sync/mappings
aliases: [QuickBooks mappings, chart of accounts mapping, account mapping, map accounts, QuickBooks items, map customers to QuickBooks, map carriers to vendors, accounting sync setup]
related:
  - /admin/integrations
---

## What it's for
Accounting mappings say which QuickBooks Online account, item, customer and vendor each Trenova
record is sent as: the account roles (accounts receivable, revenue, deposit account, write-off,
accounts payable and purchased transportation), invoice line types, accessorial charges, the
short-pay write-off item, customers, carriers, payment terms and payment methods.

Trenova reads the QuickBooks company's records and proposes a match where it is sure enough.
A proposal is only a suggestion: nothing is used when syncing until a person confirms it. The
strip at the top counts **Required confirmed**, **Proposed**, **Unmatched** and **Confirmed**
mappings; selecting a figure filters the list to it.

## Tasks

### Confirm Trenova's proposals
Keywords: accept suggested matches, approve mappings, bulk confirm
1. Open [Mappings](/accounting/sync/mappings).
2. Select **Proposed** in the strip to list the proposals. The ones Trenova is sure of are
   already ticked; tick or untick the rest.
3. Select the confirm button above the list; it names how many are ticked.

### Choose a different record
Keywords: change mapping, wrong account, remap customer, pick QuickBooks record
1. Open [Mappings](/accounting/sync/mappings) and select the mapping on the left. Use **Search
   mappings** or **Every kind** to narrow the list.
2. Under **Other records Trenova considered**, select **Use** beside the right record, or search
   the QuickBooks Online records below it and select **Use**.
3. The mapping is confirmed with the record you chose.

### Turn down or clear a mapping
Keywords: reject proposal, unmap, remove mapping
1. Open [Mappings](/accounting/sync/mappings) and select the mapping.
2. Select **Turn down** to reject a proposal; Trenova will not propose that record for it again.
3. Select **Clear** to remove the chosen record so the mapping is unmatched again.

### Create a missing item, customer or vendor in QuickBooks
Keywords: add item to QuickBooks, create vendor, create customer in QuickBooks
1. Open [Mappings](/accounting/sync/mappings) and select an accessorial charge, line type,
   customer or carrier mapping that has no match.
2. At the bottom of the mapping, check the name Trenova will create it under, then select the
   create button.
3. Trenova creates the record and confirms the mapping. Items are created with the confirmed
   revenue account as their income account, so confirm that account first.

### Read QuickBooks again
Keywords: refresh chart of accounts, pull new accounts, reload QuickBooks records
1. Open [Mappings](/accounting/sync/mappings).
2. Select the button that reads QuickBooks Online again, beside when it was last read. The
   proposals update when the read finishes.

### Finish setup
Keywords: complete accounting setup, done mapping
1. Confirm every required mapping: **Required confirmed** shows how many are left.
2. Select **Finish setup**.

## Notes
Viewing mappings needs read access to the accounting integration; confirming, choosing, clearing,
creating records and reading QuickBooks again need update access. Accounts, terms and payment
methods are the bookkeeper's and are never created from Trenova. QuickBooks is read again every
day on its own. A mapping Trenova proposed from a model's suggestion is never ticked by default.
