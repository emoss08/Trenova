---
path: /admin/sequence-configs
aliases: [numbering, PRO number format, invoice number format, settlement number format, auto numbering, number sequence, location code format]
related:
  - /shipment-management/shipments
  - /billing/invoices
  - /dispatch/locations
  - /admin/data-entry-controls
---

## What it's for
Sequence configuration sets the format of the numbers the system generates: PRO numbers,
consolidation, order and work order numbers, invoice and invoice run numbers, journal batch and
journal entry numbers, manual journal requests, driver and carrier settlement numbers and location
codes.

Five tabs group the sequences: **Shipments**, **Billing**, **Accounting**, **Settlements** and
**Locations**. Each sequence is a section with a readout of the next number it would issue; press
a part of the readout to jump to the setting behind it. Administrators use it when setting up the
organization or changing a numbering scheme.

## Tasks

### Change a number format
Keywords: prefix, PRO format, invoice numbering, add year to number
1. Open [Sequence configuration](/admin/sequence-configs) and select the tab that holds the
   sequence, for example **Billing** for invoice numbers.
2. In the sequence's section, set the **Prefix** and the **Counter digits**.
3. Turn on the pieces to add to the number: **Year** (choose the **Year digits**), **Month**,
   **Week**, **Day**, and for numbers drawn for records **Business unit code** and **Location
   code**.
4. Turn on **Separators** to pick a **Separator** between the pieces.
5. Check the readout, then select **Save changes**.

### Use a custom format or add check digits
Keywords: custom template, check digit, random digits, token template
1. Open [Sequence configuration](/admin/sequence-configs) and select the tab that holds the
   sequence.
2. Turn on **Random digits** and set **How many**, or turn on **Check digit**.
3. To write the layout yourself, turn on **Write your own pattern** and enter the **Pattern**
   using placeholders such as {P} for prefix, {Y} for year and {S} for the sequence; pressing a
   placeholder under the pattern adds it to the end.
4. Select **Save changes**.

### Set how location codes are generated
Keywords: location code, site code
1. Open [Sequence configuration](/admin/sequence-configs) and select the **Locations** tab.
2. Under **Parts**, choose what the code is built from (name, city, state or ZIP code), in order.
3. Set the **Characters per part**, **Counter digits**, **Separator**, **Letter case** and
   **Fallback**.
4. Select **Save changes**.

### Go back to the default format
Keywords: reset numbering
1. Open [Sequence configuration](/admin/sequence-configs) and select the tab that holds the
   sequence.
2. Select **Use the default**, then **Save changes**.

## Notes
Opening the page needs read access to sequence configuration; saving needs update access. A save that is refused opens the tab holding the setting to fix. Changes affect numbers generated from then on.
