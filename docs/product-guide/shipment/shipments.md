---
path: /shipment-management/shipments
aliases: [loads, freight, load board, shipment list, pro number, BOL, command center, trips]
related:
  - /shipment-management/orders
  - /shipment-management/service-failures
  - /shipment-management/shipments/import
  - /dispatch/console
  - /billing/queue
  - /intake
---

## What it's for
Shipments is the operations command center. The top rail shows shipment KPIs, the map shows where loads are, and the side panels (**Unassigned**, **Exceptions** and other watch lists) surface work that needs a person. The table below lists every shipment with its status, customer, stops and coverage; selecting a row expands it to show the route timeline, financials, documents and quick actions. Dispatchers, customer service and billing staff work loads from here.

Opening a shipment shows its full record in a side panel: **Details** (general information, service and classification, billing and rating, commodities and moves), plus service failures, **Documents**, **Comments** and **History**.

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
Keywords: search loads, look up pro number, filter shipments
1. Open [Shipments](/shipment-management/shipments).
2. Type in **Search shipments...**, or pick a view such as **All shipments**, **In transit**, **At risk**, **Unassigned** or **Delivering today**.
3. Narrow further with the **At risk**, **Reefer** and **Today** chips or with **Filter**, and switch between **Table** and **Timeline**.
4. Select a row to expand it, or choose **Edit** from its row menu to open the full shipment.

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
4. Or select **Cancel**, optionally enter a **Cancel reason**, and select **Cancel shipment**. A canceled shipment can be restored with **Uncancel**.

### Scan or print paperwork into a shipment
Keywords: scan POD, scan BOL, scan paperwork, print to shipment, cover sheet, Kofax
1. Open the shipment and go to its **Documents** tab.
2. Select **Scan**, choose the **Computer**, the **Scanner** and **Scan settings**, optionally a **Document type**, then **Start scan**. The pages are filed onto the shipment as they arrive; anything Trenova cannot place waits in [Intake](/intake).
3. To file something you print from another program instead, open the menu beside **Scan**, choose **Print into this record**, select **Wait for my print**, and print to the Trenova printer within ten minutes.
4. To scan paperwork for this shipment later or on another scanner, choose **Print cover sheets**, pick how many **Sheets**, select **Print**, and put a sheet on top of the paper before it is scanned.

### Send an EDI load tender
Keywords: 204, tender to partner, EDI tender
1. Open [Shipments](/shipment-management/shipments) and open the row menu of a shipment in New status whose customer has an EDI partner.
2. Select **Send EDI load tender** and confirm.

## Notes
Needs read access to shipments; **New shipment** appears only with create permission. **Send EDI load tender** needs create access to EDI. An invoiced shipment is locked and cannot be edited. Scanning needs Trenova Capture paired from [My scanners](/capture/devices) and scanning turned on for the organization.

To create a shipment from a rate confirmation document instead of typing it, use [Import from rate confirmation](/shipment-management/shipments/import).
