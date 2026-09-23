---
path: /admin/stored-mileages
aliases: [mileage cache, saved lane miles, stored lanes, PC*Miler history]
related:
  - /admin/distance-controls
  - /admin/distance-overrides
  - /admin/distance-profiles
---

## What it's for
Stored mileages are reusable mileage records captured from PC*Miler calculations. When stored
mileage is turned on in [Distance controls](/admin/distance-controls), calculations check these
records before calling PC*Miler. The table shows each record's lane (with any intermediate
stops), distance, routing type, method, status, how many times it has been used (**Hits**) and
when it was calculated.

The page is for review: records are created by the mileage calculations, not typed in here.

## Tasks

### Review stored lane mileage
Keywords: find lane miles, search stored mileage, most used lanes
1. Open [Stored mileages](/admin/stored-mileages).
2. Search, or use **Filter** to narrow by route, distance, routing, method or status.
3. Sort by **Hits** to see the lanes reused most, or by **Calculated** to see the newest records.

### Stop a stored mileage from being reused
Keywords: deactivate mileage, remove stored lane, bad mileage
1. Open [Stored mileages](/admin/stored-mileages).
2. Right-click an active record and choose **Deactivate**.
3. Confirm with **Deactivate**. The record is kept for audit and history but is no longer used in
   future mileage lookups.

## Notes
Viewing the page needs read access to stored mileage. To fix the distance for a lane rather than
stop reusing it, add a [distance override](/admin/distance-overrides).
