---
path: /accounting/ar/customer-ledger
aliases: [customer account, AR ledger, customer history, customer statement, account statement, credit memo, debit memo]
related:
  - /accounting/ar/aging
  - /accounting/ar/open-items
  - /accounting/ar/payments
covers:
  - /accounting/ar/customer-statement/:customerId
---

## What it's for
The customer ledger shows one customer's receivables history as a running statement: every
posting with its **Date**, **Document**, **Event**, **Source**, **Debit**, **Credit** and running
**Balance**. Above it sits the customer's AR profile: **Open balance**, **Credit utilization**,
**DSO / days to pay**, **Delinquency score**, a chart of **Payments — trailing 12 months** and
**Account details** such as **Oldest open invoice**, **Last payment** and **Unapplied cash**.

AR clerks and collectors use it to answer a customer's balance questions, raise credit or debit
memos, record payments and open a customer statement. The customer statement page
shows an **Opening balance**, **Charges**, **Payments** and **Ending balance** for a date range, an
**Aging summary**, the **Transaction history** and the customer's **Open items**.

## Tasks

### Look up a customer's ledger
Keywords: customer balance, account history, what does customer owe
1. Open [Customer ledger](/accounting/ar/customer-ledger).
2. Pick the **Customer**. Their profile and ledger load below.

### Export a customer's ledger
Keywords: download ledger, CSV
1. Open [Customer ledger](/accounting/ar/customer-ledger) and pick the **Customer**.
2. Select **Export**.

### Open a customer statement
Keywords: statement of account, customer statement, send statement
1. Open [Customer ledger](/accounting/ar/customer-ledger) and pick the **Customer**.
2. Select **Statement**. The customer statement opens.
3. Change **Statement date** or **Start date** to cover a different range.
4. Select **Back to open items** to go to [Open items](/accounting/ar/open-items).

### Raise a credit memo or debit memo for a customer
Keywords: credit note, debit note, adjust balance, write off, add charge
1. Open [Customer ledger](/accounting/ar/customer-ledger) and pick the **Customer**.
2. Select **Credit memo** or **Debit memo**.
3. Fill in **Reason**, and optionally **Memo date** and **Memo text**.
4. Under **Lines**, enter a **Description**, **Amount** and **Qty** for each line. Select
   **Add line** for more.
5. Turn on **Post immediately** to post the memo to the ledger in the same step.
6. Select **Create credit memo** or **Create debit memo**. The new memo opens in the invoice
   workspace.

### Record a payment from the customer
Keywords: receive payment, cash receipt
1. Open [Customer ledger](/accounting/ar/customer-ledger) and pick the **Customer**.
2. Select **Record payment**. The record payment panel opens on
   [Customer payments](/accounting/ar/payments) with this customer filled in.

## Notes
Opening the ledger or a customer statement needs read access to accounts receivable.
**Credit memo** and **Debit memo** only appear for people who can create invoices, and
**Record payment** only for people who can create customer payments.
