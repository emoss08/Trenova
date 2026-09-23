---
path: /admin/distance-controls
aliases: [mileage settings, PC*Miler settings, mileage storage, IFTA miles setting, distance settings]
related:
  - /admin/distance-profiles
  - /admin/stored-mileages
  - /admin/distance-overrides
---

## What it's for
Distance controls decide how Trenova calculates and reuses mileage for your business unit. The
page is one settings form in three cards: **Stored mileage policy** (whether lane mileage is
reused before calling PC*Miler and how new mileage records are captured), **Jurisdiction
mileage** (whether each move's routed distance is broken down by state or province for IFTA
returns), and **Distance profile assignments** (which distance profile is used for each mileage
purpose).

Administrators set this up once and revisit it when routing policy, billing mileage or fuel tax
reporting needs change.

## Tasks

### Reuse stored lane mileage
Keywords: cache mileage, save PC*Miler calls, stored miles
1. Open [Distance controls](/admin/distance-controls).
2. In **Stored mileage policy**, turn on **Use stored mileage** so calculations check stored lane
   mileage before calling PC*Miler.
3. Turn on **Auto-create Stored Mileage** to buffer successful PC*Miler results for the scheduled
   job that saves them as stored mileage, and **Postal code fallback** to fall back to city and state lane keys when
   postal-code matching is unavailable.
4. Choose the **Stored distance units**.
5. Select **Save changes** in the bar that appears at the bottom.

### Capture state-by-state miles for IFTA
Keywords: IFTA, fuel tax, jurisdiction miles, state mileage
1. Open [Distance controls](/admin/distance-controls).
2. In **Jurisdiction mileage**, turn on **Capture jurisdiction miles**.
3. Select **Save changes**.

### Choose which distance profile each workflow uses
Keywords: routing profile, practical vs shortest, billing miles, pay miles
1. Open [Distance controls](/admin/distance-controls).
2. In **Distance profile assignments**, pick a profile for each purpose: **Loaded move**,
   **Empty move**, **Pay**, **Billing**, **Fuel**, **ETA out-of-route**, **Calculator practical**
   and **Calculator shortest**. Every one is required.
3. Select **Save changes**. Use **Reset** instead to discard your edits.

## Notes
Opening the page needs read access to distance control. The profiles offered in the assignments
come from [Distance profiles](/admin/distance-profiles), so create a profile there first if the
one you need is missing.

Capturing jurisdiction miles asks PC*Miler for a state-by-state report on every move route, which
PC*Miler may bill as an additional transaction per route. Routes calculated before it is turned
on have no jurisdiction breakdown until they are recalculated.
