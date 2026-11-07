---
path: /admin/dispatch-controls
aliases: [dispatch settings, auto assignment, auto dispatch, DOT compliance checks, service failure settings, HOS enforcement, anticipatory dispatch, demand forecast]
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
compliance enforcement** (checks that run before an assignment goes ahead). **Freight forecast**
shows whether the organization's freight forecast, learned from its own shipment history, has
proven itself well enough for plans to use.

Dispatch managers and safety or compliance leads use it.

## Tasks

### Turn on automated assignment
Keywords: auto assign, auto dispatch, optimizer, deadhead limit
1. Open [Dispatch controls](/admin/dispatch-controls).
2. In **Automated resource assignment**, turn on **Enable automated assignment**.
3. Optionally turn on **Learn our dispatch style** (the planner learns how much your dispatchers
   weigh each preference from the drivers they assign against the plan) and **Plan for where
   drivers end up** (a move that leaves its driver where drivers have been worth more at that time
   of the week is credited) and **Anticipate freight** (see below). All three are off by default.
4. Set the **Auto-execute confidence threshold** (0 to 1), **Maximum deadhead miles**, **Planning
   horizon (hours)** and **Coverage risk window (hours)**.
5. Select **Save changes**.

### Set what the planner weighs
Keywords: costs, prices, empty mile cost, candidate ranking, scoring
1. Open [Dispatch controls](/admin/dispatch-controls).
2. In **Costs and prices**, set what an **Empty mile**, a **Late minute**, a **Driver hour** and a
   **Day past home time** cost your operation, and what each of the **Preferences** is worth. The planner
   prices every driver for every move and picks the plan that costs least overall; a cost of $0 is
   ignored. Home time needs the driver's home address and **Last home** on their worker record.
   **Changing a planned driver** is what moving a move off the driver the last plan chose costs,
   so proposals don't change for a few dollars.
3. Under **Leaving a move uncovered**, set the **Share of revenue (%)**, the **At least** floor and
   the **Late premium** and **Due now premium**. When drivers are short, the moves that cost most to
   leave are covered first.
4. Select **Save changes**.

### Anticipate freight before it is booked
Keywords: anticipatory dispatch, position drivers ahead of freight, forecast freight, likely loads
1. Open [Dispatch controls](/admin/dispatch-controls).
2. In **Automated resource assignment**, turn on **Anticipate freight**. Plans then credit a move
   that leaves its driver near freight your history says is likely to appear in the twelve hours
   after the driver is free, beyond what other drivers expected there can take, less what the
   driver was worth staying where they are. A move that takes a driver out of an area about to
   get busy costs the plan instead. The first extra driver in a busy area is worth much more than
   the fifth, and two drivers are not both credited for the same likely load.
3. In **Costs and prices**, set **Most a likely load is worth**: the most ending near likely
   freight, or staying near it, can be worth on one move. Leave it empty (it shows
   **Automatic**) to use what a typical load is worth to you, from its revenue and your share
   of revenue.
4. Select **Save changes**.
5. Check **Freight forecast**: plans use the forecast only while it says **Ready**. Until then the
   card says **Anticipate freight is on, but plans ignore the forecast until it is Ready.** and
   plans run as if the setting were off.
6. On a planned move, the **Where they end up** line of the driver's cost in the
   [Console](/dispatch/console) says which likely loads the credit is for and, when staying put
   was worth something, how much the move adds or costs against it.

### Check the freight forecast
Keywords: forecast quality, is the forecast ready, forecast accuracy, not enough history
1. Open [Dispatch controls](/admin/dispatch-controls) and find **Freight forecast**.
2. The forecast is learned from your shipment history and tested every night on the weeks it has
   not seen. Its status is one of:
   - **Ready**: how many actual counts fell inside the forecast range, and how much more accurate
     it was than repeating last week. Plans use it when **Anticipate freight** is on.
   - **Not better than last week yet**: it was tested but did not beat repeating last week, so
     plans ignore it until it does better.
   - **Not enough history yet**: it needs 8 weeks and 200 loads of history before it can be
     tested; the card says how much it has.
   - **Not fitted yet**: it has not been fitted from your history yet.
3. **History**, **Average miss** and **Last fitted** give the figures behind the status.
4. If the card says **The freight forecast could not be loaded.**, select **Try again**.

### See and reset the learned dispatch style
Keywords: learned weights, dispatcher preferences, reset learning
1. Open [Dispatch controls](/admin/dispatch-controls).
2. Under **Learned dispatch style**, read how much your dispatchers weigh each preference against
   your costs, for example "1.6× your cost, learned from 312 decisions".
3. To start over, select **Reset learned style** and confirm. The planner goes back to your own
   costs; decisions already made are not learned again.

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
assignment, costs and prices, learned dispatch style, freight forecast and DOT compliance cards
only appear for organizations with asset operations turned on; a brokerage-only organization sees just **Service failure monitoring**.
Turning off **Enable DOT compliance enforcement** turns off the four individual checks.
