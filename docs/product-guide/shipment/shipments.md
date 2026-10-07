---
path: /shipment-management/shipments
aliases: [loads, freight, load board, shipment list, pro number, BOL, command center, trips, dispatch board, capacity, suggested actions]
related:
  - /shipment-management/orders
  - /shipment-management/service-failures
  - /shipment-management/shipments/import
  - /dispatch/console
  - /billing/queue
  - /intake
---

## What it's for
Shipments is the dispatcher's board. The table comes first, grouped by stage, from loads that need attention or coverage down to delivered ones. Above it is a one-sentence briefing of the day (when an AI provider is connected) and a capacity strip showing the drivers ready now or within two hours, or the carriers posting trucks, depending on whether the organization runs its own trucks, brokers freight, or both. A floating panel holds the **Brief**: one suggested action at a time, plus a watchlist of today's deliveries, uncovered pickups, accruing detention and freight ready to bill. Its **Activity** tab is a live feed of shipment events. Dispatchers, customer service and billing staff work loads from here.

Selecting a row expands it in place, showing the route, the money, the documents, the next step and quick actions. Opening a shipment shows its full record in a side panel: **Details** (general information, service and classification, billing and rating, commodities and moves), plus service failures, **Documents**, **Comments** and **History**.

## Tasks

### Create a shipment
Keywords: new load, enter a load, book a shipment, add shipment
1. Open [Shipments](/shipment-management/shipments) and select **New shipment**.
2. Under **General information**, enter the **BOL** and other reference details.
3. Under **Service & classification**, choose the **Service type** and **Shipment type**, and the **Tractor type** and **Trailer type** if needed.
4. Under **Billing & rating**, choose the **Customer**, check **Bill To**, and set the **Rating method** and **Base rate**.
5. Under **Move details**, select **Add first move** and add at least one pickup and one delivery stop.
6. Select **Save** (or **Save & close**).

### Find a shipment
Keywords: search loads, look up pro number, filter shipments, quick filters
1. Open [Shipments](/shipment-management/shipments).
2. Type in the search field. Focusing it while it is empty (or pressing /) lists the quick filters, such as late, uncovered, moving and delivering today, each with its count; a chosen quick filter shows as a chip under the toolbar, beside the other filters.
3. Narrow further with **Filter** (status, tender, billing, PRO number, customer, pickup and delivery dates, and revenue), order with **Sort**, and switch between **Table**, **Timeline** and **Map**. **Group** turns the stage groups on or off; a group's header collapses it. The **Lane** column stays pinned while the table scrolls sideways.
4. Select a row (or move to it with J and K and press Enter) to expand it, or choose **Edit** from its row menu to open the full shipment.

### Cover loads from the capacity strip
Keywords: available drivers, ready drivers, carrier capacity, posted trucks, best load, tender to carriers
1. Open [Shipments](/shipment-management/shipments). The strip under the briefing shows **Drivers** or **Carriers** (both tabs when the organization does both).
2. Select a driver or carrier to see the best loads for them, then select **Assign** or **Tender** on the one you want.
3. To clear the backlog at once, use the action under the summary, such as tendering the loads your drivers can't cover to their best-matched carriers.

### Work the suggested actions
Keywords: brief, action queue, exceptions, approve suggestion, notify customer of delay
1. Open [Shipments](/shipment-management/shipments) and open the panel with **Toggle side panel** (it stays open on later visits once opened), then its **Brief** (shown as **Overview** when no AI provider is connected).
2. Read the action at the top, then approve it with its button (Ctrl+Enter), open the shipment with **Review**, or push it to the back with **Later** (Alt+L).
3. If you approved something by mistake, select **Undo** on the line that confirms it.

### Assign a driver or carrier to a move
Keywords: dispatch load, cover a load, assign truck, assign driver, broker load
1. Open the shipment from [Shipments](/shipment-management/shipments) (**Edit**) and go to **Move details**. The shipment must be saved first.
2. Open the move's menu and select **Assign** (or **Reassign**).
3. Choose **Driver** to set the **Tractor**, **Trailer**, **Primary worker** and **Secondary worker**, or **Carrier** to broker the move to an outside carrier with its rate and reference details.
4. Save the assignment. To take it back, use **Unassign** or **Cancel carrier assignment** from the same menu.

### Send a shipment to billing
Keywords: ready to bill, bill the load, invoice shipment
1. Open [Shipments](/shipment-management/shipments) and open the shipment's row menu (or expand the row).
2. Select **Mark ready to bill**, **Transfer to billing**, or **Mark ready & transfer to billing**. Only the actions that fit the shipment's current state are shown.

### Duplicate, transfer or cancel a shipment
Keywords: copy load, repeat shipment, change owner, void shipment, cancel load
1. Open [Shipments](/shipment-management/shipments) and open the shipment's row menu.
2. Select **Duplicate**, set the **Number of copies** and whether to **Override dates**, then **Duplicate**. Copies are made in the background.
3. Or select **Transfer ownership**, choose the **New owner**, and confirm.
4. Or select **Cancel shipment**, optionally enter a **Cancel reason**, and confirm with **Cancel shipment**. A canceled shipment can be restored with **Uncancel**.

### Scan or print paperwork into a shipment
Keywords: scan POD, scan BOL, scan paperwork, print to shipment, cover sheet, Kofax
1. Open the shipment and go to its **Documents** tab.
2. Select **Scan**, choose the **Computer**, the **Scanner** and **Scan settings**, optionally a **Document type**, then **Start scan**. The pages are filed onto the shipment as they arrive; anything Trenova cannot place waits in [Intake](/intake).
3. To file something you print from another program instead, open the menu beside **Scan**, choose **Print into this record**, select **Wait for my print**, and print to the Trenova printer within ten minutes.
4. To scan paperwork for this shipment later or on another scanner, choose **Print cover sheets** from the menu beside **Scan**, pick how many **Cover sheets**, select **Print cover sheets**, and put a sheet on top of the paper before it is scanned.

### Send an EDI load tender
Keywords: 204, tender to partner, EDI tender
1. Open [Shipments](/shipment-management/shipments) and open the row menu of a shipment in New status whose customer has an EDI partner.
2. Select **Send EDI load tender** and confirm.

## Notes
Needs read access to shipments; **New shipment** appears only with create permission. **Send EDI load tender** needs create access to EDI. An invoiced shipment is locked and cannot be edited. Scanning needs Trenova Capture paired from [My scanners](/capture/devices) and scanning turned on for the organization.

To create a shipment from a rate confirmation document instead of typing it, use [Import from rate confirmation](/shipment-management/shipments/import).
