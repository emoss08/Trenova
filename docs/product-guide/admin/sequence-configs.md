---
path: /admin/sequence-configs
aliases: [numbering, PRO number format, invoice number format, auto numbering, number sequence, location code format]
related:
  - /shipment-management/shipments
  - /billing/invoices
  - /dispatch/locations
  - /admin/data-entry-controls
---

## What it's for
Sequence configuration sets the format of the numbers the system generates: PRO numbers,
consolidation, order and work order numbers, invoice numbers, journal batch and journal entry
numbers, manual journal requests, driver settlement numbers and location codes.

A list on the left groups the sequences under Operations, Billing, Accounting, Payroll and
Locations. Selecting one shows its settings and a **Live preview** of a sample value. Administrators
use it when setting up the organization or changing a numbering scheme.

## Tasks

### Change a number format
Keywords: prefix, PRO format, invoice numbering, add year to number
1. Open [Sequence configuration](/admin/sequence-configs).
2. Select the sequence in the list on the left, for example the invoice number.
3. Under **Core structure**, set the **Prefix** and **Sequence digits**, and turn on **Use
   separators** to pick a **Separator character**.
4. Under **Date components**, turn on **Include Year** (choose the **Year digits**), **Include
   Month**, **Include ISO Week Number** or **Include Day**.
5. Under **Context components**, turn on **Include Location Code** or **Include Business Unit Code**
   if wanted.
6. Check the **Live preview**, then select **Save changes**.

### Use a custom format or add check digits
Keywords: custom template, check digit, random digits, token template
1. Open [Sequence configuration](/admin/sequence-configs) and select the sequence.
2. Expand **Advanced**.
3. Turn on **Include Random Digits** and set the **Random digits count**, or turn on **Include Check
   Digit**.
4. To write the layout yourself, turn on **Allow custom format** and enter a **Custom format
   template** using tokens such as {P} for prefix, {Y} for year and {S} for the sequence.
5. Select **Save changes**.

### Set how location codes are generated
Keywords: location code, site code
1. Open [Sequence configuration](/admin/sequence-configs) and select the location code in the list.
2. Under **Code strategy**, choose the **Components** to build the code from (name, city, state or
   postal code), in order.
3. Set the **Component width**, **Sequence digits**, **Separator**, **Casing** and **Fallback
   prefix**.
4. Select **Save changes**.

### Go back to the default format
Keywords: reset numbering
1. Open [Sequence configuration](/admin/sequence-configs) and select the sequence.
2. Select **Reset to default**, then **Save changes**.

## Notes
Opening the page needs read access to sequence configuration; saving needs update access. A dot next
to a sequence in the list marks unsaved changes. Changes affect numbers generated from then on.
