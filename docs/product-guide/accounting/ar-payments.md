---
path: /accounting/ar/payments
aliases: [cash receipts, receive payment, customer remittance, cash application, apply cash, payments received]
related:
  - /accounting/ar/open-items
  - /accounting/ar/customer-ledger
  - /accounting/ar/invoices
  - /accounting/reconciliation/bank-receipts
---

## What it's for
Customer payments is where AR staff post money received from customers and apply it to their
open invoices, all in one step, with the matching general ledger entries. Figures at the top show
**Posted today**, **Unapplied cash** and **Reversed — 30 days**. The table lists each payment's
**Status**, **Reference**, **Customer**, **Method**, **Payment date**, **Accounting date**,
**Amount**, **Applied**, **Unapplied** and **Invoices**.

Selecting a payment opens its detail: the dates, method and reference, the invoices it was
applied to, any short-pay written off, and the **GL postings** it created.

## Tasks

### Record a customer payment
Keywords: post payment, receive check, ACH received, apply payment to invoices
1. Open [Customer payments](/accounting/ar/payments).
2. Select the New customer payment button in the table toolbar. The **Record payment** panel
   opens.
3. Pick the **Customer** and fill in **Payment amount**, **Payment method**, **Payment date**
   (when the funds arrived) and **Accounting date** (the GL date, which must fall in an open
   fiscal period). Add a **Reference number** such as the check number or ACH trace, and a
   **Memo** if needed.
4. Under **Apply to open invoices**, tick the invoices this payment pays and enter the
   **Applied** amount for each, or select **Auto-apply oldest first**. Enter a **Short-pay**
   amount on an invoice to write off the difference.
5. Select **Post payment**. Anything not applied stays on the payment as unapplied cash.

### Apply unapplied cash later
Keywords: apply remaining balance, on-account cash, unapplied payment
1. Open [Customer payments](/accounting/ar/payments) and select the payment.
2. Select **Apply unapplied**.
3. Set the **Accounting date**, tick the invoices and enter the amounts, then select
   **Apply cash**.

### Reverse a payment
Keywords: NSF, bounced check, undo payment, wrong customer, void payment
1. Open [Customer payments](/accounting/ar/payments) and select the payment.
2. Select **Reverse**.
3. Enter the **Accounting date** and **Reason**, then select **Reverse payment**. The cash is
   backed out, the invoices it paid are reopened and a reversing GL entry is posted.

### Check a payment's ledger entries
Keywords: journal entry for payment, GL trace
1. Open [Customer payments](/accounting/ar/payments) and select the payment.
2. Look under **GL postings**, or select **Open full view** to see the entries on their own page.
3. Select the customer name under the amount to open their
   [Customer ledger](/accounting/ar/customer-ledger).

## Notes
- Opening the page needs read access to customer payments. **Apply unapplied** and **Reverse**
  need update access and only appear on posted payments; **Apply unapplied** only shows when the
  payment still has unapplied cash.
- The payment cannot be posted if the applied total is more than the payment amount, or if any
  invoice would be paid past its open balance.
- A reversal cannot be undone.
- **Apply payment** on [Open items](/accounting/ar/open-items) opens this page's payment panel
  with the customer and chosen invoices already filled in.
