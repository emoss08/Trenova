---
path: /dispatch/locations
aliases: [facilities, stops, shippers, consignees, pickup and delivery sites, addresses, geofences]
related:
  - /dispatch/configuration-files/location-categories
  - /shipment-management/shipments
  - /billing/configuration-files/customers
---

## What it's for
Locations are the places your shipments pick up at and deliver to: warehouses, shippers, consignees and other facilities. Each location has a name, a category, an address, an optional timezone and notes, and, when the Google Maps integration is set up, a geofence drawn on a map. Dispatchers, customer service and operations staff add and maintain them so stops can be picked from a list.

## Tasks

### Add a location
Keywords: new location, create facility, add shipper, add consignee
1. Open [Locations](/dispatch/locations).
2. Select **New location**.
3. Choose a **Status**, fill in **Name** and pick a **Location category**.
4. Start typing in **Address Line 1** and pick the address from the suggestions; this fills **City**, **State** and **Postal code**. Add **Address Line 2** if needed.
5. Optionally set a **Timezone** and add **Notes** for dispatchers and drivers.
6. Select **Save**, or pick **Save & close** or **Save & add another** from the arrow on the save button.

### Set a location's geofence
Keywords: geofence radius, draw boundary, arrival area
1. Open [Locations](/dispatch/locations) and select the location, or select **New location**.
2. Under **Geofence**, choose **Auto**, **Circle**, **Rectangle** or **Draw**.
3. Adjust the shape on the map beside the form, then select **Save**.

### Edit a location
Keywords: change address, update location
1. Open [Locations](/dispatch/locations).
2. Select the row, or right-click it and choose **Edit**.
3. Change the fields and select **Save**.

### Activate or deactivate several locations
Keywords: bulk status, inactivate locations
1. Open [Locations](/dispatch/locations).
2. Tick the rows to change.
3. Select **Update status** in the bar that appears and choose **Active** or **Inactive**.

## Notes
**Status**, **Name**, **Location category**, **City** and **Postal code** are required.

Address search and the geofence map need the Google Maps integration to be configured; without it the **Geofence** controls and map are not shown.

A location's **Timezone** matters for rating: rating formulas read pickup and delivery hours, weekdays and dates in that zone, and use UTC when none is set.

Adding needs create permission for locations; editing needs update permission.
