---
path: /payroll/pay-codes
aliases: [earning codes, deduction codes, pay types, payroll codes, GL mapping for pay]
related:
  - /payroll/deductions
  - /payroll/earnings
  - /payroll/pay-profiles
---

## What it's for
Pay codes are your catalog of earning and deduction codes. Every recurring earning, recurring deduction and manual settlement adjustment can carry a code, and the code decides whether it adds or withholds pay, which GL account its settlement lines post to, whether it is taxable, and whether it counts toward a driver's guaranteed minimum. Payroll and accounting managers maintain it.

The table shows each code's **Status**, **Direction**, **Code**, **Name**, **Behavior**, **GL account** and **Default amount**.

## Tasks

### Add a pay code
Keywords: new earning code, new deduction code, create pay type
1. Open [Pay codes](/payroll/pay-codes).
2. Select **New pay code**.
3. Choose the **Direction**: earning codes add pay and deduction codes withhold it.
4. Enter the **Code** (uppercase letters, digits, dashes or underscores, shown on statements) and the **Name**, and optionally a **Description**.
5. Optionally choose a **GL account** and a **Default amount**, which prefills recurring earnings and deductions that use the code.
6. For an earning code, set **Taxable** and **Counts toward guaranteed minimum**.
7. Select **Save**.

### Edit a pay code
Keywords: change GL account for pay code
1. Open [Pay codes](/payroll/pay-codes) and select the code.
2. Change the fields, including **Status**, then select **Save**. The direction cannot be changed after the code is created.

### Activate or deactivate pay codes
Keywords: retire pay code, hide pay code
1. Open [Pay codes](/payroll/pay-codes) and tick the codes.
2. Select **Update status**, then **Activate** or **Deactivate**.

## Notes
Needs read access to pay codes to open the page; creating and editing need the matching permissions. The page is only available when the organization has the Asset operations capability.

A code with no GL account posts to the accounting control defaults. Taxable amounts post as earnings; non-taxable ones, such as per diem and stipends, post as reimbursements.

Inactive codes stay on historical records but no longer appear in the dropdowns for new entries.
