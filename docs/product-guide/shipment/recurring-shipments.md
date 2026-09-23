---
path: /shipment-management/recurring-shipments
aliases: [repeating loads, scheduled shipments, standing orders, dedicated lane, weekly run, shipment templates, auto-generate loads]
related:
  - /shipment-management/shipments
  - /shipment-management/orders
---

## What it's for
A recurring shipment is a series that copies a source shipment on a schedule, so a lane that repeats (a weekly run for the same customer, for example) is booked without re-entering it. Each series sets when pickups happen, in which timezone, how many days ahead the shipment is created, when the series starts and ends, which days are blocked, and whether shipments are created automatically or only on demand. Dispatch and customer service staff set these up for their repeating freight.

## Tasks

### Set up a recurring shipment
Keywords: create recurring load, schedule a lane, repeat shipment weekly
1. Open [Recurring shipments](/shipment-management/recurring-shipments).
2. Select **New recurring shipment**.
3. Under **Series**, enter a **Name**, choose the **Status**, pick the **Source shipment** to copy (search by Pro number or BOL), and add a **Description** if useful.
4. Under **Schedule**, set how often pickups happen (every day, week, month or a custom schedule) and at what time, the **Timezone**, and the **Lead time** in days before pickup that the shipment is created.
5. Under **Series window**, optionally set a **Start date**, an **End date** and **Max occurrences**.
6. Under **Blocked days**, turn on **Skip weekends** if needed, add **Blackout dates** (or **Add US federal holidays** from **Holidays**), and choose the **Exception policy** for a pickup that lands on a blocked day.
7. Under **Generation**, leave **Generate shipments automatically** on, or turn it off to keep the series as an on-demand template.
8. Select **Save**.

### Create the next shipment now
Keywords: generate now, run series, book the next occurrence
1. Open [Recurring shipments](/shipment-management/recurring-shipments).
2. Open the series' row menu and select **Generate now**. Expired series cannot generate.

### Pause or resume a series
Keywords: stop recurring shipment, hold series, restart series
1. Open [Recurring shipments](/shipment-management/recurring-shipments).
2. Open the series' row menu and select **Pause / resume**.

### See what a series has generated
Keywords: recurring shipment history, runs, generated loads
1. Open [Recurring shipments](/shipment-management/recurring-shipments).
2. Open the series' row menu and select **View history** to see its **Generation history**, including any occurrence the exception policy moved.

## Notes
Needs read access to recurring shipments; creating and editing need create and update permission.

The **Exception policy** either skips the occurrence (**Skip the occurrence**) or pulls the pickup earlier (**Move to previous business day**). Changing the schedule recalculates the next pickup; shipments already generated are never changed. A series expires once it reaches its end date or its occurrence cap.

When you enter a shipment on [Shipments](/shipment-management/shipments) that looks like a repeating lane, the shipment form may suggest **Set up recurring shipment**, or offer **Generate from series** when a series already covers the lane.
