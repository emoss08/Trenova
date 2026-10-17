---
path: /admin/document-intelligence
aliases: [Document intelligence, OCR settings, document AI, document classification, document extraction, Docling, extraction engine, shipment drafts from documents, rate confirmation extraction]
related:
  - /admin/document-parsing-rules
  - /admin/document-operations
  - /admin/agent-control
  - /billing/configuration-files/document-types
---

## What it's for
Document extraction decides what Trenova does with uploaded documents: whether it reads text
from scanned pages, which engine reads them, how it classifies them into document kinds and
types, whether it extracts structured data with an AI provider, whether documents such as rate
confirmations become reviewable shipment drafts, and whether their text is indexed for search.
The page is one settings form in five cards: **Platform availability**, **Extraction engine**,
**Classification and extraction**, **Shipment draft extraction** and **Search and retrieval**.

Administrators use it to switch document processing on and choose how much of it is automatic.

## Tasks

### Turn document extraction on
Keywords: enable OCR, document processing on
1. Open [Document extraction](/admin/document-intelligence).
2. In **Platform availability**, turn on **Enable document extraction**. This is the master
   switch; every other option on the page stays unavailable while it is off.
3. Turn on **Enable OCR** to read scanned pages, photos and faxes that carry no text of their
   own.
4. Select **Save changes** in the bar that appears at the bottom.

### Choose the extraction engine
Keywords: Docling, legacy engine, switch engine, layout extraction
1. Open [Document extraction](/admin/document-intelligence).
2. In **Extraction engine**, choose **Legacy** or **Docling**. Docling reads the page layout,
   tables and scanned text, and every field it extracts needs an AI provider that serves
   document work; parsing rules are not applied. Legacy reads native PDF text, runs OCR on
   scans and applies your parsing rules.
3. Select **Save changes**. The engine applies to documents uploaded from then on; documents
   already read keep their text.

### Choose how documents are classified and extracted
Keywords: auto classify, document types, AI extraction
1. Open [Document extraction](/admin/document-intelligence).
2. In **Classification and extraction**, turn on the options you want: **Enable automatic
   classification**, **Enable AI-assisted classification**, **Enable automatic document type
   association**, **Enable automatic document type creation** and **Enable AI-assisted
   extraction**.
3. Select **Save changes**.

### Create shipment drafts from documents
Keywords: rate confirmation to shipment, shipment draft, auto create shipment
1. Open [Document extraction](/admin/document-intelligence).
2. In **Shipment draft extraction**, turn on **Enable shipment draft extraction**.
3. Tick the records whose documents may produce drafts: **Shipment**, **Trailer**, **Tractor**
   or **Worker**.
4. Select **Save changes**.

### Make document text searchable
Keywords: full text search, index documents
1. Open [Document extraction](/admin/document-intelligence).
2. In **Search and retrieval**, turn on **Enable full-text indexing**.
3. Select **Save changes**.

## Notes
Opening the page needs read access to document control. The AI-assisted options, and every
document read by the Docling engine, depend on an enabled AI provider assigned to the Document
extraction task, which is set up on [AI control](/admin/agent-control); a provider assigned only
to chat does not read documents. The server must also have document extraction turned on. Without
both, Docling documents are stored but not read, and upload screens say which is missing. Provider-specific parsing rules live on
[Parsing rules](/admin/document-parsing-rules) and apply only to the legacy engine.
