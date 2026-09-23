---
path: /edi/partners
aliases: [trading partners, EDI trading partner, EDI customers, EDI connections, partner setup, EDI onboarding]
related:
  - /edi/communication-profiles
  - /edi/mapping-profiles
  - /edi/designer
  - /edi/test-cases
---

## What it's for
The EDI Partners page lists your trading partners: external companies you exchange X12 documents with, and internal connections to other organizations in Trenova. Each partner holds its identifiers, contact, default transport and mapping profiles, and an onboarding readiness checklist. Connection requests from other organizations waiting for your answer appear above the table.

EDI coordinators and implementation teams use it to onboard new partners, keep partner defaults current, and accept or reject connection requests.

## Tasks

### Add an external trading partner
Keywords: new EDI partner, onboard partner, set up EDI customer
1. Open [Partners](/edi/partners).
2. Select New EDI connection at the top of the table. The **New EDI partner** panel opens on the **External partner** tab.
3. Under **Profile**, fill in **Partner code** (the SCAC or ISA ID), **Partner name**, **Status** and **Country**, and optionally **Customer**, **Timezone** and **Description**.
4. Under **Contact**, add the **Contact name**, **Contact email** and **Contact phone**.
5. Under **Defaults**, turn **Inbound enabled** and **Outbound enabled** on or off and pick a **Default transport profile** and **Default mapping profile** if they already exist.
6. Select **Create partner**.

### Request a connection with another Trenova organization
Keywords: internal EDI, connect organizations, partner pairing
1. Open [Partners](/edi/partners) and select New EDI connection.
2. Switch to the **Internal connection** tab.
3. Under **Organization pairing**, pick the **Target organization**. The partner code and name for both sides fill in from the two organizations; review them and the contact fields.
4. Select **Request connection**. The other organization sees the request at the top of its Partners page.

### Accept or reject a connection request
Keywords: pending EDI connection, approve connection
1. Open [Partners](/edi/partners). Requests waiting on your organization are listed under **Pending EDI connection requests**.
2. Select **Accept** to connect. Accepting creates reciprocal internal partners and communication profiles.
3. Or select **Reject**, enter a reason (it is shared with the requesting organization), then select **Reject connection**.

### Edit a partner and check its readiness
Keywords: partner settings, onboarding checklist, partner contact
1. Open [Partners](/edi/partners) and select the partner's row.
2. On the **Details** tab, change the fields you need, then select **Save partner**.
3. Open the **Readiness** tab to see which onboarding steps are complete. Missing steps offer shortcuts such as **Create communication profile**, **Open designer** or **Open test cases**.

### Map partner values to your records
Keywords: partner mapping, code mapping, cross reference
1. Open [Partners](/edi/partners), select the partner's row and open the **Mappings** tab.
2. Choose the tab for the kind of record, such as **Customer**.
3. Enter the partner's value in **Source value key** (and optionally **Source label**), pick your record in **Select local record** (for failure reasons, type the **Partner X12 code**), then select **Save**.
4. To remove a mapping, select the delete icon on its row.

## Notes
Viewing partners needs read access to EDI; creating partners and connections needs create access, and editing, mapping, accepting or rejecting needs update access to EDI. For internal partners, the code and name are controlled by the organization connection and cannot be edited here. Avoid changing a partner code after documents have been exchanged, since it is used in EDI envelopes.
