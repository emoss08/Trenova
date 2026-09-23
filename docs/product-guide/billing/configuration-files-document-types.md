---
path: /billing/configuration-files/document-types
aliases: [document categories, paperwork types, BOL type, POD type, file types]
related:
  - /billing/configuration-files/document-packet-rules
  - /billing/configuration-files/customers
  - /billing/queue
---

## What it's for
Document types is the list of kinds of document your organization stores, such as a signed BOL or proof of delivery. Each type has a code, name, classification, category, color and description. Uploads are tagged with a type, and other settings refer to types by name: a customer's required documents for billing, and the packet rules that say which documents a shipment, worker or other record must carry.

Admins and billing leads maintain this list. Some types are built into the system and cannot be changed.

## Tasks

### Add a document type
Keywords: new document type, create document category
1. Open [Document types](/billing/configuration-files/document-types).
2. Select **New document type**.
3. Fill in **Code**, **Name**, **Classification** (**Public**, **Private**, **Sensitive** or **Regulatory**) and **Category** (for example **Shipment**, **Worker** or **Invoice**).
4. Optionally choose a **Color** and add a **Description**, then select **Save**.

### Change a document type
Keywords: edit document type, rename document type
1. Open [Document types](/billing/configuration-files/document-types).
2. Select the type's row to open it.
3. Change the fields and select **Save**. A type marked **System document type** cannot be modified.

### Find a document type
Keywords: search document types, filter by category
1. Open [Document types](/billing/configuration-files/document-types).
2. Type in the search box, or use **Filter** and **Sort** in the table toolbar, for example to show one **Category** or **Classification**.

## Notes
Viewing the page needs read access to document types; adding and changing them need create and update access.

To make a type required before a customer's shipments can be invoiced, add it to the customer's **Required document types** on [Customers](/billing/configuration-files/customers). To require it on a kind of record, add a rule on [Packet rules](/billing/configuration-files/document-packet-rules).
