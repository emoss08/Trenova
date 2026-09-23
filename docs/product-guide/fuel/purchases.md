---
path: /fuel/purchases
aliases: [fuel receipts, fuel transactions, diesel purchases, fuel card statement, fuel import, fuel log, IFTA gallons]
related:
  - /fuel/ifta-returns
  - /fuel/feed-runs
  - /fuel/configuration-files/fuel-cards
  - /fuel/unassigned-cards
  - /equipment/tractors
---

## What it's for
Fuel purchases lists every gallon bought for a tractor, whether keyed in by hand from a receipt or imported from a fuel card statement (Comdata, EFS or WEX). Tax-paid gallons credit the jurisdiction they were bought in on the quarterly IFTA return, so this is the fuel side of [IFTA returns](/fuel/ifta-returns).

Fleet, fuel and compliance staff use it. The table shows **Purchased**, **Tractor**, **Worker**, **Jurisdiction**, **Vendor**, **Fuel**, **Gallons**, **Amount**, **Tax**, **Source**, **Reference** and **Updated**.

## Tasks

### Record a fuel purchase from a receipt
Keywords: add fuel receipt, key fuel, enter diesel purchase
1. Open [Fuel purchases](/fuel/purchases).
2. Select **New fuel purchase** (then **New fuel purchase** again if a menu opens).
3. Under **Unit, driver & place**, pick the **Tractor**, optionally the **Driver** (filled from the tractor's primary driver when left empty), the **Purchased at** date and time, the **Jurisdiction**, and the **Vendor** and **City**.
4. Under **Fuel**, choose the **Fuel type** and **Unit**, and enter the **Quantity**, **Total paid**, **Currency** and **Odometer**.
5. Under **Card & tax**, pick the **Fuel card** (or enter the card's last four digits), the **Transaction reference**, and whether **Fuel tax paid at the pump** applies. Add any **Notes**.
6. Select **Save**. Attach the receipt on the purchase's **Documents** tab.

### Import a fuel card statement
Keywords: upload Comdata, EFS statement, WEX file, bulk fuel import
1. Open [Fuel purchases](/fuel/purchases).
2. Select **New fuel purchase** and choose **Import card statement**.
3. Pick the **Provider**, and optionally a **Default card**, **Default fuel type** and **Currency**, then upload the statement. Use **Download template** if you are unsure of the layout.
4. Review the rows: new rows, **Duplicates in file**, rows **Already on file** and **Errors**. Nothing is recorded until you confirm.
5. Confirm the import, or select **Discard** to drop it.

### Correct or delete a purchase
Keywords: fix fuel purchase, remove duplicate fuel, delete fuel receipt
1. Open [Fuel purchases](/fuel/purchases).
2. Select the purchase's row to change it and select **Save**.
3. To delete it, right-click the row, choose **Delete**, then **Delete purchase**.

### Find purchases
Keywords: search fuel, filter by jurisdiction, fuel by date
1. Open [Fuel purchases](/fuel/purchases).
2. Type in the search box, or use **Filter** in the table toolbar, for example on **Purchased** date, **Jurisdiction** or **Fuel**.

## Notes
This page is only available when your organization runs its own trucks (asset operations). Viewing it needs read access to fuel purchases; importing statements needs permission to create fuel purchase imports, and **Delete** needs permission to delete fuel purchases.

DEF, reefer and other non-IFTA products are tracked as spend only. Litres are converted to US gallons for the return. After deleting a purchase, recompute the quarter's return on [IFTA returns](/fuel/ifta-returns) to take it out of the figures; importing the same statement again records the row afresh.
