---
path: /admin/billing-controls
aliases: [billing settings, invoice settings, auto invoicing, billing automation, billing rules, late charge settings]
related:
  - /billing/queue
  - /billing/invoices
  - /accounting/ar/late-charges
  - /billing/configuration-files/formula-templates
  - /admin/invoice-adjustment-controls
---

## What it's for
Billing control holds the organization-wide rules for billing. The page is one settings form in
five cards: **Invoice defaults** (what invoices show and the fallback payment term, terms and
footer), **Automation policy** (how shipments move into billing and whether invoices are drafted and
posted automatically), **Exception policy** (how missing billing requirements and rate variances are
handled), **Rating policy** (what happens to a shipment no rate agreement covers) and **Late
charges** (whether the nightly late-charge run raises debit memos).

Billing managers and finance leads use it. A warning at the top of the page notes that these
settings affect revenue processing and invoicing, and should change only after review.

## Tasks

### Set invoice defaults
Keywords: payment terms, invoice footer, due date on invoice, net 30
1. Open [Billing controls](/admin/billing-controls).
2. In **Invoice defaults**, turn **Show due date on invoice** and **Show balance due on invoice**
   on or off.
3. Choose the **Default payment term** and fill in **Default invoice terms** and **Default invoice
   footer**. These apply when a customer's billing profile does not set its own.
4. Select **Save changes**.

### Automate the billing queue and invoice drafts
Keywords: auto transfer, auto invoice, ready to bill, auto post invoices
1. Open [Billing controls](/admin/billing-controls).
2. In **Automation policy**, set **Ready-to-bill assignment mode** and **Billing queue transfer
   mode**. With **Automatic when ready**, also set the **Billing queue transfer schedule** and
   **Billing queue transfer batch size**.
3. Set **Invoice draft creation mode**. With **Automatic when transferred**, also set the **Auto
   invoice batch size** and whether to **Notify on auto invoice creation**.
4. Set **Invoice posting mode** to **Manual review required** or **Automatic when no blocking
   exceptions**.
5. Select **Save changes**.

### Decide how billing exceptions are handled
Keywords: rate variance, billing requirements, block billing, review routing
1. Open [Billing controls](/admin/billing-controls).
2. In **Exception policy**, set **Shipment billing requirement enforcement** and **Rate validation
   enforcement** to **Ignore**, **Warn**, **Require review** or **Block**.
3. If either uses **Require review**, choose the **Billing exception disposition**: **Route to
   billing review** or **Return to operations**.
4. Set the **Rate variance tolerance percent** and the **Rate variance auto resolution mode**, and
   turn on **Notify on billing exceptions** if wanted.
5. Select **Save changes**.

### Handle shipments with no rate agreement
Keywords: unrated shipments, fallback formula, margin floor, rate override reason
1. Open [Billing controls](/admin/billing-controls).
2. In **Rating policy**, choose the **Unrated shipment disposition**: **Fall back to formula
   template**, **Zero the rate and flag for review** or **Block the save**.
3. For the fallback, optionally pick a **Fallback formula template**.
4. Turn on **Require rate override reason** and **Enforce margin floor** as needed.
5. Select **Save changes**.

### Turn on automatic late charges
Keywords: late fees, overdue invoices, debit memo, finance charges
1. Open [Billing controls](/admin/billing-controls).
2. In **Late charges**, set **Assessment mode** to **Disabled**, **Preview only** or **Automatic**.
3. Set the **Minimum late charge**; customers whose charges for a run add up to less are skipped.
4. Select **Save changes**.

## Notes
Opening the page needs read access to billing control; saving needs update access. Late charge
rates and grace periods are set on each customer's billing profile, not here. In **Preview only**
the nightly run writes nothing, and memos are raised by hand on
[Late charges](/accounting/ar/late-charges).
