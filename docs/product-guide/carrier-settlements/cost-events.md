---
path: /carrier-settlements/cost-events
aliases: [carrier costs, purchased transportation cost, accrued carrier cost, carrier accruals, buy side cost]
related:
  - /carrier-settlements/workspace
  - /carrier-settlements/settlements
  - /carrier-settlements/invoice-matching
---

## What it's for
Carrier cost events are the purchased-transportation costs recorded automatically as carrier-covered shipments deliver. Each one is an amount owed to a carrier on a shipment, and together they are the source every carrier settlement is built from. AP and accounting staff use this page to see what has accrued for each carrier and whether it has been settled yet.

The table shows **Status** (pending, attached, settled or voided), **Type**, **Carrier**, **Pro #**, **Description**, **Accrued** date and **Amount**. Cost events are created by the system, not by hand.

## Tasks

### Look up what a carrier is owed
Keywords: carrier accruals, unsettled carrier cost
1. Open [Cost events](/carrier-settlements/cost-events).
2. Search for the carrier or pro number, or use **Filter** on **Status** or **Carrier** to see pending cost.
3. Select a cost event to see its **Type**, **Description**, **Accrued** date, **Pro number**, the **Settlement** it is on (or **Unsettled**) and the **Amount**.

### Settle pending cost
Keywords: pay carrier cost, generate carrier settlement
1. Open [Workspace](/carrier-settlements/workspace).
2. Select **Generate settlements** to roll pending cost events into carrier settlements.

## Notes
Needs read access to carrier settlements, and the page is only available when the organization has the Brokerage capability.

Voiding a carrier settlement in the workspace releases its cost events back to the accrual pool so they can be settled again.
