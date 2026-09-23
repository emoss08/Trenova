---
path: /admin/hazmat-segregation-rules
aliases: [hazmat seg. rules, segregation table, incompatible hazmat, dangerous goods compatibility, 49 CFR 177.848, load compatibility]
related:
  - /shipment-management/configuration-files/hazardous-materials
  - /admin/shipment-controls
---

## What it's for
Hazmat segregation rules say which hazardous material classes may not travel together, or may
only travel together with separation. Each rule pairs a **Class A** with a **Class B** (and can
narrow the pair to a specific **Hazardous material A** and **Hazardous material B**) and sets the
**Segregation type**: **Prohibited**, **Separated**, **Distance** (with a minimum distance) or
**Barrier**. Rules can record exceptions and the regulation they come from, such as a 49 CFR
reference.

The table lists **Status**, **Name**, **Description**, **Class A**, **Class B**, **Segregation
type**, **Min distance**, **Has exceptions** and **Updated At**. Safety and compliance staff
maintain it; the rules are used when shipments are checked for hazmat segregation.

## Tasks

### Add a segregation rule
Keywords: new segregation rule, incompatible classes, hazmat pairing
1. Open [Hazmat segregation rules](/admin/hazmat-segregation-rules).
2. Select **New hazmat segregation rule**.
3. Set **Status**, and enter a **Name** and optional **Description**.
4. Choose **Class A** and **Class B**. To apply the rule to specific materials only, also pick
   **Hazardous material A** and **Hazardous material B**.
5. Choose the **Segregation type**. For **Distance**, also enter the **Minimum distance** and
   choose the **Distance unit** (**Feet**, **Meters**, **Inches** or **Centimeters**).
6. If the rule has exceptions, turn on **Has exceptions** and describe them in **Exception
   notes** (required once the switch is on).
7. Optionally record the **Reference code** (for example 49 CFR 177.848) and **Regulation
   source**, then select **Save**.

### Change or deactivate a rule
Keywords: edit segregation rule, disable rule, inactive rule
1. Open [Hazmat segregation rules](/admin/hazmat-segregation-rules).
2. Select the rule's row (or right-click it and choose **Edit**).
3. Change the fields, or set **Status** to **Inactive** to stop using the rule, and select
   **Save**.

### Find rules for a hazard class
Keywords: search segregation rules, filter by class
1. Open [Hazmat segregation rules](/admin/hazmat-segregation-rules).
2. Use the search box, or select **Filter** to narrow by **Status**, **Class A**, **Class B**,
   **Segregation type** or **Has exceptions**.

## Notes
Viewing the page needs read access to hazmat segregation rules; **New hazmat segregation rule**
appears only for people who can create them. Whether shipments are checked against these rules is
turned on with **Check Hazmat segregation** on [Shipment controls](/admin/shipment-controls). The
hazardous material records themselves are kept on
[Hazardous materials](/shipment-management/configuration-files/hazardous-materials).
