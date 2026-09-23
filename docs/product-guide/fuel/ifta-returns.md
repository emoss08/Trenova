---
path: /fuel/ifta-returns
aliases: [IFTA, fuel tax return, quarterly fuel tax, IFTA worksheet, IFTA report, fuel tax filing, IFTA amendment]
related:
  - /fuel/purchases
  - /fuel/jurisdiction-mileage
  - /fuel/configuration-files/ifta-tax-rates
  - /admin/distance-controls
---

## What it's for
IFTA returns is the quarterly fuel tax worksheet. For each quarter it lists every jurisdiction's miles and tax-paid gallons, works out the fleet MPG they are taxed through, and totals what the quarter owes or is owed. Miles come from completed shipment moves plus the manual entries on [Jurisdiction mileage](/fuel/jurisdiction-mileage); gallons come from [Fuel purchases](/fuel/purchases); rates come from [IFTA tax rates](/fuel/configuration-files/ifta-tax-rates).

Fuel tax and compliance staff use it to generate the return, check it, lock it, record that it was filed and amend it later. The page shows the **Total miles**, **Tax-paid gallons**, **Fleet MPG** and **Status**, then the **Jurisdiction lines** with each line's **Taxable miles**, **Net taxable gal**, **Rate**, **Tax due**, **Surcharge** and **Line total**, and a section on **What the figures leave out**.

## Tasks

### Generate a quarter's return
Keywords: start IFTA return, create fuel tax return, run IFTA
1. Open [IFTA returns](/fuel/ifta-returns). It opens on the most recently completed quarter.
2. Pick the **Year** and **Quarter** at the top of the page.
3. If the quarter shows **Not generated**, select **Generate the return**.

### Bring a draft return up to date
Keywords: recompute IFTA, refresh return, update figures
1. Open [IFTA returns](/fuel/ifta-returns) and pick the quarter.
2. Select **Recompute** to rebuild every line from the miles, fuel and rates on file now. Do this after adding or deleting fuel purchases or manual miles.
3. Check **What the figures leave out** for moves with no tractor, mileage mismatches and moves that were never broken down by jurisdiction.
4. A line marked **No rate** is missing its jurisdiction's tax rate; select **Add rate** to enter it on [IFTA tax rates](/fuel/configuration-files/ifta-tax-rates), then recompute.

### Finalize and file the return
Keywords: lock IFTA return, mark filed, filing reference, export IFTA
1. Open [IFTA returns](/fuel/ifta-returns) and pick the quarter.
2. Select **Finalize…**, then **Finalize return**. This recomputes and locks the worksheet.
3. Select **Export CSV** to download every line, each fuel type's subtotal and the grand total.
4. After filing with your base jurisdiction, select **Mark filed…**, enter the **Filed on** date and optional **Filing reference**, then select **Mark filed**.

### Reopen or amend a return
Keywords: unlock IFTA, correct filed return, amended return
1. Open [IFTA returns](/fuel/ifta-returns) and pick the quarter.
2. For a finalized return that has not been filed, select **Reopen…**, enter a **Reason** and select **Reopen return**.
3. For a filed return, select **Amend…**, enter a **Reason** and select **Open the amendment**.

### Delete a draft return
Keywords: remove IFTA draft, start over
1. Open [IFTA returns](/fuel/ifta-returns) and pick the quarter.
2. Select **Delete draft…**, then **Delete draft**. Only a draft can be deleted.

## Notes
This page is only available when your organization runs its own trucks (asset operations). Viewing it needs read access to IFTA returns. Each action needs its own permission: generating or amending needs create, **Recompute** needs update, finalizing needs approve, reopening needs reopen, marking filed needs submit, **Export CSV** needs export and deleting needs delete.

Draft figures move with the data until the return is finalized. A return cannot be finalized while any member line is missing a rate. Reopening a finalized return requires a reason, which is audited.

When some moves were never broken down by jurisdiction, people with permission to manage IFTA returns can select **Backfill jurisdiction miles…** under **What the figures leave out** to re-route those moves. Jurisdiction miles are only captured while **Capture jurisdiction miles** is on in [Distance controls](/admin/distance-controls).
