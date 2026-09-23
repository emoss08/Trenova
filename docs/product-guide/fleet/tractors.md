---
path: /equipment/tractors
aliases: [trucks, power units, units, tractor list]
related:
  - /equipment/trailers
  - /equipment/configuration-files/equipment-types
  - /equipment/configuration-files/equipment-manufacturers
  - /dispatch/configuration-files/fleet-codes
  - /hr/workers
---

## What it's for
Tractors lists every power unit in your fleet with its code, status, primary worker, equipment
type, manufacturer and fleet code. Fleet and operations staff use it to add tractors, keep their
registration, fuel and telematics details current, assign the drivers who run them, and mark units
available, out of service, at maintenance or sold.

Opening a tractor also shows the documents stored against it and its driver vehicle inspection
reports (DVIR) and defect history.

## Tasks

### Add a tractor
Keywords: new truck, create tractor, add power unit
1. Open [Tractors](/equipment/tractors).
2. Select **New tractor**.
3. Fill in **Status**, **Code**, **Equipment type**, **Equip. manufacturer** and **Primary worker**.
   These are required.
4. Optionally add **Model**, **Make**, **Year** and **Fleet code**, the
   **Registration information** (**VIN**, **Registration number**, **License state**,
   **Registration expiry**, **License plate number**), a **Secondary worker**, the **Fuel type**
   and **IFTA qualified** setting, and the **Samsara vehicle ID** under **Telematics**.
5. Select **Save**, or choose **Save & close** or **Save & add another** from the save button's menu.

### Edit a tractor
Keywords: update truck, change driver on tractor, reassign tractor
1. Open [Tractors](/equipment/tractors).
2. Find the tractor with the search box or **Filter**, then select its row.
3. Change the fields on the **Details** tab.
4. Select **Save** or **Save & close**.

### Change a tractor's status
Keywords: out of service, at maintenance, mark sold, available
1. Open [Tractors](/equipment/tractors).
2. Select the status badge in the tractor's **Status** column and pick the new status.
3. To change several at once, tick the rows, then choose **Update status** in the bar at the
   bottom of the table and pick the status.

### Review a tractor's inspections and documents
Keywords: DVIR, defects, vehicle inspection, registration papers
1. Open [Tractors](/equipment/tractors) and select the tractor's row.
2. Select the **Inspections** tab to see its inspection reports and defects, each marked
   **Open** or **Resolved**.
3. Select the **Documents** tab to view the files stored for the tractor, or **Upload** new ones.

## Notes
- The page needs read access to tractors and the asset operations feature. Adding a tractor needs
  create access; editing and status changes need update access.
- **Inspections** come from a connected telematics provider. If none is connected the tab shows
  **Telematics not connected** with a link to **Open integrations**.
- **IFTA qualified** decides whether the unit's miles and fuel count on the quarterly IFTA return.
