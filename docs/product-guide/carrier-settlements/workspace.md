---
path: /carrier-settlements/workspace
aliases: [carrier settlement workspace, carrier pay, pay carriers, purchased transportation, carrier payables, AP run, brokerage payables]
related:
  - /carrier-settlements/settlements
  - /carrier-settlements/batches
  - /carrier-settlements/cost-events
  - /carrier-settlements/invoice-matching
---

## What it's for
The carrier settlement workspace is where accounts payable staff run the current carrier pay period for brokered freight from one screen. The top strip shows the pay period and pay date, how many settlements are **In pipeline**, how many are **Posted / paid**, the **Period net payable**, the **Unsettled cost** not yet on a settlement, and whether an **Open batch** exists for the period.

Below it, the queue on the left lists every carrier settlement in the period, the middle pane shows the selected settlement's lines, remittance details, rate confirmations, invoice matches and history with the actions for its current status, and the right rail shows the carrier's unsettled cost, recent settlements and AP subledger.

## Tasks

### Generate the period's carrier settlements
Keywords: build carrier settlements, run AP, create carrier statements
1. Open [Workspace](/carrier-settlements/workspace).
2. Select **Generate settlements**. It builds one settlement per carrier from the pending cost events. If a batch for the period is already open, generating tops it up.

### Review and approve a carrier settlement
Keywords: approve carrier pay, submit carrier settlement, post carrier settlement
1. Open [Workspace](/carrier-settlements/workspace).
2. Find the carrier with **Search carrier or number**, or narrow the queue with the chips (**All**, **Draft**, **Pending**, **Approved**, **Posted**, **Paid**).
3. Select the settlement and check its lines, the **Remittance** details, **Rate confirmations** and **Invoice matches**.
4. On a draft, select **Add adjustment** to raise or reduce the payable, or **Recalculate** to rebuild it from current cost events.
5. Select **Submit for approval**, then **Approve** (or **Reject** with a reason to send it back to draft).
6. Select **Post to GL**, then **Mark paid**, choose the **Payment method** and enter an optional **Reference**.

### Act on several settlements at once
Keywords: bulk approve carrier settlements, bulk post, bulk mark paid
1. Open [Workspace](/carrier-settlements/workspace).
2. Tick the settlements in the queue, or tick the box above the list to select all visible ones.
3. Use **Submit**, **Approve**, **Post** or Mark Paid in the bar at the bottom of the queue. Each button shows how many selected settlements are in a status it applies to; the rest are skipped.

### Download the remittance file
Keywords: remittance CSV, carrier payment file
1. Open [Workspace](/carrier-settlements/workspace) and select a settlement that belongs to a batch.
2. Select **Batch CSV** to download the remittance CSV for that settlement's batch.

### Void a carrier settlement
1. Open [Workspace](/carrier-settlements/workspace) and select the settlement.
2. Select **Void**, enter the reason, and confirm with **Void**.

## Notes
Needs read access to carrier settlements, and the page is only available when the organization has the Brokerage capability.

Voiding releases the settlement's cost events back to the accrual pool and reverses any GL postings. Paid and voided settlements cannot be voided.

**Add adjustment** takes dollars, not cents: a positive amount increases the payable and a negative amount reduces it. Marking a settlement paid records the cash journal and the payment on the carrier's AP subledger.

If a settlement shows no remit-to address, set it on the carrier record.
