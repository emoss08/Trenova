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
form, saved once, in three tabs: **Pay period** (when each period ends, when it is paid and when a
move's pay counts toward it), **Review & approval** (which settlements are flagged for review and
what approving one does) and **Advances & escrow** (how pay advances are recovered and the interest
owner-operators earn on escrow).

Payroll and settlement managers use it to set the settlement cycle and decide how much of it runs
without manual review.

## Tasks

### Set the pay period and when pay is earned
Keywords: weekly pay, biweekly, pay date, pay on delivery, pay trigger
1. Open [Settlement control](/admin/settlement-control).
2. On **Pay period**, under **Schedule**, choose the **Pay period** (**Weekly**, **Biweekly** or
   **Monthly**) and the **Last day of the period**.
3. Enter the **Pay delay**, the days between the end of a period and its pay date (0 to 30).
   A period ends at midnight in your organization's time zone on its last day.
4. Turn on **Generate settlements when a period ends** to have each period's settlement batch
   drafted without anyone starting it.
5. Under **Earning pay**, choose when **Pay is earned when**: **Move completed**, **Shipment
   delivered**, **POD received (ready to invoice)** or **Shipment invoiced**.
6. Turn on **Add new pay to open drafts** so pay a driver earns after their draft settlement was
   made joins it.
7. Select **Save changes**.

### Flag unusual settlements
Keywords: variance, exception, pay anomaly
1. Open [Settlement control](/admin/settlement-control) and select **Review & approval**.
2. Under **Exceptions**, set **Flag a change in net pay over**, the percentage a settlement's net
   pay may move from the driver's average before it is flagged, up to 999.9999. Enter 0 to stop
   flagging.
3. While the check is on, set **Average over**, how many of the driver's previous settlements make
   up the average (1 to 52).
4. Select **Save changes**.

### Automate approval and posting
Keywords: auto approve, auto post, separate approver
1. Open [Settlement control](/admin/settlement-control) and select **Review & approval**.
2. Under **Approval**, turn on **Approve clean settlements automatically** to approve settlements
   with no exceptions without review.
3. Turn **Require a separate approver** off if one person both submits and approves settlements.
4. Turn on **Post to the ledger on approval** to post a settlement to the general ledger as it is
   approved.
5. Select **Save changes**.

### Recover advances and set escrow interest
Keywords: escrow, owner operator, 376.12, carry forward, advance recovery
1. Open [Settlement control](/admin/settlement-control) and select **Advances & escrow**.
2. Turn on **Recover advances in full** to take an advance in full even when it is more than a
   settlement's net pay, carrying the shortfall to the next settlement. Off, recovery stops at zero
   net pay.
3. Under **Escrow interest**, enter the **Yearly interest rate** new escrow accounts start with,
   and choose how often to **Pay interest**: **Monthly**, **Every two months** or **Quarterly**.
4. Select **Save changes**.

## Notes
The page is only available to organizations with asset operations turned on. Opening it needs read
access to settlement control; saving needs update access. A save the server refuses opens the tab
holding the setting to fix. The yearly interest rate applies to new escrow accounts unless an
account sets its own; see [Escrow accounts](/payroll/escrow-accounts).
