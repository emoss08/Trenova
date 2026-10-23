---
path: /billing/configuration-files/rate-matrices
aliases: [tariff grid, rate table, rate grid, LTL tariff, weight breaks, class rates, zone to zone rates, pricing grid, zip ranges, postal ranges, publish matrix, matrix versions]
related:
  - /billing/rate-agreements
  - /billing/configuration-files/rate-zones
  - /billing/configuration-files/formula-templates
---

## What it's for
Rate matrices holds published tariffs entered as the grid they were published as: rates laid out by origin and destination zone, postal range, weight break, freight class and similar axes, instead of one lane per cell. A formula template reads a matrix by its code (`tables.ZONES[…]`), and the matrix's **Value type** says what each number in the grid is: money, money per a unit (a mile or kilometre, a pound, hundredweight, kilogram or ton, a foot, inch or metre, a cubic foot, cubic metre, gallon or litre, a minute, hour or day, a pallet, piece or stop), a percent, a plain number, or a whole number. Its money is always in the rating currency of the contract that reads it.

A matrix is versioned. Edits change its draft; a reviewer publishes the draft as the next numbered version, and published versions never change. Each formula template version pins the version of every matrix it reads when it is approved, so publishing a new matrix version does not reprice any contract until a new template version is approved.

Pricing staff maintain matrices here. The table shows each matrix's **Status** (**Draft**, **In review**, **Published** or **Archived**), **Published** version, **Code**, **Name**, **Values are**, **Description** and **Created** date. A matrix opens with the tabs **Overview**, **Axes**, **Rates** and **Versions**.

## Tasks

### Add a rate matrix
Keywords: new tariff, create rate grid, enter tariff
1. Open [Rate matrices](/billing/configuration-files/rate-matrices).
2. Select **New rate matrix**.
3. On **Overview**, fill in **Code**, **Name** and optionally **Description**. Under **Values**, pick the **Value type** the grid's numbers are.
4. On **Axes**, select **Add axis** for each dimension of the grid (one to four). For each, choose the **Dimension** (for example **Zone**, **Postal range**, **Weight break**, **Freight class** or **Distance**), the **Match mode** (**Exact key** or **Band**), and give it a **Label**. A banded axis also takes an **Outside every band** policy; a **Postal range** axis is always matched by range and has no such policy.
5. Select **Save**. The matrix starts as a draft that no tariff reads; the **Rates** and **Versions** tabs are available once it has been saved.

### Change rates in the grid
Keywords: edit tariff rates, update cell, reprice matrix
1. Open [Rate matrices](/billing/configuration-files/rate-matrices) and select the matrix's row.
2. Open the **Rates** tab. The first axis runs down the rows and the second across the columns; with more axes, pick the sheet to edit from the selectors above the grid.
3. Type the new values into the cells (each cell shows the value type's unit, such as `$/cwt`), then select **Save rates**. Rates save to the draft, separately from the rest of the form. A whole-number matrix refuses a fraction: the cell is marked and **Save rates** stays off until it is fixed.
4. Submit the draft for review (below). Tariffs keep reading the published version until a reviewer publishes the draft.

### Enter postal ranges
Keywords: zip ranges, postal code ranges, zip to zip, 3-digit zip ranges
1. Give the matrix an axis whose **Dimension** is **Postal range**, and save it.
2. On **Rates**, the axis's postal ranges are listed above the grid. Type the first and last postal code of a range (for example `100` through `149`) and select **Add range**. The range arrives as a row of blank cells across the other axes; fill each in before **Save rates** turns on.
3. To change a range, select **Edit** beside it, change its ends and select **Apply**; **Remove** deletes the range's cells on every sheet.
4. Keep both ends of a range the same number of characters, every range on one axis the same length, and ranges apart: both ends are included, and overlapping ranges are refused. A postal code matches the range its first characters fall between, compared character by character after trimming and upper-casing, so `10001` and `10001-1234` both match `100` through `149`; a postal code that no range holds finds no rate.

### Submit, publish or send back a draft
Keywords: review matrix, approve tariff, publish rates, request changes
1. Open the matrix and its **Versions** tab. It says which version tariffs read and whether the draft has changes that are not published.
2. To send the draft to review, select **Submit for review**, pick a reviewer and add a note.
3. A reviewer opens the same tab and selects **Publish** to make the draft the next version, **Request changes** to send it back with a comment, or **Reject**. Rejecting a matrix that was never published archives it.

### See or restore an earlier version
Keywords: version history, previous rates, roll back matrix
1. Open the matrix and its **Versions** tab. Each published version shows its cell count, who published it, when, and how many template versions pin it.
2. Select **View rates** to read a version's grid, which cannot be edited.
3. Select **Restore to draft** to replace the draft's axes and rates with that version's, then submit the draft to publish it again.
4. Check **Pinned by** to see every formula template version that pins a version of this matrix, with the version it pins.

### Retire a rate matrix
Keywords: deactivate tariff, disable matrix, archive matrix
1. Open [Rate matrices](/billing/configuration-files/rate-matrices) and select the matrix's row.
2. On **Versions**, select **Archive**. When template versions pin the matrix, confirm that you want to archive it anyway: they keep reading the version they pinned, but no new template version can read the matrix.
3. To bring an archived matrix back, select **Submit for review** and have it published.

### Find a rate matrix
Keywords: search tariffs, filter matrices
1. Open [Rate matrices](/billing/configuration-files/rate-matrices).
2. Type a code or name in the search box, or use **Filter** and **Sort** in the table toolbar.

## Notes
Viewing the page needs read access to rate matrices; **New rate matrix** only appears for people who can create them, **Submit for review** for people who can submit them, **Publish** for people who can publish them, **Request changes** and **Reject** for people who can reject them, and **Archive** for people who can archive them. Nobody publishes their own draft.

A matrix's **Code** is fixed once a version is published, because tariffs read it by that code. Changing an axis after rates exist changes what every existing cell means, so re-enter the grid after any change on **Axes**. A matrix with no published version, or an archived one, is not a table any new template version can read; a tariff that finds no cell refuses to rate with its own reason rather than billing nothing. Rounding is the tariff's, written in its source on [Formula templates](/billing/configuration-files/formula-templates). A rate increase applied from [Rate agreements](/billing/rate-agreements) does not change matrix rates.
