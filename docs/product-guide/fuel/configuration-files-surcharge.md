---
path: /fuel/configuration-files/surcharge
aliases: [fuel surcharge, FSC, fuel surcharge programs, DOE price, diesel price, fuel index, fuel table, peg price, fuel management]
related:
  - /billing/configuration-files/customers
  - /billing/rate-agreements
  - /billing/configuration-files/accessorial-charges
---

## What it's for
Fuel management is where fuel surcharges are set up and applied automatically. It has three tabs: **Price dashboard** shows the latest weekly diesel prices; **Surcharge programs** holds the programs that turn a week's fuel price into a surcharge (per mile, a percentage of the charge, or a flat amount); and **Fuel indices** lists the price series programs read from, including the DOE / EIA prices that are brought in automatically every week and any custom indices you maintain yourself.

Billing and pricing staff create a program here, then assign it to customers through the **Fuel surcharge program** on the customer's billing profile, or to a rate agreement on its **Fuel** tab.

## Tasks

### Create a fuel surcharge program
Keywords: new FSC program, set up fuel surcharge, peg and increment
1. Open [Fuel management](/fuel/configuration-files/surcharge) and go to **Surcharge programs**.
2. Select **New program**.
3. Fill in **Name**, **Code** and **Status**, choose the **Method**, the **Fuel index** it reads and the **Accessorial charge** the surcharge line posts against.
4. For a formula method, set the **Peg price**, **Increment** and **Rate per Increment**, or the **Miles per gallon** divisor. For a custom table method, build the **Price band table**: select **Generate** to fill it from a lowest and highest price and a band width, or **Add band** to enter bands yourself.
5. Adjust **Week resolution & rounding**, optional **Minimum amount** and **Maximum amount**, and the **Applicability** filters (**Shipment types**, **Service types**, **Tractor types**, **Trailer types**).
6. Select **Create program**.

### Change or retire a program
Keywords: edit fuel surcharge, deactivate program, delete program
1. Open [Fuel management](/fuel/configuration-files/surcharge) and go to **Surcharge programs**.
2. Select the program's card, make your changes and select **Save changes**. Setting **Status** to inactive stops it applying surcharges immediately.
3. To remove a program, hover its card, select the delete icon and confirm with **Delete**.

### Check this week's fuel price
Keywords: DOE diesel price, weekly price, current surcharge rate
1. Open [Fuel management](/fuel/configuration-files/surcharge).
2. The **Price dashboard** shows each index's latest weekly price and its history. On **Surcharge programs**, each card shows the rate that applies this week.

### Add a custom fuel index and its prices
Keywords: custom index, Canadian diesel, manual fuel price
1. Open [Fuel management](/fuel/configuration-files/surcharge) and go to **Fuel indices**.
2. Select **New custom index**, fill in **Name**, **Code**, **Fuel type**, **Region** and **Currency**, and select **Create index**.
3. Open the index's **Price history**, enter the **Week (Monday)** and **Price ($/gal)**, and select **Add**. Record one price per week.

## Notes
Opening the page needs read access to fuel surcharge programs.

A program assigned to customer billing profiles cannot be deleted until it is unassigned. DOE / EIA prices are ingested automatically and cannot be deleted; only manually entered prices can be removed. Each program picks the price week from the shipment date it is set to use, and Monday's DOE price applies from the program's **Price effective day**.
