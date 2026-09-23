---
path: /dispatch/console
aliases: [dispatch board, planning board, cover loads, driver assignment, load coverage, dispatch desk, tender board]
related:
  - /shipment-management/shipments
  - /dispatch/routing-guides
  - /dispatch/carriers
  - /hr/workers
---

## What it's for
The dispatch console is where dispatchers cover open moves against available capacity without opening each shipment. The strip at the top shows **Needs coverage**, **Late**, **At risk**, **Open drivers**, **Utilization** and **Assigned today**. The center shows open moves either as a **Board** grouped by urgency (Late, Next 4 hours, Today, Tomorrow, Planned) or as a **Timeline**; the **Capacity** rail lists drivers by availability (Open, Finishing, Working, Blocked, Time off); and the inspector on the right ranks drivers for the selected move or finds work for the selected driver.

## Tasks

### Assign a driver to an open move
Keywords: cover a load, dispatch a driver, assign truck, rank drivers
1. Open [Console](/dispatch/console) and select a move on the board or timeline.
2. In the inspector, review the ranked drivers. Select **Why this score** to see how a driver was scored, or **Show ineligible** to see drivers who cannot take it and why.
3. Select **Assign** on the driver you want.
4. Check the preflight summary (empty miles, drive time left, any findings) and select **Assign driver** (or **Reassign driver** on a covered move).

### Find work for a driver
Keywords: what can this driver take, next load for driver, open drivers
1. Open [Console](/dispatch/console).
2. In the **Capacity** rail, search with **Search driver, tractor, fleet** or filter by availability, and select a driver.
3. The inspector lists the driver's best-fit open moves. Select one and assign it.

### Let the console propose assignments
Keywords: auto dispatch, optimize assignments, bulk assign
1. Open [Console](/dispatch/console) and select **Auto-assign**.
2. Review the **Auto-assign proposal**. Nothing is written until you apply it.
3. Untick any pairings you do not want and select the assign button, or **Discard plan**.

### Broker a move to a carrier
Keywords: tender load, send to carriers, spot tender, waterfall, cover with carrier
1. Open [Console](/dispatch/console) and select an uncovered move.
2. Select **Assign to carrier** to book one carrier directly, then **Assign to carrier** in the dialog.
3. Or select **Tender to carriers**. On **Waterfall**, keep **Use the matched guide** (or pick one under **Override guide**) and select **Start waterfall**. On **Spot**, choose each **Carrier** with its **Rate method**, **Rate**, **Offer expiry** and **Channel**, use **Add carrier** for more, and select **Send offers**.

### Move around the plan
Keywords: change date, show tomorrow, covered moves, undo assignment
1. Open [Console](/dispatch/console).
2. Use the arrows or **Today** (T) to move the time window, and switch between **Board** and **Timeline**.
3. Turn on **Show covered** to include moves that already have coverage.
4. Select **Undo** (u) to reverse the last assignment.

## Notes
Needs read access to shipment moves. Ranking drivers and **Auto-assign** need the Asset operations capability. **Assign to carrier** and **Tender to carriers** need the Brokerage capability; **Assign to carrier** also needs assign permission on shipment moves, and **Tender to carriers** needs create permission on tenders.

A driver with no tractor cannot be assigned until one is set, and an assignment the organization's policy blocks shows **Blocked by policy**.
