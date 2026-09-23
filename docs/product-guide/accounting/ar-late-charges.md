---
path: /accounting/ar/late-charges
aliases: [late fees, finance charges, overdue charges, interest on overdue invoices, late payment fees]
related:
  - /accounting/ar/invoices
  - /accounting/ar/aging
  - /billing/configuration-files/customers
---

## What it's for
Late charges shows what the late-charge run would bill for overdue invoices, and lets you raise
those charges now for the customers you choose. Each overdue invoice is charged once per
thirty-day period at the customer's late charge rate, after their grace period, and each customer
gets one debit memo per run.

The preview table groups lines by customer, with **Invoice**, **Period**, **Basis**, **Rate** and
**Charge** for each overdue period and a total per customer. Customers the run would skip show the
reason instead of a checkbox. AR staff use the page to check charges before they go out and to run
them by hand.

## Tasks

### Preview late charges
Keywords: what late fees are due, check late charges, overdue interest
1. Open [Late charges](/accounting/ar/late-charges).
2. Pick a **Customer** to see only that customer, or leave it on all customers.
3. Set **As of** to preview charges as of another date; it starts on today.
4. Select an invoice number to open the invoice.

### Raise late charges now
Keywords: assess late fees, run late charges, create debit memo, bill late fees
1. Open [Late charges](/accounting/ar/late-charges).
2. Tick the customers to charge, or tick the header box to select them all.
3. Select **Assess now**. The confirmation shows how many debit memos will be raised and their
   total.
4. Select **Assess late charges**. The table then shows each new debit memo number with
   **Posted** or **Draft**; select a memo number to open it.

### Turn automatic late charges on or off
Keywords: late charge mode, nightly late charges, enable late fees
1. Open **Billing controls** in the organization settings.
2. In the **Late charges** card, set **Assessment mode** to **Disabled**, **Preview only** or
   **Automatic**, and optionally a **Minimum late charge**.
3. Save the billing controls.

### Set a customer's late charge rate
Keywords: late fee percentage, grace period
1. Open [Customers](/billing/configuration-files/customers) and open the customer.
2. In their billing profile, turn on **Apply late charges** and fill in **Late charge rate** and
   **Grace period**.

## Notes
- Opening the page needs read access to accounts receivable. **Assess now** only appears for people
  who can create invoices.
- A notice above the table says how the nightly run is set: **Automatic** raises and posts the
  memos each night, **Preview only** writes nothing, and **Disabled** means **Assess now** stays
  unavailable even though you can still preview.
- Each invoice period is charged only once, so running again cannot double-charge a customer.
- Customers whose charges for a run add up to less than the **Minimum late charge** are skipped.
