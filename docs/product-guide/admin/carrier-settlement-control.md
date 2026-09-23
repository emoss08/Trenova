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
The page is one settings form in four cards: **Pay period** (how often carriers are settled and when
carrier cost accrues), **Workflow automation** (automatic batches and posting), **Invoice matching**
(how carrier invoices are compared to the buy rate) and **Posting accounts** (the GL accounts
settlements post to).

Carrier payables and brokerage finance staff use it.

## Tasks

### Set the carrier pay period
Keywords: carrier pay cycle, pay date, cost accrual trigger
1. Open [Carrier settlement control](/admin/carrier-settlement-control).
2. In **Pay period**, choose the **Frequency** and **Period end day**, and enter the **Pay Delay
   (days)**.
3. Choose the **Pay trigger**, the shipment milestone at which carrier cost accrues.
4. Select **Save changes**.

### Automate batches and posting
Keywords: auto batch, auto post
1. Open [Carrier settlement control](/admin/carrier-settlement-control).
2. In **Workflow automation**, turn on **Auto-generate batches** to create a batch when each pay
   period closes, and **Auto-post on approval** to post a settlement to the ledger as soon as it is
   approved.
3. Select **Save changes**.

### Match carrier invoices automatically
Keywords: invoice matching, variance tolerance, EDI 210, auto accept
1. Open [Carrier settlement control](/admin/carrier-settlement-control).
2. In **Invoice matching**, enter the **Variance tolerance** in USD.
3. Turn on **Auto-match inbound invoices** to match inbound EDI 210 invoices to their carrier
   assignment.
4. Turn on **Auto-accept within tolerance** to resolve matched invoices within the tolerance without
   review. It requires auto-match.
5. Select **Save changes**.

### Set the posting accounts
Keywords: AP account, purchased transportation account
1. Open [Carrier settlement control](/admin/carrier-settlement-control).
2. In **Posting accounts**, choose the **Accounts payable account** and the **Purchased
   transportation account**.
3. Select **Save changes**.

## Notes
The page is only available to organizations with brokerage features turned on. Opening it needs
read access to carrier settlement control; saving needs update access. A blank posting account falls
back to the defaults on [Accounting controls](/admin/accounting-control). Invoices outside the
tolerance still land in the review workspace.
