---
path: /billing/configuration-files/document-packet-rules
aliases: [required documents, document requirements, compliance documents, document packet, expiring documents]
related:
  - /billing/configuration-files/document-types
  - /billing/configuration-files/customers
---

## What it's for
Document packet rules decide which document types each kind of record must carry. A rule pairs a **Resource type** (**Shipment**, **Trailer**, **Tractor** or **Worker**) with a **Document type**, and says whether that document is **Required**, whether more than one is allowed, where it sits in the packet, and whether it must have an expiration date with an early warning.

Compliance, safety and billing leads use it to set up the document packet for shipments, equipment and workers, including documents that expire and should be flagged before they do.

## Tasks

### Add a packet rule
Keywords: require document, new packet rule, document requirement
1. Open [Packet rules](/billing/configuration-files/document-packet-rules).
2. Select **New document packet rule**.
3. Choose the **Resource type** and the **Document type** it requires.
4. Turn on **Required** if the document is mandatory, and **Allow multiple** if more than one may be attached. Set **Display order** (lower numbers appear first in the packet).
5. To track expiry, turn on **Expiration required** and set **Warning days**. Then select **Save**.

### Change a packet rule
Keywords: edit document requirement, change warning days
1. Open [Packet rules](/billing/configuration-files/document-packet-rules).
2. Select the rule's row, change the settings and select **Save**.

### Delete a packet rule
Keywords: remove document requirement
1. Open [Packet rules](/billing/configuration-files/document-packet-rules).
2. Right-click the rule's row and choose **Delete**, then confirm with **Delete**. This cannot be undone.

## Notes
The page is governed by document type permissions: viewing needs read access to document types, and adding or changing rules needs create or update access to them.

The document types a rule can use are managed on [Document types](/billing/configuration-files/document-types). Documents a customer requires before invoicing are set separately, in the customer's **Required document types** on [Customers](/billing/configuration-files/customers).
