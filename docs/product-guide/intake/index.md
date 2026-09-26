---
path: /intake
aliases: [scanning, scanned documents, scan queue, virtual printer, print to Trenova, paperwork queue, document intake, Kofax, batch scanning]
related:
  - /inbox
  - /shipment-management/shipments
---

## What it's for
Intake holds paper scanned into Trenova and documents printed into it from other programs through Trenova Capture, the Windows companion. Each scan or print job arrives as a stack. Trenova splits a stack into documents on patch code sheets, Trenova cover sheets or blank pages, reads each document, and suggests where it goes: the record a cover sheet names, the record it was scanned into, or a shipment whose number appears on the page. Clerks, dispatchers and billing staff use it to check each document and file it onto its record.

It is one queue with views on the left (**To file**, **Arriving**, **Filed**, **Discarded and expired**, **Everything**), filters for where the stack **Came from** and **Only mine**, the list of stacks in the middle with a search and a sort, and the open stack on the right with its documents.

## Tasks

### File a scanned stack
Keywords: file scans, assign documents, batch scan, process scanned paperwork
1. Open [Intake](/intake) and select **To file**.
2. Select a stack. Each document shows its pages, what Trenova suggested and why, and where it will be filed.
3. For each document, check **File onto**, the record and the **Document type**, and change any that are wrong.
4. Select **File** on one document, or the button at the top to file every document that has a record chosen. A document that cannot be filed says why on its card; the rest are filed.

### Split, join or reorder pages
Keywords: separate documents, merge pages, wrong split, rotate page, remove page
1. Open the stack in [Intake](/intake).
2. Drag a page to another document or to **Set aside**, or open a page's menu and choose **Start a new document after this page**, **Move to**, **Rotate right**, **Rotate left** or **Set aside**.
3. Select **Join with next** on a document to make it and the one after it one document.
4. Select **Save split**, or **Undo changes** to go back. Documents are filed as saved, so save before filing.

### Bring back a page that was set aside
Keywords: missing page, blank page removed, separator page
1. Open the stack in [Intake](/intake) and find the page under **Set aside**.
2. Drag it into a document, or open its menu and choose **Make it a document of its own**, then **Save split**.

### Discard a stack
Keywords: delete scan, throw away pages, duplicate scan
1. Open the stack in [Intake](/intake).
2. Open the menu beside the filing button and choose **Discard the stack**, then **Discard stack**. Documents already filed stay on their records; every other page is deleted.

### Find a stack
Keywords: search scans, find print job, expiring scans
1. Open [Intake](/intake) and pick a view on the left.
2. Type in **Search scanner, print job or device**, or change the sort to **Expiring soonest** to see what will be deleted first.

## Notes
Needs read access to capture batches to see the queue; filing needs update access and the permission to add documents to the record, and discarding needs delete access. A person whose access is limited to their own records sees only the stacks they captured.

Pages that are not filed are deleted when the stack's retention runs out, set in the organization's document settings. A stack within a week of that says so on its row and when it is opened.
