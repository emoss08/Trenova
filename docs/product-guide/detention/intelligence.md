---
path: /detention/intelligence
aliases: [detention analytics, detention report, dwell analysis, facility dwell, detention margin, waived detention, worst facilities]
related:
  - /detention/desk
  - /detention/configuration-files/detention-policies
  - /billing/configuration-files/customers
---

## What it's for
Detention intelligence looks back over settled detention and answers three questions: which facilities cost the most, which customers lose money on detention once driver pay is netted off, and where revenue is being forgiven through waivers. It is for billing managers, controllers and operations leads deciding which facilities to raise with customers and which contract terms to renegotiate.

The page opens with a ledger of what was **Billed**, paid out as **Driver pay**, **Forgiven** and **Kept**, and the **Net detention margin**. Below it are three panels: **Facility profiles**, **Customer margin** and **Waiver leakage**. Nothing on this page changes a charge; charges are worked on the [Detention desk](/detention/desk).

## Tasks

### Change the period being analysed
Keywords: last 30 days, 90 days, 180 days, date range
1. Open [Detention intelligence](/detention/intelligence).
2. Pick **30d**, **90d** or **180d** in the page header. The page starts on **90d**.
3. Select **Refresh** to recalculate the figures.

### Find the facilities that cost the most
Keywords: worst facilities, long dwell, shipper dwell, receiver detention
1. Open [Detention intelligence](/detention/intelligence) and go to **Facility profiles**.
2. Rank facilities by **Billed**, **Margin**, **Breach** or p90 (the long tail of dwell).
3. Select a facility to see its **Average dwell**, **Driver pay**, **Margin per stop** and **Leakage**.

### Find customers who lose money on detention
Keywords: unprofitable customers, negative detention margin
1. Open [Detention intelligence](/detention/intelligence) and go to **Customer margin**.
2. Rank customers by **Worst margin** or **Most billed**. Customers whose detention costs more in driver pay than it brings in are marked as a loss.

### See where waived detention is going
Keywords: forgiven detention, waivers, discretionary revenue, leakage
1. Open [Detention intelligence](/detention/intelligence) and go to **Waiver leakage**.
2. Read the forgiven amount grouped by the coded reason given when each charge was waived, and each reason's share of the total.

## Notes
Opening the page needs read access to detention policies.

The figures cover stops in the chosen window only. If there is no detention in the window, the page says so and offers a wider window. Waiver reasons come from the coded **Reason** picked when a charge is waived on the detention desk.
