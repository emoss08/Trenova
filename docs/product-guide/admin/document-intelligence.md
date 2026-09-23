---
path: /admin/document-intelligence
aliases: [OCR settings, document AI, document classification, document extraction, shipment drafts from documents, rate confirmation extraction]
related:
  - /admin/document-parsing-rules
  - /admin/document-operations
  - /admin/agent-control
  - /billing/configuration-files/document-types
---

## What it's for
Document intelligence decides what Trenova does with uploaded documents: whether it reads them
with OCR, classifies them into document kinds and types, extracts structured data (optionally
with an AI provider), turns documents such as rate confirmations into reviewable shipment
drafts, and indexes their text for search. The page is one settings form in four cards:
**Platform availability**, **Classification and extraction**, **Shipment draft extraction** and
**Search and retrieval**.

Administrators use it to switch document processing on and choose how much of it is automatic.

## Tasks

### Turn document intelligence on
Keywords: enable OCR, document processing on
1. Open [Document intelligence](/admin/document-intelligence).
2. In **Platform availability**, turn on **Enable document intelligence**. This is the master
   switch; every other option on the page stays unavailable while it is off.
3. Turn on **Enable OCR** to read scanned documents when native text extraction is not enough.
4. Select **Save changes** in the bar that appears at the bottom.

### Choose how documents are classified and extracted
Keywords: auto classify, document types, AI extraction
1. Open [Document intelligence](/admin/document-intelligence).
2. In **Classification and extraction**, turn on the options you want: **Enable automatic
   classification**, **Enable AI-assisted classification**, **Enable automatic document type
   association**, **Enable automatic document type creation** and **Enable AI-assisted
   extraction**.
3. Select **Save changes**.

### Create shipment drafts from documents
Keywords: rate confirmation to shipment, shipment draft, auto create shipment
1. Open [Document intelligence](/admin/document-intelligence).
2. In **Shipment draft extraction**, turn on **Enable shipment draft extraction**.
3. Tick the records whose documents may produce drafts: **Shipment**, **Trailer**, **Tractor**
   or **Worker**.
4. Select **Save changes**.

### Make document text searchable
Keywords: full text search, index documents
1. Open [Document intelligence](/admin/document-intelligence).
2. In **Search and retrieval**, turn on **Enable full-text indexing**.
3. Select **Save changes**.

## Notes
Opening the page needs read access to document control. The AI-assisted options depend on an
enabled AI provider that serves document work, which is set up on
[AI control](/admin/agent-control). Provider-specific parsing rules live on
[Parsing rules](/admin/document-parsing-rules).
