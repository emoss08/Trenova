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
Shipment control holds the organization-wide rules for shipments. The page is one settings form
in five cards: **Performance metrics configuration** (tracking customer rejections), **Shipment
processing configuration** (weight limit, duplicate bill of lading check, move removals and
hazmat segregation checks), **Shipment delay management** (automatically marking shipments
delayed), **Auto cancel shipments** (cancelling shipments left in New status) and **Detention
management** (how detention time is tracked and charged).

Administrators and operations managers use it to set how strict shipment entry is and what the
system does automatically.

## Tasks

### Set shipment entry checks
Keywords: max weight, duplicate BOL, hazmat check, remove moves
1. Open [Shipment controls](/admin/shipment-controls).
2. In **Shipment processing configuration**, set the **Max shipment weight limit** in pounds.
3. Turn on **Check for duplicate bills of lading** to require a unique BOL number on each new
   shipment, and **Check Hazmat segregation** to verify hazmat shipments are properly segregated.
4. Turn on **Allow move removals** only if users may permanently remove moves from shipments
   instead of cancelling them.
5. Select **Save changes** in the bar that appears at the bottom.

### Mark late shipments as delayed automatically
Keywords: delay status, late shipments, auto delay
1. Open [Shipment controls](/admin/shipment-controls).
2. In **Shipment delay management**, turn on **Automatic delay status updates**.
3. Enter the **Delay status threshold** in minutes.
4. Select **Save changes**.

### Cancel stale new shipments automatically
Keywords: auto cancel, void old shipments
1. Open [Shipment controls](/admin/shipment-controls).
2. In **Auto cancel shipments**, turn on **Automatic cancel shipments**.
3. Enter the **Auto cancel shipments threshold** in days a shipment may stay in New status.
4. Select **Save changes**.

### Set up detention tracking and charges
Keywords: detention, free time, detention billing, detention policy
1. Open [Shipment controls](/admin/shipment-controls).
2. In **Detention management**, turn on **Use detention policy engine** to compute detention with
   detention policies, and optionally pick a **Default detention policy**.
3. Or, for the flat threshold, turn on **Track detention time**, then set **Detention charge**
   and **Detention threshold** (free time in minutes), and turn on **Auto generate detention
   charges** to add detention lines to invoices automatically.
4. Select **Save changes**.

## Notes
Opening the page needs read access to shipment control. The detention policy engine needs at
least one active detention policy or a default policy; set those up on
[Detention policies](/detention/configuration-files/detention-policies). **Track customer
rejections** in **Performance metrics configuration** records when customers refuse shipments.
