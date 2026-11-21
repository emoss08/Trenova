---
path: /admin/document-intelligence
aliases: [Document intelligence, OCR settings, document AI, document classification, document extraction, Docling, shipment drafts from documents, rate confirmation extraction]
related:
  - /admin/agent-control
  - /billing/configuration-files/document-types
---

## What it's for
Document extraction decides what Trenova does with uploaded documents: whether it reads them,
including scanned pages, how it files them under your document types, and whether rate
confirmations become reviewable shipment drafts. Docling reads every document: the page layout,
tables and scanned text. It is one settings form, saved once, in tabs: **Reading** (whether
documents are read at all, scanned pages and search by content), **Filing** (recognizing
document kinds and filing them under document types) and **Shipment drafts** (drafting
shipments from rate confirmations and reading them with AI). When documents cannot be read,
**Reading** says what is missing. While reading is off, **Filing** and **Shipment drafts** say
that their settings do nothing yet, and **Open Reading** takes you to the switch.

Administrators use it to switch document processing on and choose how much of it is automatic.
When a save is refused, the tab holding the setting to fix opens and is marked. The capture
settings are on [Capture](/admin/capture).

## Tasks

### Turn document extraction on
Keywords: enable OCR, document processing on, full text search, index documents
1. Open [Document extraction](/admin/document-intelligence).
2. On **Reading**, turn on **Read uploaded documents**. This is the master switch: nothing else
   on the page acts while it is off.
3. **Read scanned pages** and **Search by content** appear beneath it. Turn on **Read scanned
   pages** to read scans, photos and faxes that carry no text of their own, and **Search by
   content** so people can find a document by the words inside it.
4. Select **Save changes** in the bar that appears at the bottom.

### Choose how documents are filed
Keywords: auto classify, document types, AI classification
1. Open [Document extraction](/admin/document-intelligence).
2. On **Filing**, under **Document kinds**, turn on **Recognize document kinds**. **Ask AI when
   unsure** appears beneath it. With it on, a document whose wording does not clearly say what it
   is goes to an AI model, and the model's answer decides the kind when the model is sure of it.
   A document the wording recognizes clearly keeps that kind whatever the model says, and when
   neither is sure the document keeps the kind its wording suggests. The kind decides the
   document type it is filed under, whether it becomes a shipment draft and whether **Read with
   AI** reads it. With it off, no model is asked what a document is.
3. Under **Document types**, turn on **File under a document type** so a recognized document
   gets the matching type from your list. **Add missing types** appears beneath it; turn it on
   to add a type when none matches.
4. Select **Save changes**.

### Create shipment drafts from rate confirmations
Keywords: rate confirmation to shipment, shipment draft, auto create shipment, AI extraction
1. Open [Document extraction](/admin/document-intelligence).
2. On **Shipment drafts**, turn on **Draft shipments**.
3. Under **Draft from**, tick the records whose uploaded rate confirmations may become drafts:
   **Shipments**, **Trailers**, **Tractors** or **Workers**. At least one is needed while
   drafting is on.
4. Under **AI reading**, turn on **Read with AI** so an AI model reads each rate confirmation's
   stops, dates and rate.
5. Select **Save changes**.

## Notes
Opening the page needs read access to document control. Every document Docling reads, and the
AI-assisted options, depend on an enabled AI provider assigned to the Document extraction task,
which is set up on [AI control](/admin/agent-control); a provider assigned only to chat does not
read documents. The server must also have document extraction turned on and its document
converter configured. Without all three, documents are stored but not read, and this page, the
upload screens and the document's failure message say which is missing.
