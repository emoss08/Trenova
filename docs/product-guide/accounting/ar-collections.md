---
path: /accounting/ar/collections
aliases: [collections worklist, dunning list, past due accounts, overdue customers, promise to pay, collector]
related:
  - /accounting/ar/open-items
  - /accounting/ar/customer-ledger
  - /accounting/ar/aging
---

## What it's for
Collections lists every customer with an overdue invoice, ranked so the first row is the account to
call next: a broken promise to pay or anything sixty days late comes first, an account waiting on an
open promise or on a follow-up that has not come round yet comes last. Each row shows what is
**Overdue**, the **Oldest** invoice's days past due, the **Promise** standing against the balance,
the next **Follow-up**, the **Collector** and the **Last contact**.

The figures above the table give the total **Overdue**, **Follow-ups due**, **Broken promises** and
**Unassigned** accounts. Select **Follow-ups due** or **Unassigned** to narrow the list to them.

## Tasks

### Chase today's overdue accounts
Keywords: who to call, follow-ups due, dunning
1. Open [Collections](/accounting/ar/collections).
2. Choose **Mine** to see the accounts you collect, and turn on **Due today only**.
3. Select a customer to open their account.

### Log a call or email
Keywords: record call, collection note, contact log
1. Open [Collections](/accounting/ar/collections) and select the customer.
2. Under **Log a contact**, choose **Call**, **Email** or **Note**.
3. Fill in **Spoke with** and **What happened**, and optionally **Follow up on**.
4. Select **Log it**.

### Record a promise to pay
Keywords: promise to pay, payment commitment, PTP
1. Open [Collections](/accounting/ar/collections) and select the customer.
2. Under **Log a contact**, choose **Promise to pay**.
3. Fill in **Promised amount** and **Will pay by**, then select **Record promise**.
4. The account waits on the promise and comes back the day after the promised date. Payments
   posted since the promise count toward it: it shows **Kept** once paid in full and **Broken** if the
   date passes short.

### Send a payment reminder now
Keywords: email reminder, dunning letter, past due notice
1. Open [Collections](/accounting/ar/collections) and select the customer.
2. Select **Send reminder**.
3. Optionally write a **Note to the customer**, then select **Send reminder**.
4. The reminder lists every overdue invoice not in dispute and goes to the invoice recipients on the
   customer's email profile. It is recorded in the account's **History**.

### Send a customer its statement
Keywords: statement of account, account statement email, balance summary
1. Open [Collections](/accounting/ar/collections) and select the customer.
2. Under **Receivables**, select **Statement**.
3. Select **Email statement**, optionally write a **Note to the customer**, then select
   **Send statement**.

### Assign a collector or set the next follow-up
Keywords: assign collector, owner, next follow-up
1. Open [Collections](/accounting/ar/collections) and select the customer.
2. Under **Collector and follow-up**, choose the **Collector**. They are notified.
3. Choose the **Next follow-up** date and select **Save follow-up**, or clear the date and select
   **Clear follow-up**.

## Notes
Opening the page needs read access to accounts receivable; logging contacts, recording promises,
assigning collectors and setting follow-ups needs permission to update accounts receivable. History is kept as it was recorded and cannot be edited.

Invoices sold to a non-recourse factor are the factor's to collect, so they are left out of the
worklist and its totals. Invoices sold to a recourse factor stay, because an unpaid one comes back to
you. Mark a factor as non-recourse on
[Factoring companies](/billing/configuration-files/factoring-companies).
