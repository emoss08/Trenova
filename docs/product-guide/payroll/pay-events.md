---
path: /payroll/pay-events
aliases: [driver earnings, accrued pay, earnings ledger, unsettled pay, load pay, trip pay]
related:
  - /payroll/workspace
  - /payroll/settlements
  - /payroll/pay-profiles
---

## What it's for
Pay events are the driver earnings recorded automatically as shipments deliver. Each one is the pay a driver earned on one shipment, worked out from their pay profile, and it is the source every settlement is built from. Payroll staff use this page to check what a driver has earned, see how a load's pay was calculated, and hold pay that should not settle yet.

The table shows **Status** (Accrued, Settled or Voided), **Driver**, **Pro #**, **Earned** date, **Miles**, **Breakdown** and **Gross pay**. Pay events are created by the system, not by hand.

## Tasks

### See how a load's pay was calculated
Keywords: pay breakdown, pay detail, check driver pay
1. Open [Pay events](/payroll/pay-events).
2. Search for the pro number or driver, or use **Filter** and **Sort**.
3. Select the pay event to see each pay **Component** with its **Qty × rate**, **Amount** and the **Total**.

### Hold pay so it does not settle yet
Keywords: defer pay, hold pay event, disputed load
1. Open [Pay events](/payroll/pay-events).
2. Tick the accrued pay events to hold and select **Hold** in the bar that appears. For a single event you can also hover its status and select the hold icon.
3. Enter the reason and select **Hold pay**. Held events show **Held** and are skipped by settlement generation until released.

### Release held pay
Keywords: unhold, release hold
1. Open [Pay events](/payroll/pay-events).
2. Select **Held** on the event's status to release it, or tick several held events and select **Release holds**.

## Notes
Needs read access to driver settlements, and the page is only available when the organization has the Asset operations capability.

Only accrued pay events that are not already on hold can be held. One reason is recorded on every event held together. Released events settle normally on the next settlement generation.
