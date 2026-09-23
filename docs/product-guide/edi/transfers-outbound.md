---
path: /edi/transfers/outbound
aliases: [outbound load tenders, sent tenders, tendered loads, 204 sent, brokered tenders]
related:
  - /shipment-management/shipments
  - /edi/messages
  - /edi/partners
---

## What it's for
Outbound transfers are load tenders your organization has sent to trading partners, with their current state: submitted, awaiting mappings or approval on the receiving side, approved, rejected, expired or canceled.

Dispatch and brokerage staff use it to follow up on tenders they have sent. The list refreshes itself every 30 seconds.

## Tasks

### Send a shipment as a load tender
Keywords: tender load to partner, send 204, EDI tender
1. Open [Shipments](/shipment-management/shipments) and open the row actions for the shipment.
2. Select **Send EDI load tender**, then **Send tender**. The action is shown only for shipments in New status whose customer is linked to an EDI partner and that are eligible to be tendered.
3. The tender then appears on [Outbound transfers](/edi/transfers/outbound).

### Check the status of a sent tender
Keywords: tender status, did the partner accept, tender rejected
1. Open [Outbound transfers](/edi/transfers/outbound).
2. Find the tender by its **Reference** (the BOL) or **Partner**, and read its **Status**.
3. Select the row to see the **Tender**, **Freight** and **Mappings** tabs. The **Target shipment** column reads Pending until the receiving side approves the tender and its shipment is created.

### Cancel a sent tender
Keywords: withdraw tender, cancel 204
1. Open [Outbound transfers](/edi/transfers/outbound) and select the tender's row.
2. Select **Cancel transfer**. It is offered only while the tender is Submitted, Mapping required or Pending approval.

## Notes
Viewing transfers needs read access to EDI; canceling needs update access to EDI, and sending a tender from Shipments needs create access to EDI.
