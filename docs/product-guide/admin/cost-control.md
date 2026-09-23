---
path: /admin/cost-control
aliases: [cost per mile, CPM, operating cost model, shipment profitability settings, margin target]
related:
  - /shipment-management/shipments
  - /admin/accounting-control
  - /fuel/configuration-files/surcharge
---

## What it's for
Cost control sets up the cost-per-mile model behind shipment profitability estimates. The page is
one settings form in four cards: **Cost basis** (fuel pricing, fleet miles per gallon, deadhead and
the target margin), **Variable costs** and **Fixed costs** (one row per cost category, each with an
industry benchmark rate) and **GL actuals** (deriving category rates from posted expenses).

Finance and operations managers use it to make the cost and margin figures on shipments reflect the
fleet's real costs.

## Tasks

### Set the cost basis
Keywords: fuel cost per mile, MPG, deadhead, target margin
1. Open [Cost control](/admin/cost-control).
2. In **Cost basis**, enter the **Fleet miles per gallon**.
3. Turn on **Use live fuel price** to price fuel from a diesel index instead of the benchmark, and
   choose the **Fuel index**.
4. Turn **Include Deadhead Miles** on or off, and enter a **Target margin percent**. Margins below
   it show as thin; it defaults to 10% when left empty.
5. Select **Save changes**.

### Override a cost category's rate
Keywords: custom rate, override benchmark, insurance cost, maintenance cost
1. Open [Cost control](/admin/cost-control).
2. In **Variable costs** or **Fixed costs**, find the category. Its benchmark rate is shown on the
   right.
3. Turn on **Override rate** and enter the **Override rate per mile**.
4. To leave a category out of the total, turn off **Active**.
5. Select **Save changes**.

### Derive rates from the general ledger
Keywords: GL actuals, actual costs, posted expenses
1. Open [Cost control](/admin/cost-control).
2. In **GL actuals**, turn on **Enable GL actuals** and set the **Rolling window (months)**.
   Optionally set **Planned monthly miles** as a floor for fixed categories.
3. For each category, choose its **Mapped GL accounts** and turn on **Use GL actuals**.
4. Select **Save changes**.

## Notes
Opening the page needs read access to costing control; saving needs update access. Each category
uses one rate source at a time: the benchmark, an override or GL actuals. Saving refreshes the cost
estimates shown on shipments.
