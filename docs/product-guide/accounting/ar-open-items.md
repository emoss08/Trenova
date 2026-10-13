---
path: /accounting/ar/open-items
aliases: [open invoices, unpaid invoices, outstanding invoices, open AR, receivables list]
related:
  - /accounting/ar/payments
  - /accounting/ar/aging
  - /accounting/ar/customer-ledger
---

## What it's for
Open items lists every invoice that still has money owing, across all customers: **Invoice**,
**Customer**, **Type**, **PRO / BOL**, **Invoice date**, **Due date**, **Status**, **Total**,
**Applied** and **Open**. Invoices under dispute show **Disputed** and short-paid invoices show
**Short-paid**. Figures above the table give **Total open**, **Current**, **Overdue**,
**Average age** and **Count**.

AR clerks use it to find what a customer owes, and to pick the invoices a payment should pay off.

## Tasks

### Find open invoices
Keywords: unpaid invoices, what is outstanding, search open invoices
1. Open [Open items](/accounting/ar/open-items).
2. Pick a **Customer** to see only their invoices, or leave it on all customers.
3. Set **As of date** to see what was open on an earlier date.
4. Select a column heading to sort by it.

### Apply a payment to selected invoices
Keywords: pay invoices, cash application, receive payment against invoices
1. Open [Open items](/accounting/ar/open-items).
2. Tick the invoices the payment covers. They must all belong to one customer.
3. Select **Apply payment**. The record payment panel opens on
   [Customer payments](/accounting/ar/payments) with the customer and invoices filled in.
4. Select **Clear** to drop the selection instead.

### Record a payment
Keywords: receive payment, cash receipt
1. Open [Open items](/accounting/ar/open-items).
2. Select **Record payment** in the page header.

### Open a customer's statement
Keywords: statement of account
1. Open [Open items](/accounting/ar/open-items).
2. Select the customer's name in the **Customer** column. Their customer statement opens.

### Download or email a customer's statement
Keywords: statement PDF, send statement, statement of account email, account summary
1. Open [Open items](/accounting/ar/open-items) and select the customer's name in the **Customer**
   column.
2. Optionally set the **Statement date** and **Start date**.
3. Select **Download PDF**, or select **Email statement**.
4. To email it, optionally write a **Note to the customer**, then select **Send statement**.

## Notes
Opening the page needs read access to accounts receivable. **Record payment** and
**Apply payment** only appear for people who can create customer payments.

The statement PDF shows the opening balance, every invoice, payment and credit in the period, the
balance due, the open invoices and how late they are. **Email statement** sends it to the invoice
recipients on the customer's email profile, needs permission to update accounts receivable, and is
recorded in the customer's collections history and under **Statements sent** on the statement page.
The statement uses your organization's customer statement template, which can be changed in
document templates.
