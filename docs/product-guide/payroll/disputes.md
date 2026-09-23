---
path: /payroll/disputes
aliases: [settlement disputes, pay disputes, driver pay questions, pay complaints, challenge settlement]
related:
  - /payroll/settlements
  - /payroll/workspace
  - /payroll/pay-codes
---

## What it's for
Disputes lists the questions and challenges drivers submit against their issued settlements from the driver portal (Dash). Payroll staff review each one, then either resolve it in the driver's favor, optionally with a correcting adjustment, or deny it with an explanation the driver can read.

The table shows each dispute's **Status** (Open, In review, Resolved, Denied or Withdrawn), **Driver**, **Category**, **Settlement**, **Description**, **Submitted** date and **Resolved** date. Disputes are raised by drivers, so there is no create button on this page.

## Tasks

### Start reviewing a dispute
Keywords: pick up dispute, triage dispute
1. Open [Disputes](/payroll/disputes).
2. Select an open dispute to see what the driver wrote, the settlement it is about and, when they picked one, the disputed line.
3. Select **Start review** to move it to In review.
4. To move several at once, tick the open disputes and select **Start review** in the bar that appears.

### Resolve a dispute in the driver's favor
Keywords: approve dispute, correct pay, pay adjustment
1. Open [Disputes](/payroll/disputes) and select the dispute.
2. Under **Resolve this dispute**, select **Resolve in driver's favor**.
3. Enter a **Resolution note**. The driver sees it word for word in Dash.
4. To correct the pay, turn on **Apply a correcting adjustment**, enter a description and an amount in dollars (positive adds pay, negative deducts), and optionally choose a **Pay code (optional)**.
5. Select **Resolve dispute**.

### Deny a dispute
Keywords: reject dispute
1. Open [Disputes](/payroll/disputes) and select the dispute.
2. Select **Deny**, enter a **Resolution note** explaining why, and select **Deny dispute**.

## Notes
Needs read access to settlement disputes to open the page, and the page is only available when the organization has the Asset operations capability.

A correcting adjustment is added as a line on the driver's open settlement; if the driver has none, an off-cycle draft settlement is created for it. Choosing a pay code routes the adjustment to that code's GL account when the settlement is posted.

Resolved, denied and withdrawn disputes are final: the panel shows the resolution note, who resolved it and when, and whether a correcting adjustment was applied.
