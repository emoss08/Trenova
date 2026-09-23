---
path: /billing/configuration-files/rate-matrices
aliases: [tariff grid, rate table, rate grid, LTL tariff, weight breaks, class rates, zone to zone rates, pricing grid]
related:
  - /billing/rate-agreements
  - /billing/configuration-files/rate-zones
  - /billing/configuration-files/formula-templates
---

## What it's for
Rate matrices holds published tariffs entered as the grid they were published as: rates laid out by origin and destination zone, weight break, freight class and similar axes, instead of one lane per cell. A lane on a rate agreement can point at a matrix, and the matrix's own rating method (a formula template) says what each number in the grid means, for example a per-mile rate or a flat charge.

Pricing staff maintain matrices here. The table shows each matrix's **Status**, **Code**, **Name**, **Rates are**, **Currency**, **Description** and **Created** date. A matrix opens with the tabs **Overview**, **Axes** and **Rates**.

## Tasks

### Add a rate matrix
Keywords: new tariff, create rate grid, enter tariff
1. Open [Rate matrices](/billing/configuration-files/rate-matrices).
2. Select **New rate matrix**.
3. On **Overview**, fill in **Status**, **Code**, **Name** and optionally **Description**. Under **Pricing**, pick the **Rating method** and set **Currency**, **Rounding mode** and **Rounding precision**.
4. On **Axes**, select **Add axis** for each dimension of the grid (up to four). For each, choose the **Dimension** (for example **Zone**, **Weight break**, **Freight class** or **Distance**), the **Match mode** (**Exact key** or **Band**), and give it a **Label**. A banded axis also takes an **Outside every band** policy.
5. Select **Save**. The **Rates** tab is available once the matrix has been saved.

### Change rates in the grid
Keywords: edit tariff rates, update cell, reprice matrix
1. Open [Rate matrices](/billing/configuration-files/rate-matrices) and select the matrix's row.
2. Open the **Rates** tab. The first axis runs down the rows and the second across the columns; with more axes, pick the sheet to edit from the selectors above the grid.
3. Type the new values into the cells, then select **Save rates**. Rates save on their own, separately from the rest of the form.

### Retire a rate matrix
Keywords: deactivate tariff, disable matrix
1. Open [Rate matrices](/billing/configuration-files/rate-matrices) and select the matrix's row.
2. On **Overview**, set **Status** to **Inactive** and select **Save**.

### Find a rate matrix
Keywords: search tariffs, filter matrices
1. Open [Rate matrices](/billing/configuration-files/rate-matrices).
2. Type a code or name in the search box, or use **Filter** and **Sort** in the table toolbar.

## Notes
Viewing the page needs read access to rate matrices; **New rate matrix** only appears for people who can create them.

Changing an axis after rates exist changes what every existing cell means, so re-enter the grid after any change on **Axes**. A matrix with no axes or no rates prices nothing, and an inactive matrix stops pricing; in each case every lane pointing at it stops with it. A lane is attached to a matrix on its rate agreement's **Lanes** tab in [Rate agreements](/billing/rate-agreements). A rate increase applied from Rate agreements does not change matrix rates.
