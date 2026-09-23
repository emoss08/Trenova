---
path: /fuel/feed-runs
aliases: [fuel card feed, fuel card sync, fuel transactions import, WEX feed, Comdata feed, held fuel rows, fuel import runs]
related:
  - /fuel/unassigned-cards
  - /fuel/purchases
  - /fuel/configuration-files/fuel-cards
  - /admin/integrations
---

## What it's for
Feed runs lists every time a connected fuel card feed read your transactions, and what became of the rows. Once a provider is connected under [Integrations](/admin/integrations), Trenova reads its transactions on an hourly schedule; each read appears here with how many rows it read, how many it posted as fuel purchases and how many it is holding.

Rows are held when the run cannot place them: usually the card has not been assigned to a tractor or driver yet, or the unit number on the receipt does not match a tractor. They wait here until that is fixed. The table shows **Provider**, **Read**, **Rows**, **Posted**, **Held** and **Ran**.

## Tasks

### Read a feed now
Keywords: sync fuel card, pull transactions, refresh feed
1. Open [Feed runs](/fuel/feed-runs).
2. Select **Sync now** in the page header and pick the provider (Comdata, EFS or WEX).
3. A message reports how many rows were posted of those read, and how many are waiting on a card assignment or a missing unit.

### See what a run held back
Keywords: held rows, unmatched fuel transactions, feed errors
1. Open [Feed runs](/fuel/feed-runs).
2. Select the run's row (or right-click it and choose **Review rows**). The rows it could not place are shown first.
3. Read the note on each held row to see why it was held.

### Post held rows after fixing them
Keywords: reprocess fuel rows, retry feed, resolve held rows
1. Assign the card on [Unassigned cards](/fuel/unassigned-cards), or add the missing tractor.
2. Open [Feed runs](/fuel/feed-runs) and select the run.
3. Select **Work rows out again**. Rows that can now be placed are posted as fuel purchases without fetching the statement again; any still waiting stay held.

## Notes
This page is only available when your organization runs its own trucks (asset operations). Viewing it needs read access to fuel purchase imports. Runs are started by the schedule or **Sync now**, so there is no create button. Rows a run posts appear on [Fuel purchases](/fuel/purchases).
