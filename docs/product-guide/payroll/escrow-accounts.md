---
path: /payroll/escrow-accounts
aliases: [maintenance escrow, owner-operator escrow, escrow ledger, escrow interest, 376.12(k), reserve account]
related:
  - /payroll/deductions
  - /payroll/workspace
  - /payroll/settlements
---

## What it's for
Escrow accounts holds owner-operator maintenance escrow. Each owner-operator has at most one active account with a funding target and an annual interest rate. Contributions flow in from settlements through a recurring deduction, interest accrues at least quarterly, and every movement is kept on a transaction ledger, as 49 CFR 376.12(k) requires.

The table shows each account's **Status**, **Driver**, **Balance**, **Target**, how far it is **Funded**, **Interest**, the **Opened** date and the **Last interest** date.

## Tasks

### Open an escrow account
Keywords: new escrow, start escrow for owner-operator
1. Open [Escrow accounts](/payroll/escrow-accounts).
2. Select **New escrow account**.
3. Choose the **Driver**. Only owner-operators are listed.
4. Enter the **Funding target** (contributions stop once the balance reaches it) and the **Annual interest rate**, which defaults to your settlement control rate.
5. Select **Save**.
6. To start contributions, add a recurring deduction for the driver in [Recurring deductions](/payroll/deductions) with **Contribute to escrow account** turned on.

### Review an account's ledger
Keywords: escrow balance, escrow transactions, escrow history
1. Open [Escrow accounts](/payroll/escrow-accounts) and select the account.
2. See the **Balance**, **Target** and **Interest rate**, and every entry in the **Transaction ledger** with its **Date**, **Type**, **Description**, **Amount** and running **Balance**.

### Record an escrow adjustment
Keywords: apply escrow to repair, deposit to escrow, escrow withdrawal
1. Open [Escrow accounts](/payroll/escrow-accounts) and select an active account.
2. Select **Record adjustment**.
3. Enter the amount in dollars (positive deposits into escrow, negative applies funds out, for example a repair paid from escrow) and a description.
4. Select **Record**.

### Close an escrow account
Keywords: refund escrow, driver leaving, terminate escrow
1. Open [Escrow accounts](/payroll/escrow-accounts) and select the active account.
2. Select **Close account** and confirm with **Close account**. The remaining balance is refunded to the driver as a ledger entry.
3. To close several at once, tick them and select **Close accounts**. From the bulk action, accounts that still hold a balance fail to close until their funds are refunded or applied.

## Notes
Needs read access to escrow accounts to open the page; the other actions need the matching permissions. The page is only available when the organization has the Asset operations capability.

Closing cannot be undone. Closed accounts stop accepting contributions and accruing interest. Adjustment descriptions are recorded permanently on the ledger.

A driver's escrow balance also appears in the right rail of the [Workspace](/payroll/workspace), where **View ledger** opens this page.
