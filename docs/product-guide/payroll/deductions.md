---
path: /payroll/deductions
aliases: [standing deductions, withholdings, truck lease deduction, insurance deduction, escrow deduction, loan repayment, chargebacks]
related:
  - /payroll/earnings
  - /payroll/pay-codes
  - /payroll/escrow-accounts
  - /payroll/workspace
---

## What it's for
Recurring deductions are standing amounts withheld from a driver's settlements, such as insurance, lease payments, escrow contributions and loan repayments. Each one belongs to one driver, uses a deduction pay code, and is withheld automatically every settlement or once a month until it is paused, reaches its end date, or hits its total cap.

The table shows each deduction's **Status**, **Driver**, **Code**, **Description**, **Frequency**, **Amount** and **Deducted / cap**.

## Tasks

### Add a recurring deduction
Keywords: new deduction, set up lease payment, withhold insurance
1. Open [Recurring deductions](/payroll/deductions).
2. Select **New recurring deduction**.
3. Choose the **Driver**, the **Pay code** and the **Frequency** (every settlement, or monthly on the first settlement of each month).
4. Enter a **Description** and the **Amount per application**. Optionally set a **Total cap**, after which the deduction stops on its own.
5. Set the **Start date** and, if it should end, an **End date**.
6. To send the withheld amount into the driver's escrow account, turn on **Contribute to escrow account**.
7. Select **Save**.

### Edit a deduction
Keywords: change deduction amount
1. Open [Recurring deductions](/payroll/deductions) and select the deduction.
2. Change the fields, including **Status**, then select **Save**.

### Pause or resume deductions
Keywords: skip deduction, stop deduction temporarily
1. Open [Recurring deductions](/payroll/deductions) and tick the deductions.
2. Select **Update status**, then **Pause** or **Resume**.

## Notes
Needs read access to recurring deductions to open the page; creating and editing need the matching permissions. The page is only available when the organization has the Asset operations capability.

Paused deductions are skipped on future settlements and keep their history. A completed deduction has reached its cap and has stopped permanently.

An escrow contribution links to the driver's active escrow account when saved, so open an escrow account for the driver in [Escrow accounts](/payroll/escrow-accounts) first. It stops automatically at the account's funding target, and the escrow setting cannot be changed after the deduction is created.

Deductions can also be paused from the right rail of the [Workspace](/payroll/workspace).
