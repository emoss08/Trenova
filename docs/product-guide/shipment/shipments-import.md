---
path: /shipment-management/shipments/import
title: Import from rate confirmation
aliases: [rate con import, import rate confirmation, create load from PDF, upload rate confirmation, tender PDF, OCR shipment]
related:
  - /shipment-management/shipments
  - /shipment-management/orders
---

## What it's for
This page creates a shipment from a customer's rate confirmation document. You upload a PDF or image, Trenova extracts the shipment details (customer, stops, dates, rates and references), and you review each extracted field beside a preview of the document before creating the shipment. An AI assistant panel helps fill in or correct fields and says when the draft is ready. Customer service and dispatch staff use it to enter tendered loads without retyping them.

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

### Recover from a failed extraction
Keywords: extraction failed, wrong file, retry OCR
1. On [Import from rate confirmation](/shipment-management/shipments/import), if extraction fails, select **Retry** to run it again.
2. Or select **Replace file** to start over with a different document.

## Notes
Needs create permission for shipments. If the shipment is created but the document could not be attached, the success screen says so; attach the document from the shipment's **Documents** tab.
