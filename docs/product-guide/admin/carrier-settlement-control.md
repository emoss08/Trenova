---
path: /admin/carrier-settlement-control
aliases: [carrier pay settings, carrier payables settings, AP settings, invoice match tolerance, brokerage settlement rules]
related:
  - /carrier-settlements/workspace
  - /carrier-settlements/batches
  - /carrier-settlements/invoice-matching
  - /admin/accounting-control
---

## What it's for
Carrier settlement control holds the organization's rules for paying carriers on brokered freight.
The page is one settings form in three tabs, saved together: **Pay period** (when each period ends,
when carriers are paid and when a carrier's cost counts), **Invoice matching** (how carrier invoices
are checked against the buy rate, and whether pay waits for a match) and **Approval & posting** (who
approves a settlement, whether approval posts it, and the ledger accounts it posts to).

Carrier payables and brokerage finance staff use it.

## Tasks

### Set the carrier pay period
Keywords: carrier pay cycle, pay date, cost accrual trigger
1. Open [Carrier settlement control](/admin/carrier-settlement-control). It opens on **Pay period**.
2. Under **Schedule**, choose the **Pay period** and the **Last day of the period**, and enter the
   **Pay delay** in days (0 to 30).
3. Turn on **Generate settlements when a period ends** to have a batch drafted for each period.
4. Under **Carrier cost**, choose when **Cost counts when**: when the carrier's own leg is completed, when
   the shipment is delivered, when it is ready to invoice or when the customer has been invoiced.
5. Select **Save changes**.

### Match carrier invoices automatically
Keywords: invoice matching, variance tolerance, EDI 210, auto accept
1. Open [Carrier settlement control](/admin/carrier-settlement-control) and select the **Invoice
   matching** tab.
2. Enter the **Variance tolerance**: an invoice within this amount of the buy rate matches, and a
   larger gap waits for review.
3. Turn on **Match EDI invoices as they arrive** to match invoices carriers send by EDI to their load.
4. Under it, turn on **Accept matches within the tolerance** to accept those matches without review.
5. Select **Save changes**.

### Hold carrier pay until the invoice is matched
Keywords: pay on invoice, hold settlement, three-way match, unmatched invoice, freight audit hold
1. Open [Carrier settlement control](/admin/carrier-settlement-control) and select the **Invoice
   matching** tab.
2. Under **Paying on a matched invoice**, turn on **Hold pay until the invoice is matched**.
3. Select **Save changes**.

### Set approval and the posting accounts
Keywords: separate approver, auto post, AP account, purchased transportation account
1. Open [Carrier settlement control](/admin/carrier-settlement-control) and select the **Approval &
   posting** tab.
2. Under **Approval**, choose whether to **Require a separate approver** and whether to **Post to the
   ledger on approval**.
3. Under **Ledger accounts**, choose the **Accounts payable account** and the **Purchased
   transportation account**.
4. Select **Save changes**.

## Notes
The page is only available to organizations with brokerage features turned on. Opening it needs
read access to carrier settlement control; saving needs update access. A save the server refuses
opens the tab holding the setting to fix and marks it. **Match EDI invoices as they arrive** appears
only when the plan includes EDI. A blank ledger account falls back to the defaults on
[Accounting controls](/admin/accounting-control). Invoices outside the tolerance still land in the
review workspace. With **Hold pay until the invoice is matched** on, a load's carrier cost stays out
of settlements until its carrier invoice match is resolved and is picked up by the next settlement
after that; a settlement that already holds cost for an unmatched load cannot be approved until the
match is resolved or the settlement is recalculated.
