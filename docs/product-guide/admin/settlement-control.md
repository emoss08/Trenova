---
path: /admin/settlement-control
aliases: [driver pay settings, pay period settings, payroll settings, driver settlement rules, escrow interest]
related:
  - /payroll/settlements
  - /payroll/settlement-batches
  - /payroll/escrow-accounts
  - /admin/accounting-control
---

## What it's for
Settlement control holds the organization's rules for driver settlements. The page is one settings
form in four cards: **Pay period** (how often drivers are settled and when pay accrues), **Workflow
automation** (automatic batches, approval and posting), **Exception detection** (flagging
settlements that differ from a driver's recent history) and **Escrow interest** (interest on
owner-operator escrow).

Payroll and settlement managers use it to set the settlement cycle and decide how much of it runs
without manual review.

## Tasks

### Set the pay period and pay trigger
Keywords: weekly pay, biweekly, pay date, pay on delivery
1. Open [Settlement control](/admin/settlement-control).
2. In **Pay period**, choose the **Frequency** (**Weekly**, **Biweekly** or **Monthly**) and the
   **Period end day**.
3. Enter the **Pay Delay (days)** between the period end and the pay date.
4. Choose the **Pay trigger**, the milestone at which driver pay accrues: **Move completed**,
   **Shipment delivered**, **POD received (ready to invoice)** or **Shipment invoiced**.
5. Select **Save changes**.

### Automate settlement batches and approval
Keywords: auto approve, auto post, auto batch, carry forward
1. Open [Settlement control](/admin/settlement-control).
2. In **Workflow automation**, turn on the steps to automate: **Auto-generate batches**,
   **Auto-approve clean settlements**, **Auto-attach new pay to open drafts** and **Auto-post on
   approval**.
3. Turn on **Allow negative net (carry forward)** to carry a negative balance to the next
   settlement instead of capping deductions.
4. Select **Save changes**.

### Flag unusual settlements
Keywords: variance, exception, pay anomaly
1. Open [Settlement control](/admin/settlement-control).
2. In **Exception detection**, set the **Variance threshold** percentage and the **Lookback
   (settlements)** count used for the driver's trailing average.
3. Select **Save changes**.

### Set escrow interest
Keywords: escrow, owner operator, 376.12
1. Open [Settlement control](/admin/settlement-control).
2. In **Escrow interest**, enter the **Default annual interest rate** and the **Accrual Frequency
   (months)** (1 to 3).
3. Select **Save changes**.

## Notes
The page is only available to organizations with asset operations turned on. Opening it needs read
access to settlement control; saving needs update access. The default interest rate applies to new
escrow accounts unless an account overrides it; see
[Escrow accounts](/payroll/escrow-accounts).
