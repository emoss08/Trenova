---
path: /shipment-management/shipments/import
title: Import from rate confirmation
aliases: [rate con import, import rate confirmation, create load from PDF, upload rate confirmation, tender PDF, OCR shipment]
related:
  - /shipment-management/shipments
  - /shipment-management/orders
---

## What it's for
This page creates a shipment from a customer's rate confirmation document. You upload a PDF or image, Trenova extracts the shipment details (customer, stops, dates, rates and references), and you review each extracted field beside a preview of the document before creating the shipment. The import assistant beside the fields reads the document and the draft, accepts or corrects fields, matches stops to your locations, and says when the draft is ready. Its changes land on the page as if you had made them; nothing is saved until you create the shipment. Customer service and dispatch staff use it to enter tendered loads without retyping them.

## Tasks

### Create a shipment from a rate confirmation
Keywords: import rate con, upload PDF, build load from document
1. Open [Import from rate confirmation](/shipment-management/shipments/import).
2. Under **Upload rate confirmation**, drop or choose a PDF, JPG, PNG or WEBP file. Extraction starts automatically and usually takes a few seconds.
3. Review the fields next to the document preview. Select **Accept confident** to accept every field the extraction is sure of, and use **Issues only** to see just the fields that need review, are missing or conflict.
4. Complete **Required details**: **Customer**, **Service type**, **Shipment type** and **Rating method** (plus **Tractor type** and **Trailer type** if needed).
5. Check **Stops**, match each stop to a location, and correct any names, addresses, dates or windows.
6. Select **Create shipment**. The source document is attached to the new shipment.
7. Select **Open shipments** to go to the shipment list, or **Done**.

### Let the import assistant finish the draft
Keywords: AI import, assistant, match stops, missing fields
1. On [Import from rate confirmation](/shipment-management/shipments/import), after extraction, the import assistant opens beside the fields and starts on what is missing.
2. Ask it to accept fields, set the customer, service type, shipment type or rating method, or match each stop to a location. Each change appears on the page at once.
3. When a stop's facility is not one of your locations, the assistant proposes a new location. Review the card and approve it before it is saved; because the assistant has read a document from outside your organization, every change it proposes to your records waits for your approval.
4. If **Create shipment** fails, the assistant is told why and walks you through fixing it.

### Recover from a failed extraction
Keywords: extraction failed, wrong file, retry OCR
1. On [Import from rate confirmation](/shipment-management/shipments/import), if extraction fails, select **Retry** to run it again.
2. Or select **Replace file** to start over with a different document.

## Notes
Needs create permission for shipments. The import assistant also needs permission to start assistant conversations and access to the Shipment import assistant agent; without them the panel says which is missing. Re-extracting a document closes its conversation and starts a new one. If the shipment is created but the document could not be attached, the success screen says so; attach the document from the shipment's **Documents** tab.
