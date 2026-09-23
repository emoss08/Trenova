---
path: /fuel/jurisdiction-mileage
aliases: [IFTA miles, state miles, miles by state, deadhead miles, manual miles, bobtail miles, yard moves, jurisdiction miles]
related:
  - /fuel/ifta-returns
  - /fuel/purchases
  - /equipment/tractors
---

## What it's for
Jurisdiction mileage holds miles by state or province that dispatch did not compute: deadhead, bobtailing to a shop, yard moves, repositioning between customers, and units without telematics. Routed miles from completed shipment moves are captured automatically; the manual miles entered here are added to them on the quarter's IFTA return at its next recompute.

Fuel tax and compliance staff use it. The table shows **Travelled**, **Tractor**, **Jurisdiction**, **Miles**, **Loaded**, **Source** and **Notes**. Rows written by the distance provider or telematics also appear here, read-only.

## Tasks

### Add miles the system did not see
Keywords: enter manual miles, add deadhead, record yard move, state miles entry
1. Open [Jurisdiction mileage](/fuel/jurisdiction-mileage).
2. Select **New jurisdiction mileage**.
3. Under **Where and when**, pick the **Tractor** and **Jurisdiction**, and set **Travelled on** (the day the miles were run, which fixes the quarter; it cannot be in the future).
4. Enter the **Miles** and turn on **Under load** if the tractor was loaded.
5. Add **Notes** explaining why these miles exist, so an auditor can see the reason, then select **Save**.

### Correct a move's routed miles
Keywords: fix IFTA miles, wrong state miles, override move miles
1. Open [Jurisdiction mileage](/fuel/jurisdiction-mileage) and select the system-written row to see which move it came from.
2. System rows are read-only. Add a manual entry for the same move instead; it replaces those rows on the return.

### Change or delete a manual entry
Keywords: edit miles, remove mileage entry
1. Open [Jurisdiction mileage](/fuel/jurisdiction-mileage).
2. Select the entry's row, change it and select **Save**.
3. To delete it, right-click the row, choose **Delete**, then **Delete entry**.

## Notes
This page is only available when your organization runs its own trucks (asset operations). Viewing it needs read access to jurisdiction mileage, and **Delete** needs permission to delete it.

Manual miles are never subtracted from routed miles, so do not enter miles a move already carries unless the entry names that move. After adding, changing or deleting miles, recompute the quarter on [IFTA returns](/fuel/ifta-returns) to bring the figures up to date.
