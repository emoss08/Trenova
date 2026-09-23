---
path: /admin/distance-profiles
aliases: [routing profiles, PC*Miler profiles, routing policy, mileage profiles]
related:
  - /admin/distance-controls
  - /admin/distance-overrides
  - /admin/stored-mileages
---

## What it's for
Distance profiles hold the routing policy Trenova sends to PC*Miler when it calculates distance:
the data version, region, routing type, distance units, how much of each stop's address is used,
and route behavior such as toll roads and cross-border routes. The table lists each profile with
its status, provider, routing type, units and data version, and marks the business unit's
default profile.

Administrators create profiles here, then assign them to mileage purposes (loaded moves, pay,
billing, fuel and so on) on [Distance controls](/admin/distance-controls).

## Tasks

### Create a distance profile
Keywords: new routing profile, add PC*Miler profile
1. Open [Distance profiles](/admin/distance-profiles).
2. Select **New distance profile**.
3. Under **Profile details**, fill in **Name**, **Status** and **Description**, and turn on
   **Default profile** if this should be the business unit's default.
4. Under **Provider policy**, set **Provider**, **Data version**, **Region**, **Routing type**,
   **Distance units** and **Location granularity**, and optionally a PC*Miler vehicle profile
   name.
5. Under **Route behavior**, choose **Highway only**, **Allow toll roads**, **Borders open** and
   **Include toll data** as needed.
6. Select **Save**.

### Edit a distance profile
Keywords: change routing type, update profile
1. Open [Distance profiles](/admin/distance-profiles).
2. Select the profile's row (or right-click it and choose **Edit**).
3. Change the fields you need and select **Save**.

### Make a profile the default
Keywords: default routing profile
1. Open [Distance profiles](/admin/distance-profiles).
2. Right-click an active profile and choose **Set default**.

### Delete a distance profile
Keywords: remove profile
1. Open [Distance profiles](/admin/distance-profiles).
2. Right-click the profile and choose **Delete**, then confirm with **Delete**.

## Notes
Viewing the page needs read access to distance profiles; **New distance profile** appears only
for people who can create them. Inactive profiles cannot be set as the default, and the default
profile cannot be deleted.
