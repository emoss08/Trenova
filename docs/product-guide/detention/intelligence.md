---
path: /detention/intelligence
aliases: [detention analytics, detention report, dwell analysis, facility dwell, detention margin, waived detention, worst facilities, facility scorecard, lower detention, reduce detention, detention by hour, detention collection]
related:
  - /detention/desk
  - /detention/configuration-files/detention-policies
  - /billing/configuration-files/customers
---

## What it's for
Detention intelligence answers what an organization needs to know to lower detention: where trucks wait past free time, who caused the wait, when it happens, how much of the money is recovered, whether it is getting better, and what to do next. It is for operations leads, billing managers and controllers deciding which facilities to raise with shippers, which free-time terms to change, and where billing loses money.

The page opens with Nova's summary and five figures: **Hours past free time**, **Stops detained** (beside the industry rate, 39% of stops in ATRI's 2023 survey), **The dock's share**, **Recovered** and **Kept after driver pay**. Under them, **What to do next** ranks up to four moves by the hours or dollars each would recover. The sections below are **Where to focus**, **Why trucks wait**, **When they wait**, **Getting paid for it** and **Is it getting better**. Nothing on this page changes a charge; charges are worked on the [Detention desk](/detention/desk).

Every wait is put down to one cause, the first of: **Arrived late** (the truck missed its appointment, so free time was forfeited), **No appointment** (an open dock), **Arrived early** (the truck waited for its slot), or **On time, the dock was slow** (the facility's wait).

## Tasks

### Change the period being analysed
Keywords: last 30 days, 90 days, 180 days, date range
1. Open [Detention intelligence](/detention/intelligence).
2. Pick **30d**, **90d** or **180d** in the page header. The page starts on **90d**, and every figure is compared with the window of the same length before it.

### Find the facilities that cause most of the wait
Keywords: worst facilities, long dwell, shipper dwell, receiver detention, where to focus
1. Open [Detention intelligence](/detention/intelligence) and go to **Where to focus**.
2. Read **Hours past free time, by facility**: each column is a facility, split by who caused the wait. The strip under it shows how few facilities cause most of the hours.
3. Use **Rank by** to order the facilities by **Hours past free time**, **On-time waiting**, **Share of stops detained**, **Slowest one in ten**, **Billed** or **Net margin**. The figure over each column follows the ranking; the columns still show hours.
4. Select a facility to open its scorecard.

### Bring a facility's scorecard to a shipper review
Keywords: facility scorecard, quarterly review, export scorecard, shipper meeting
1. Open [Detention intelligence](/detention/intelligence) and select a facility in any chart, or **Open scorecard** on a next step.
2. Read its hours past free time, stops detained, median stop against its free time, who caused the wait, when it happens and what it cost.
3. Select **Export scorecard** to download it as a CSV file.

### See whether free time matches how long a dock really takes
Keywords: free time too short, median dwell, detention built into the schedule
1. Open [Detention intelligence](/detention/intelligence) and go to **Why trucks wait**.
2. Read **Free time against how long the dock takes**. A facility whose median stop runs past its free time is detained before anyone is late.
3. Select **Backtest a policy** on the next step, or open [Detention policies](/detention/configuration-files/detention-policies) to try more free time.

### Find the hours when trucks wait
Keywords: arrival hour, day of week, appointment times, detention by hour
1. Open [Detention intelligence](/detention/intelligence) and go to **When they wait**.
2. Read **Share of stops detained, by arrival**: weekday against the hour the truck arrived, in your organization's time zone. Pick a facility above the grid to see only its stops.
3. **Which stops wait** compares pickups with deliveries, appointments with open docks, and live loads with dropped trailers.

### Find where detention money is lost
Keywords: collected detention, late notice, forgiven detention, waivers, unprofitable customers, leakage
1. Open [Detention intelligence](/detention/intelligence) and go to **Getting paid for it**.
2. **From accrued to kept** follows the money from what the clock accrued, through what arrived late, missed its notice deadline, was forgiven, disputed or is not yet billed, to what was billed, invoiced and collected on paid invoices. Each leak floats where its money left and links to the screen that fixes it. Driver pay comes out of what was billed, and **Kept** is what remains: the same figure as **Kept after driver pay** at the top of the page.
3. **Customers whose terms cost us** lists customers whose detention costs more in driver pay than it is billed, with how often notice went out on time. **What was forgiven, and why** groups waivers by their coded reason.

### Mark a stop as a drop trailer
Keywords: drop and hook, live load, drop trailer, live unload
1. Open the shipment and edit its move.
2. On the stop, set **Live or drop** to **Drop trailer** when a trailer is left or taken, or **Live load** when the truck waits while the facility loads or unloads.
3. Detention intelligence then compares live stops with drops under **Which stops wait**.

## Notes
Opening the page needs read access to detention. Driver pay, net margin and the customer steps built on them are hidden from anyone without access to margins.

The figures cover stops whose detention clock started in the chosen window: whole days in your organization's time zone, ending today. They are kept up as stops settle and trail them by about a minute. **Collected** counts a billed charge only when its invoice is posted and paid in full. A facility's policy change is marked on **Hours past free time, weekly** where it took effect. If there is no detention in the window, the page says so and offers a wider window.
