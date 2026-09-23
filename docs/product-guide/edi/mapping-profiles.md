---
path: /edi/mapping-profiles
aliases: [EDI mappings, code mapping, cross reference, value mapping, entity mapping, partner codes]
related:
  - /edi/partners
  - /edi/transfers/inbound
---

## What it's for
Mapping profiles translate a trading partner's values into your own records. When a partner tenders a load, it refers to customers, locations, commodities, service types, shipment types, rating templates and accessorial charges by its own codes; a mapping profile says which of your records each code means. Service failure reasons map the other way, to the partner's X12 code. Each partner has one mapping profile, created when its first mapping is saved.

EDI coordinators maintain these so inbound load tenders can be approved without stopping for unmapped values.

## Tasks

### Create a mapping profile for a partner
Keywords: new mapping, map partner codes
1. Open [Mapping profiles](/edi/mapping-profiles).
2. Select New EDI mapping profile at the top of the table. The **New mapping profile** panel opens.
3. Under **Partner**, choose the trading partner.
4. Pick the tab for the kind of record you are mapping, such as **Customer**.
5. Enter the partner's value in **Source value key** and optionally a **Source label**, choose your record in **Select local record**, and optionally add a **Target label**.
6. Select **Save**. Saving the first mapping creates the partner's mapping profile.

### Add or remove mappings on an existing profile
Keywords: edit mapping, delete mapping, fix unmapped value
1. Open [Mapping profiles](/edi/mapping-profiles) and select the profile's row.
2. Pick the tab for the kind of record.
3. To add a mapping, fill in **Source value key** and **Select local record**, then select **Save**. For failure reasons, choose the reason and type the **Partner X12 code** instead.
4. To remove a mapping, select the delete icon on its row.

## Notes
Viewing mapping profiles needs read access to EDI; adding or deleting mappings needs update access to EDI. The same mappings can be edited from a partner's **Mappings** tab on [Partners](/edi/partners). Inbound transfers with values that have no mapping wait in Mapping required status on [Inbound transfers](/edi/transfers/inbound).
