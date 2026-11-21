---
path: /admin/distance-controls
aliases: [mileage settings, PC*Miler settings, mileage storage, IFTA miles setting, distance settings]
related:
  - /admin/distance-profiles
  - /admin/stored-mileages
  - /admin/distance-overrides
---

## What it's for
Distance controls decide how Trenova measures distance for your business unit. The page is one
settings form with five sections: **Routing** (the distance profile PC*Miler routes loaded moves,
empty moves and arrival estimates with), **Pay, billing and fuel** (the profile each kind of
mileage is measured with), **Stored mileage** (whether lane mileage already
calculated is reused and kept), **Jurisdiction miles** (whether each move's miles are broken
down by state or province for fuel tax returns) and **Rating** (the unit tariffs and rate tables
read distance in).

Administrators set this up once and revisit it when routing policy, billing mileage or fuel tax
reporting needs change.

## Tasks

### Reuse stored lane mileage
Keywords: cache mileage, save PC*Miler calls, stored miles
1. Open [Distance controls](/admin/distance-controls).
2. In **Stored mileage**, turn on **Reuse stored mileage** so a lane's stored mileage is looked up
   before PC*Miler is asked.
3. Turn on **Keep new mileage** to save each PC*Miler result as stored mileage for its lane.
4. While either switch is on, **Match by city** appears under them: turn it on so a stop with no
   postal code is matched by its city and state instead of being skipped.
5. Select **Save changes** in the bar that appears at the bottom.

### Capture state-by-state miles for IFTA
Keywords: IFTA, fuel tax, jurisdiction miles, state mileage
1. Open [Distance controls](/admin/distance-controls).
2. In **Jurisdiction miles**, turn on **Capture jurisdiction miles**.
3. Select **Save changes**.

### Choose which distance profile each workflow uses
Keywords: routing profile, practical vs shortest, billing miles, pay miles
1. Open [Distance controls](/admin/distance-controls).
2. Pick a profile for each purpose: **Loaded moves**, **Empty moves** and **Arrival estimates** in
   **Routing**; and **Pay**, **Billing** and **Fuel** in **Pay, billing and fuel**. Every one is required, and
   each must be an active PC*Miler profile.
3. Select **Save changes**. Use **Reset** instead to discard your edits.

### Rate in kilometers
Keywords: kilometers, metric, distance unit, tariff unit
1. Open [Distance controls](/admin/distance-controls).
2. In **Rating**, set **Distance unit** to **Kilometers**.
3. Select **Save changes**.

## Notes
Opening the page needs read access to distance control; saving needs update access. The profiles
offered come from [Distance profiles](/admin/distance-profiles), so create a profile there first
if the one you need is missing. A save that leaves a purpose without a profile is refused on that
row.

Capturing jurisdiction miles asks PC*Miler for a state-by-state report on every move route, which
PC*Miler may bill as an additional transaction per route. Routes calculated before it is turned
on have no jurisdiction breakdown until they are recalculated.

A move is routed with its **Loaded moves** or **Empty moves** profile. When **Pay**, **Billing**
or **Fuel** names a different profile, each save routes the move again with that profile in the
same PC*Miler request: driver pay and per-mile carrier rates use the pay miles, rating, tariffs,
fuel surcharges and invoices use the billing miles, and fuel tax returns use the fuel miles, with
the state-by-state breakdown taken from the fuel route. When a purpose uses the same profile as the
move, or the move's distance was typed in by hand, the move's own distance is used for it.

The unit stored mileage is kept in comes from each distance profile, not from **Distance unit**,
which only sets how tariffs and rate tables read distance.
