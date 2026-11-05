---
path: /admin/dispatch-controls
aliases: [dispatch settings, auto assignment, auto dispatch, DOT compliance checks, service failure settings, HOS enforcement]
related:
  - /admin/service-failure-reason-codes
  - /admin/shipment-controls
  - /hr/workers
---

## What it's for
Dispatch control holds the organization-wide rules for assigning and dispatching work. The page is
one settings form in up to four cards: **Automated resource assignment** (letting the system assign
drivers and equipment), **Costs and prices** (what the planner weighs, in money), **Service failure
monitoring** (which late pickups and deliveries are recorded as service failures) and **DOT
compliance enforcement** (checks that run before an assignment goes ahead).

Dispatch managers and safety or compliance leads use it.

## Tasks

### Turn on automated assignment
Keywords: auto assign, auto dispatch, optimizer, deadhead limit
1. Open [Dispatch controls](/admin/dispatch-controls).
2. In **Automated resource assignment**, turn on **Enable automated assignment**.
3. Optionally turn on **Learn our dispatch style** (the planner learns how much your dispatchers
   weigh each preference from what they approve, decline and choose instead) and **Plan for where
   drivers end up** (a move that leaves its driver where drivers have been worth more is credited).
4. Set the **Auto-execute confidence threshold** (0 to 1), **Maximum deadhead miles**, **Planning
   horizon (hours)** and **Coverage risk window (hours)**.
5. Select **Save changes**.

### Set what the planner weighs
Keywords: costs, prices, empty mile cost, candidate ranking, scoring
1. Open [Dispatch controls](/admin/dispatch-controls).
2. In **Costs and prices**, set what an **Empty mile**, a **Late minute**, a **Driver hour** and a
   **Day past home time** cost your operation, and what each **Preference** is worth. The planner
   prices every driver for every move and picks the plan that costs least overall; a cost of $0 is
   ignored.
3. Under **Leaving a move uncovered**, set the **Share of revenue (%)**, the **At least** floor and
   the **Late premium** and **Due now premium**. When drivers are short, the moves that cost most to
   leave are covered first.
4. Select **Save changes**.

### Record service failures
Keywords: late pickup, late delivery, on-time tracking, service failure target
1. Open [Dispatch controls](/admin/dispatch-controls).
2. In **Service failure monitoring**, choose which incidents to capture in **Record Service
   failures**, for example **Pickup**, **Delivery** or **Pickup/delivery**. **Never** turns it off.
3. Set the **Service failure grace period** in minutes and, optionally, a **Service failure target**
   percentage.
4. Select **Save changes**.

### Enforce DOT compliance at dispatch
Keywords: medical card, driver qualification, hazmat endorsement, drug testing, block dispatch
1. Open [Dispatch controls](/admin/dispatch-controls).
2. In **DOT compliance enforcement**, turn on **Enable DOT compliance enforcement**.
3. Turn on the checks to run: **Medical certification validation**, **Driver qualification
   verification**, **Hazardous materials compliance** and **Drug and alcohol testing compliance**.
4. Set the **Compliance enforcement level** to **Warning**, **Block** or **Audit**.
5. Turn on any assignment rules needed: **Require worker assignment**, **Require trailer
   continuity**, **Enforce worker PTA restrictions** and **Enforce worker tractor fleet
   continuity**.
6. Select **Save changes**.

## Notes
Opening the page needs read access to dispatch control; saving needs update access. The automated
assignment, costs and prices and DOT compliance cards only appear for organizations with asset
operations turned on; a brokerage-only organization sees just **Service failure monitoring**.
Turning off **Enable DOT compliance enforcement** turns off the four individual checks.
