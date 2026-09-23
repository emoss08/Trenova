---
path: /admin/distance-overrides
aliases: [mileage override, fixed lane miles, manual distance, lane distance override]
related:
  - /admin/distance-profiles
  - /admin/stored-mileages
  - /admin/distance-controls
---

## What it's for
Distance overrides replace the calculated distance between two locations with a figure you set,
for routing and billing adjustments. An override names an origin and destination location, the
distance, optionally a customer it applies to, and optionally intermediate stops in travel order.
The table lists each override with its origin, destination, distance, customer and stops.

## Tasks

### Add a distance override
Keywords: new override, set lane miles, fix distance
1. Open [Distance overrides](/admin/distance-overrides).
2. Select **New distance override**.
3. Choose the **Origin location** and **Destination location** and enter the **Distance**.
4. Optionally choose a **Customer** to limit the override to that customer.
5. Under **Intermediate stops**, select **Add stop** for each stop between origin and
   destination, in travel order.
6. Select **Save**.

### Change or delete an override
Keywords: edit override, remove override
1. Open [Distance overrides](/admin/distance-overrides).
2. Select a row to edit it, change the fields and select **Save**.
3. To delete one, right-click the row, choose **Delete** and confirm with **Delete**. This cannot
   be undone.

## Notes
Viewing the page needs read access to distance overrides; **New distance override** appears only
for people who can create them.
