---
path: /admin/document-operations
aliases: [document diagnostics, stuck document, failed upload, document recovery, reprocess document, missing preview, document not searchable]
related:
  - /admin/document-intelligence
  - /admin/document-parsing-rules
  - /admin/audit-logs
---

## What it's for
Document operations is a support tool for one uploaded document at a time. Paste a document's ID
and the page shows where it is in its lifecycle: a pipeline of **Upload**, **Preview**,
**Extraction** and **Draft** with each step's status, the file's details (the record it belongs to
under **Resource**, **Version**, **Created**, **Updated**, **Preview status**, **Content status**,
**Draft status** and **Detected kind**), whether **Extracted content** and a **Shipment draft**
exist, and any errors detected.

Below that it lists the document's **Version history**, its **Upload sessions** and the
**Workflow references** behind its background processing, and offers **Recovery actions** to
restart the steps that failed. Administrators and support staff use it when a document has no
preview, its text was never extracted, or it does not show up in search.

## Tasks

### Inspect a document
Keywords: document status, why no preview, document lifecycle, check extraction
1. Open [Document operations](/admin/document-operations).
2. Paste the document's ID into **Paste a document ID to inspect...** and select **Inspect**.
3. Read the pipeline and statuses to see which step stopped, and the error count if any errors
   were detected.
4. Check **Version history**, **Upload sessions** and **Workflow references** for more detail.

### Re-run a failed step
Keywords: reprocess, retry extraction, rebuild thumbnail, fix search index
1. Inspect the document in [Document operations](/admin/document-operations).
2. Under **Recovery actions**, choose the step to restart:
   - **Reextract content** re-processes the document's text and structured data.
   - **Regenerate preview** starts a new thumbnail workflow.
   - **Resync search** updates the document's search index entry with its latest data.
3. Select **Confirm**. The action starts in the background; inspect the document again after a
   short wait to see the result.

## Notes
Viewing the page needs read access to document operations; the recovery actions need update
access to document operations. If the ID is wrong or the document no longer exists, the page shows
"Failed to load diagnostics" with **Retry**.
