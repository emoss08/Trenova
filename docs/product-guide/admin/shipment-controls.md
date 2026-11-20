---
path: /admin/shipment-controls
aliases: [shipment settings, shipment rules, auto cancel, delay settings, detention settings, duplicate BOL check]
related:
  - /admin/hazmat-segregation-rules
  - /admin/hold-reasons
  - /admin/service-failure-reason-codes
  - /admin/sequence-configs
---

## What it's for
Shipment control holds the organization-wide rules for shipments. The page is one settings form,
saved once, in three tabs: **Shipment entry** (the weight limit, the duplicate bill of lading and
hazmat segregation checks, and removing moves), **Status
automation** (marking late shipments delayed and cancelling shipments left in New) and
**Detention** (how detention is worked out and charged). The Detention tab appears only when the
organization's plan includes detention.

Administrators and operations managers use it to set how strict shipment entry is and what the
system does automatically.

## Tasks

### Set shipment entry checks
Keywords: max weight, weight limit, duplicate BOL, hazmat check, remove moves
1. Open [Shipment controls](/admin/shipment-controls). The **Shipment entry** tab opens first.
2. In **Entry checks**, set the **Weight limit** in pounds. A shipment's mode profile can set its
   own limit instead.
3. Turn on **Check for duplicate BOLs** to refuse a bill of lading number already on another
   shipment, and **Check hazmat segregation** to refuse hazardous materials that must not travel
   together.
4. In **Shipment history**, turn on **Allow removing moves** only if users may delete moves from
   shipments instead of cancelling them.
5. Select **Save changes** in the bar that appears at the bottom.

### Mark late shipments as delayed automatically
Keywords: delay status, late shipments, auto delay
1. Open [Shipment controls](/admin/shipment-controls) and select the **Status automation** tab.
2. In **Delays**, turn on **Mark late shipments delayed**.
3. Enter **Late after**, the minutes past its schedule before a shipment is marked delayed.
4. Select **Save changes**.

### Cancel stale new shipments automatically
Keywords: auto cancel, void old shipments
1. Open [Shipment controls](/admin/shipment-controls) and select the **Status automation** tab.
2. In **Cancellations**, turn on **Cancel stale new shipments**.
3. Enter **Cancel after**, the days a shipment may stay in New after it was created.
4. Select **Save changes**.

### Set up detention tracking and charges
Keywords: detention, free time, detention billing, detention policy
1. Open [Shipment controls](/admin/shipment-controls) and select the **Detention** tab.
2. To work detention out from detention policies, turn on **Use detention policies** and pick
   the **Default policy**: an active policy that covers all freight, with no customer, facility,
   type or appointment scope. It governs every stop no other active policy covers. The flat
   detention settings are hidden while policies are on, and the flat clock is turned off when
   you save.
3. Or, with policies off, turn on **Track detention time** in **Flat detention**, then set
   **Free time** in minutes and the **Detention charge**, and turn on **Add detention charges
   automatically** to add the charge to shipments that run past their free time.
4. Select **Save changes**.

## Notes
Opening the page needs read access to shipment control. Detention policies are set up on
[Detention policies](/detention/configuration-files/detention-policies). A save that is refused
opens the tab holding the setting to fix, and marks it.
