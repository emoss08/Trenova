---
path: /admin/cost-control
aliases: [cost per mile, CPM, operating cost model, shipment profitability settings, margin target]
related:
  - /shipment-management/shipments
  - /admin/accounting-control
  - /fuel/configuration-files/surcharge
---

## What it's for
Cost control sets up the cost-per-mile model behind shipment profitability estimates. It is one
settings form, saved once, in tabs: **Cost basis** (fuel pricing, fleet miles per gallon, deadhead
miles and the target margin), **Cost rates** (the variable and fixed cost categories, each on its
industry benchmark, your own rate or GL actuals) and **GL actuals** (reading category rates from
posted expenses; only when your plan includes accounting).

Finance and operations managers use it to make the cost and margin figures on shipments reflect the
fleet's real costs. When a save is refused, the tab holding the setting to fix opens and is marked.

## Tasks

### Set the cost basis
Keywords: fuel cost per mile, MPG, deadhead, target margin
1. Open [Cost control](/admin/cost-control).
2. On **Cost basis**, under **Fuel**, enter the **Fleet miles per gallon**.
3. Turn on **Use live fuel price** to price fuel from a diesel index instead of the benchmark, and
   choose the **Fuel index**.
4. Under **Profitability**, turn **Include deadhead miles** on or off, and enter a **Target
   margin**. Shipments below it show as thin; it is 10% when left empty.
5. Select **Save changes**.

### Set your own rate for a cost category
Keywords: custom rate, override benchmark, insurance cost, maintenance cost
1. Open [Cost control](/admin/cost-control) and select **Cost rates**.
2. Under **Variable costs** or **Fixed costs**, find the category. Its industry benchmark is shown
   under its name.
3. Set its **Rate** to **Your own rate** and enter **Your rate**. It starts from the benchmark.
4. To leave a category out of the total, turn its switch off.
5. Select **Save changes**.

### Derive rates from the general ledger
Keywords: GL actuals, actual costs, posted expenses
1. Open [Cost control](/admin/cost-control) and select **GL actuals**.
2. Turn on **Use GL actuals** and set the **Rolling window**. Optionally set **Planned monthly
   miles** as the fewest miles fixed costs are spread over.
3. Select **Open Cost rates**. For each category that should use the ledger, set its **Rate** to
   **GL actuals** and choose its **GL accounts**.
4. Select **Save changes**.

## Notes
Opening the page needs read access to costing control; saving needs update access. The target
margin is hidden from a role without access to margins. Each category uses one rate source at a
time: the benchmark, your own rate or GL actuals; a category on GL actuals uses its benchmark while
GL actuals are off. Saving refreshes the cost estimates shown on shipments.
