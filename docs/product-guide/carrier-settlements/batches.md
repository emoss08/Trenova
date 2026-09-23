---
path: /carrier-settlements/batches
aliases: [carrier settlement batches, AP run, carrier pay run, remittance file, carrier payment batch]
related:
  - /carrier-settlements/workspace
  - /carrier-settlements/settlements
  - /carrier-settlements/cost-events
---

## What it's for
Carrier settlement batches are the accounts payable runs for carriers on brokered freight. Each batch groups the carrier settlements generated for one pay period. This page is where AP staff generate a batch for the current period, check its totals, and download the remittance file.

The table shows each batch's **Status**, **Batch** name, **Period**, **Pay date**, number of **Settlements**, **Total gross** and **Total net**.

## Tasks

### Generate a carrier settlement batch
Keywords: run AP, create carrier batch, new pay run
1. Open [Settlement batches](/carrier-settlements/batches).
2. Select **New carrier settlement batch**. The panel shows the **Current pay period** and its pay date.
3. Optionally enter a **Batch name** (left blank, it is named after the period end date) and **Notes**.
4. Select **Save**. A draft settlement is created for every carrier with pending cost events in the current period.

### Review a batch
Keywords: batch totals, carrier batch detail
1. Open [Settlement batches](/carrier-settlements/batches).
2. Select a batch to see its period and pay date, its **Settlements**, **Total gross** and **Total net**, and each settlement in it with carrier, status, gross and net.
3. Process the settlements in the [Workspace](/carrier-settlements/workspace).

### Export the remittance file
Keywords: remittance CSV, carrier payment file, download AP file
1. Open [Settlement batches](/carrier-settlements/batches) and select the batch.
2. Select **Export remittance CSV** to download it.

## Notes
Needs read access to carrier settlements to open the page, and the page is only available when the organization has the Brokerage capability.

Depending on your carrier settlement control policy, settlements can post automatically when approved.
