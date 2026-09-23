---
path: /fuel/configuration-files/ifta-tax-rates
aliases: [IFTA rate matrix, fuel tax rates, state fuel tax, diesel tax rates, IFTA rates, surcharge per gallon]
related:
  - /fuel/ifta-returns
  - /fuel/purchases
---

## What it's for
IFTA tax rates holds the per-gallon rates each jurisdiction publishes for a quarter and fuel type, as the IFTA rate matrix lists them, along with any per-gallon surcharge. The quarterly [IFTA returns](/fuel/ifta-returns) price every jurisdiction line with these rates, and a return cannot be finalized while any of its member lines is missing one.

Fuel tax and compliance staff enter or import the rates each quarter. The table shows **Year**, **Quarter**, **Jurisdiction**, **Fuel**, **Rate / gal**, **Surcharge / gal**, **Source** and **Updated**.

## Tasks

### Add a tax rate
Keywords: enter IFTA rate, new fuel tax rate, missing rate
1. Open [IFTA tax rates](/fuel/configuration-files/ifta-tax-rates).
2. Select **New IFTA tax rate** (then **New IFTA tax rate** again if a menu opens).
3. Under **Period & product**, set the **Year**, **Quarter**, **Jurisdiction** and **Fuel type**.
4. Enter the **Rate per gallon** in USD per US gallon as printed in the matrix, and the **Surcharge per gallon** if the jurisdiction publishes one.
5. Record where it came from in **Source URL** and **Source note**, then select **Save**.

### Import a quarter's rate matrix
Keywords: upload IFTA rates, CSV rates, bulk tax rates
1. Open [IFTA tax rates](/fuel/configuration-files/ifta-tax-rates).
2. Select **New IFTA tax rate** and choose **Import rates**.
3. Pick the **Year** and **Quarter** every row is published for, then choose the CSV file. Use **Download template** if you are unsure of the layout.
4. Check the preview: rows marked **Ready** will be published and rows that need attention will be left out, with the **Problem** shown for each.
5. Confirm with the publish button to publish the rows that checked out.

### Correct or delete a rate
Keywords: fix IFTA rate, change fuel tax rate, remove rate
1. Open [IFTA tax rates](/fuel/configuration-files/ifta-tax-rates).
2. Select the rate's row, correct it and select **Save**.
3. To delete it, right-click the row, choose **Delete**, then **Delete rate**.

## Notes
This page is only available when your organization runs its own trucks (asset operations). Viewing it needs read access to IFTA tax rates; importing needs permission to create them and **Delete** needs permission to delete them.

Rates are global: every organization is taxed at the rate published here, so a change or deletion moves the figures on every return for that quarter, not only yours. A rate of zero is a published rate, not a missing one. After changing rates, select **Recompute** on the quarter's return in [IFTA returns](/fuel/ifta-returns); a deleted rate shows as missing when the return is recomputed.
