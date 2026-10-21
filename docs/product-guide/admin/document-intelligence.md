---
path: /admin/document-intelligence
aliases: [Document intelligence, OCR settings, document AI, document classification, document extraction, Docling, shipment drafts from documents, rate confirmation extraction]
related:
  - /admin/document-parsing-rules
  - /admin/document-operations
  - /admin/agent-control
  - /billing/configuration-files/document-types
---

## What it's for
Document extraction decides what Trenova does with uploaded documents: whether it reads text
from scanned pages, how it classifies them into document kinds and types, whether it extracts
structured data with an AI provider, whether documents such as rate confirmations become
reviewable shipment drafts, and whether their text is indexed for search. Docling reads every
document: the page layout, tables and scanned text.
The page is one settings form in four cards: **Platform availability**, **Classification and
extraction**, **Shipment draft extraction** and **Search and retrieval**. When documents cannot
be read, **Platform availability** says what is missing.

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
Opening the page needs read access to document control. Every document Docling reads, and the
AI-assisted options, depend on an enabled AI provider assigned to the Document extraction task,
which is set up on [AI control](/admin/agent-control); a provider assigned only to chat does not
read documents. The server must also have document extraction turned on and its document
converter configured. Without all three, documents are stored but not read, and this page, the
upload screens and the document's failure message say which is missing. The provider-specific
rules on [Parsing rules](/admin/document-parsing-rules) are applied to the text Docling reads from
a rate confirmation, after it is classified and before AI extraction.
