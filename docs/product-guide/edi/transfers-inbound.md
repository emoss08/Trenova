---
path: /edi/transfers/inbound
aliases: [inbound load tenders, 204 tenders, tender review, incoming tenders, EDI tender approval, accept tender]
related:
  - /edi/mapping-profiles
  - /edi/partners
  - /shipment-management/shipments
---

## What it's for
Inbound transfers are load tenders your trading partners have sent you. Each one waits here until someone reviews it: a tender whose partner values are not all mapped shows Mapping required, a ready one shows Pending approval. Approving a tender creates the shipment on your side; rejecting it sends the reason back to the partner on the outbound 990 response.

Customer service and dispatch staff work this queue to accept or decline incoming freight. The list refreshes itself every 30 seconds.

## Tasks

### Review and approve a load tender
Keywords: accept tender, approve 204, create shipment from EDI
1. Open [Inbound transfers](/edi/transfers/inbound) and select the tender's row.
2. Check the summary, then the **Tender** tab (moves and stops) and the **Freight** tab (commodities and additional charges).
3. Open the **Mappings** tab. For each unresolved value under **Mapping preview**, choose the matching **Local record**.
4. Select **Approve**. It becomes available once every required mapping is resolved. Trenova creates the shipment, and **Open shipment** links to it once it exists.

### Reject a load tender
Keywords: decline tender, refuse load, 990 reject
1. Open [Inbound transfers](/edi/transfers/inbound) and select the tender's row.
2. Select **Reject**.
3. Enter the reason, which is shared with the submitting partner, then select **Reject transfer**.

### Approve or reject several tenders at once
Keywords: bulk approve, bulk reject, mass tender action
1. Open [Inbound transfers](/edi/transfers/inbound).
2. Tick the rows you want. A bar appears at the bottom of the screen.
3. Select **Approve**, or select **Reject**, enter a reason and select **Reject tenders**. Only tenders still awaiting review are acted on; the rest are skipped.

## Notes
Viewing transfers needs read access to EDI; approving and rejecting need update access to EDI. A tender can be approved or rejected only while it is Submitted, Mapping required or Pending approval. Saving mappings on [Mapping profiles](/edi/mapping-profiles) ahead of time keeps future tenders from that partner out of Mapping required.
