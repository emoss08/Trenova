---
path: /payroll/settlement-batches
aliases: [payroll run, pay run, payroll batch, payroll export, payroll file, batch settlements]
related:
  - /payroll/workspace
  - /payroll/settlements
  - /payroll/pay-events
---

## What it's for
Settlement batches are driver payroll runs. Each batch groups the settlements generated for one pay period, and this page is where payroll staff generate a batch for the current period, check how many settlements it holds and how many have exceptions, and download the payroll file.

The table shows each batch's **Status**, **Batch** name, **Period**, **Pay date**, number of **Settlements**, number of **Exceptions**, **Total gross** and **Total net**.

## Tasks

### Generate a settlement batch for the current period
Keywords: run payroll, create batch, new pay run
1. Open [Settlement batches](/payroll/settlement-batches).
2. Select **New settlement batch**. The panel shows the **Current pay period** and its pay date.
3. Optionally enter a **Batch name** (left blank, it is named after the period end date) and **Notes** for reviewers.
4. Select **Save**. A draft settlement is created for every driver with accrued pay in the current period.

### Review a batch and its exceptions
Keywords: batch exceptions, batch totals
1. Open [Settlement batches](/payroll/settlement-batches).
2. Select a batch to see its period and pay date, its **Settlements**, **Exceptions**, **Total gross** and **Total net**, and every settlement in it with driver, status, gross and net.
3. Settlements flagged with exceptions are marked with a warning icon. Work them in the [Workspace](/payroll/workspace).

### Export the payroll file
Keywords: payroll CSV, download payroll, export to payroll provider
1. Open [Settlement batches](/payroll/settlement-batches) and select the batch.
2. Select **Export payroll CSV** to download the batch as a CSV file.

## Notes
Needs read access to driver settlements to open the page, and the page is only available when the organization has the Asset operations capability.

Clean settlements can be approved automatically, depending on your settlement control policy; anything with exceptions stays in review.
