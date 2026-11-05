---
path: /dispatch/console
aliases: [dispatch board, dispatch queue, planning board, cover loads, driver assignment, load coverage, dispatch desk, tender board, nova proposals]
related:
  - /shipment-management/shipments
  - /dispatch/routing-guides
  - /dispatch/carriers
  - /hr/workers
  - /desk/watchtower
---

## What it's for
The dispatch console is where dispatchers cover open moves against available capacity without opening each shipment. A status line at the top left says whether the board is **Live**, **Reconnecting**, **Offline** or **Not updated**, and when it was last read. The strip under it shows **Needs a driver**, **Late**, **Due now**, **Proposals waiting**, **Exceptions** and **Open drivers**.

The **Queue** lists the moves that need a decision in three tabs: **Needs a driver**, **Exceptions** and **Drivers**. Moves are grouped by when their window is: **Late** (the window closed with no arrival), **Due now** (the window is open or opens within 4 hours), **Within 24 hours**, **Within 48 hours** and **Later**. An open window is never shown as late. Each row shows the next step: the driver Nova proposed, a tender in progress, a hold, or **No proposal yet**. The **Timeline** is the same sheet as the timeline on Shipments: the moves that need a driver as rows under the same groups, then a lane per driver showing the work they hold, their time off and their free time. Selecting a move or a driver opens the inspector on the right.

## Tasks

### Approve or decline Nova's proposal
Keywords: approve proposal, nova picked a driver, decline proposal, proposed driver
1. Open [Console](/dispatch/console) and select a move whose next step names a driver with the Nova badge.
2. Under **Nova proposes**, read the driver, the reason and the dry run line. **Dry run: would go through as shown** means the move has not changed since it was proposed.
3. Select **Approve** (A) to assign that driver. The same proposal leaves Decisions too.
4. Or select **Decline** (D), write why in the box, and select **Decline proposal**. Nova and whoever looks after it see the reason.
5. If an approval can't be carried out, the move stays in the queue with **Approved (driver); not assigned** and the reason, such as which stop's times conflict. Open it to see the same under the move's details, select **Open shipment** to fix what was refused, then assign a driver or select **Propose coverage** again. The note goes away once the move changes or a newer proposal replaces it.

### Assign a driver yourself
Keywords: cover a load, dispatch a driver, assign truck, rank drivers, pre-flight
1. Open [Console](/dispatch/console) and select a move, or walk the queue with J and K.
2. Under **Drivers for this move**, review the ranked drivers. Select **Why this score** to see how a driver was scored, or **Show ineligible** to see drivers who cannot take it and why.
3. Press the driver's number (1–9) or select **Assign**. The pre-flight opens under the driver.
4. When it has loaded, check the empty miles, drive time left, pickup margin and any findings, then press Enter or select **Assign driver** (**Reassign driver** on a covered move, which names who it replaces). Nothing is written until you confirm.

### Propose coverage for the window
Keywords: auto dispatch, optimize assignments, propose drivers, auto assign
1. Open [Console](/dispatch/console) and select **Propose coverage**.
2. Nova picks a driver for each move it can cover. Each proposal waits on its move in the queue and in Decisions; nothing is assigned until someone approves it.
3. With shadow mode on, a dialog shows what Nova would have proposed instead, and nothing is written.
4. To have proposals waiting without pressing the button, turn on **Keep coverage proposed** under **Enable automated assignment** in [Dispatch control](/admin/dispatch-controls). Nova then plans again when loads are created or lose their driver, and every hour.

### Plan on the timeline
Keywords: drag a load onto a driver, driver lanes, free time, gantt, plan relays
1. Open [Console](/dispatch/console) and select **Timeline**. Pick **Day**, **2 Days** or **3 Days** for the window.
2. The moves that need a driver come first, one row each, grouped by when their window is. Under **Drivers**, each lane shows the moves the driver holds, time off in its own band, and free time labeled (**Free from 2:00 PM**, **Free 4h**). A red dot on a driver means two of their moves overlap.
3. Hover a bar to see its stops and times. Select a bar or a row to open it in the inspector.
4. Drag a move's bar onto a driver (or focus it, press Space, move with the arrow keys and press Space again). The pre-flight for that pairing opens; nothing is assigned until you confirm it. A lane turns red under the bar when the driver is blocked or has no tractor, and the pre-flight says why.

### Work the exceptions
Keywords: late delivery, missed stop, no check call, tender exhausted, covered move slipping
1. Open [Console](/dispatch/console) and select the **Exceptions** tab, or the **Exceptions** figure.
2. The list comes from the Watchtower, soonest due first. Select a row to open the record it is about.

### Find work for a driver
Keywords: what can this driver take, next load for driver, open drivers
1. Open [Console](/dispatch/console) and select the **Drivers** tab, or the **Open drivers** figure.
2. Drivers are grouped by availability, open capacity first. Find one with **Search drivers** in the toolbar (by name, tractor code or fleet) and select them.
3. The inspector shows when they are free, their hours, what they hold, and the open moves under **Best fit in this window**. Select a move to open it and assign the driver from its pre-flight.

### Broker a move to a carrier
Keywords: tender load, send to carriers, spot tender, waterfall, cover with carrier
1. Open [Console](/dispatch/console) and select an uncovered move.
2. Select **Assign to carrier** to book one carrier directly, then **Assign to carrier** in the dialog.
3. Or select **Tender to carriers**. On **Waterfall**, keep **Use the matched guide** (or pick one under **Override guide**) and select **Start waterfall**. On **Spot**, choose each **Carrier** with its **Rate method**, **Rate**, **Offer expiry** and **Channel**, use **Add carrier** for more, and select **Send offers**.

### Move around and undo
Keywords: change date, show tomorrow, covered moves, undo assignment, search moves
1. Open [Console](/dispatch/console).
2. Use the arrows (← →) or **Today** (T) to move the window, and switch between **Queue** and **Timeline**.
3. Search by PRO, customer or city (/), narrow by **Fleet** or **Customer**, and turn on **Show covered** to include moves that already have coverage.
4. Select **Undo** (U) to take back the last assignment or approval. Undoing a reassign puts the previous driver back.

## Notes
Needs read access to shipment moves and workers. Assigning, approving with a pre-flight and **Propose coverage** need assign permission on shipment moves and the Asset operations capability. Approving and declining Nova's proposals needs update permission on agent proposals. **Assign to carrier** and **Tender to carriers** need the Brokerage capability; **Tender to carriers** also needs create permission on tenders.

A driver with no tractor cannot be assigned until one is set, and a pairing the organization's policy blocks cannot be assigned from the pre-flight. When the board cannot be read, the queue says so and offers **Try again** instead of showing an empty board.
