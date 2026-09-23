---
path: /detention/desk
aliases: [detention board, dock clocks, free time clocks, drivers waiting, dwell board, detention notices, live detention]
related:
  - /detention/configuration-files/detention-policies
  - /detention/intelligence
  - /shipment-management/shipments
---

## What it's for
The detention desk is the live board of every stop where a driver is currently on a dock. Each row shows the facility, customer and PRO number, the free-time clock, the customer notice deadline and the detention accruing. The strip across the top totals what is **Collectable now**, what is in the **Notice window**, what is **Uncollectable** and the **Longest wait**.

Operations and billing staff use it to get customer notices out before their deadlines, since detention that needs notice cannot be defended in a dispute once the deadline passes, and to approve, dispute or waive the resulting charges. The terms behind each clock come from [Detention policies](/detention/configuration-files/detention-policies).

## Tasks

### Work the stops that need attention
Keywords: filter detention, drivers past free time, notice due, sort by urgency
1. Open [Detention desk](/detention/desk).
2. Pick a lane above the board: **All**, **Notice** (notice window open), **Accruing** (past free time), **Free time** (still inside it) or **Lost** (past the notice deadline).
3. Change the order with the sort menu: **Urgency**, **Exposure**, **Time on site** or **Next deadline**.
4. Type in **Search facility, customer, PRO** to find a particular stop.

### Send customer detention notices
Keywords: notify customer, detention notice, notice deadline
1. Open [Detention desk](/detention/desk).
2. To notify one stop, select **Send** next to its notice countdown.
3. To notify every stop whose notice window is open, select the send-notices button in the page header, check the list of customers and the amount at stake, and confirm.
4. Each notice sent is recorded as evidence on the stop.

### Approve, dispute or waive a detention charge
Keywords: approve detention, waive detention, dispute detention, forgive charge
1. Open [Detention desk](/detention/desk) and select a stop to open its detail.
2. Review the **Billable**, **Driver pay** and **Net margin** figures, **How this charge was calculated**, the **Evidence chain** and the **Notices** sent.
3. Choose an action: **Approve charge** posts the detention charge to the shipment; **Send notice** notifies the customer; **Record dispute** logs what the customer is disputing; **Waive** forgives the charge after you pick a coded **Reason** and add a **Note**, then **Waive charge**.

### Build a dispute packet
Keywords: claim file, detention evidence, defend detention
1. Open [Detention desk](/detention/desk) and select the stop.
2. Choose the copy dispute packet action. The receipt, evidence and notices are copied to your clipboard to paste into a reply to the customer.

## Notes
Opening the desk needs read access to detention policies.

**Approve charge** only appears while a charge is pending, **Send notice** only while a required notice has not been sent, and **Record dispute** only when there is a billable amount that is not already disputed. The board refreshes on its own; the refresh button in the header pulls fresh numbers straight away.
