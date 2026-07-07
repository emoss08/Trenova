# EDI Module Feature Inventory

This document inventories the EDI module as shipped. The module is a
self-service EDI platform: system administrators configure partners,
connections, transports, templates, and mappings themselves, for both
intercompany EDI (organization-to-organization inside a business unit) and
external partner EDI over the wire.

## Current Scope

The module supports two exchange styles end to end:

- **Intercompany**: load tender submit → review/approve → target shipment
  creation → shipment links → bidirectional status/lifecycle sync, entirely
  in-app with no wire protocol.
- **External**: X12 generation from the template engine, durable outbound
  delivery over SFTP, VAN, or AS2 with automatic retry and dead-lettering,
  inbound mailbox polling plus an AS2 HTTP receiver, inbound X12 parsing and
  routing (204/210/990/214/997/999), acknowledgment reconciliation, and
  outbound 997/999 generation.

## Feature Matrix

| Area | Status | Capability |
| --- | --- | --- |
| EDI navigation | Implemented | Partners, communication profiles, mapping profiles, template designer, inbound/outbound transfers, messages, and inbound files. |
| Permission model | Implemented | The `edi` resource with standard operations gates every route, action, and GraphQL query. |
| Partner management | Implemented | Internal and external partners with contact details, enablement flags, settings, linked customer, and default profile references. |
| Internal partner pairs | Implemented | Reciprocal internal partners created through a connection request/acceptance flow. |
| Connection lifecycle | Implemented | Pending acceptance, active, suspended, rejected, and revoked states with capability flags. |
| Communication profiles | Implemented | Internal, AS2, SFTP, and VAN methods with per-method config validation, encrypted secrets, and secret presence state. |
| SFTP transport | Implemented | Outbound push with host-key pinning, password or private-key auth, configurable directories and file naming; inbound mailbox polling with archive-after-read. |
| VAN transport | Implemented | VAN mailbox identity (provider, mailbox ID) plus SFTP gateway endpoint; outbound defaults to `/{mailboxId}/outbound`. |
| AS2 transport | Implemented | Outbound: S/MIME signed + encrypted (optional zlib compression) POST to the partner URL with sync or async MDNs; sync MDNs verify disposition, signature, and MIC before the message is marked Sent; async deliveries persist the AS2 Message-ID/MIC and stay Sending until the partner's MDN resolves them. Inbound: unauthenticated `POST /api/v1/edi/as2/inbound/` resolves the profile by AS2-From/AS2-To, decrypts and verifies signatures with the configured certificates, dedupes by checksum, stages into the standard inbound pipeline, and returns (or asynchronously posts) a signed MDN with the received-content MIC. Crypto lives in `shared/as2` on smallstep/pkcs7. |
| X12 generation | Implemented | 204/210/214/990/997/999 via the template engine with Starlark scripting, transforms, conditions, repeat loops, envelope control, and validation modes. |
| Control numbers | Implemented | Transactional per-partner/document-type ISA/GS/ST sequences with row locking. |
| Outbound delivery | Implemented | Per-message Temporal workflow (`DeliverEDIMessageWorkflow`, EDI task queue) with exponential retry (6 attempts, 30s→15m). Lifecycle: Queued → Sending → Sent / Failed → DeadLettered. |
| Delivery retry | Implemented | `POST /edi/messages/{id}/retry-delivery/` and a Retry Delivery action on the Messages page for failed or dead-lettered messages. |
| Inbound polling | Implemented | Temporal schedule `edi-inbound-poll` (every 2 minutes, overlap skip) polls every active SFTP/VAN profile that has an `inboundDirectory` configured and a partner assigned. |
| Inbound staging | Implemented | Files are checksummed, deduplicated (per-profile checksum and per-partner ISA control number), stored in `edi_inbound_files`, and archived on the remote mailbox. |
| Inbound parsing | Implemented | Envelope and transaction parsing via the X12 inspector; parse failures quarantine the file, per-transaction failures mark it partially processed. |
| 997/999 reconciliation | Implemented | Inbound acknowledgments are matched to sent messages by partner and control numbers; message ack status becomes Accepted/Rejected with diagnostics. |
| Inbound 990 | Implemented | Tender responses resolve the outbound tender recipient by shipment reference and set the shipment tender status to Accepted/Rejected. |
| Inbound 214 | Implemented | Status updates resolve the tendered shipment and record an auditable system comment with the AT7 status/reason codes. Automatic lifecycle mutation is intentionally not applied. |
| Inbound 210 | Implemented | Carrier freight invoices parse into a structured payload (B3/C3/N9/L11/G62/N1 loops/LX-L5-L0-L1/L3), correlate to the tendered shipment by reference, resolve the bill-to party through customer mappings, and persist an `edi_carrier_invoices` reconciliation record with variance against the tendered rate plus a shipment comment. |
| Inbound 204 | Implemented | External load tenders become inbound transfers reviewed in the existing transfers UI. Header entities map through sentinel `DEFAULT` mapping keys; locations/commodities map by partner codes. Purpose `04` changes supersede pending tenders. |
| Outbound acknowledgments | Implemented | When an inbound document profile expects acks, 997/999s are generated from auto-provisioned base templates and delivered through the outbound queue. |
| External tender responses | Implemented | Approving or rejecting an external inbound transfer generates an outbound 990 response automatically. |
| Mapping profiles | Implemented | Per-partner source→target entity mapping with preview, unresolved detection, and inline mapping during approval. |
| Intercompany sync | Implemented | Shipment links with sync policies, tender change detection/supersession, and 214-style status mirroring between linked shipments. |
| Messages monitoring | Implemented | `/edi/messages` lists every generated/received document with delivery status, attempts, ack status, and a read-only detail panel with raw X12. |
| Inbound file monitoring | Implemented | `/edi/inbound-files` lists received files with processing state, failure reasons, linked transactions, and a Reprocess action for quarantined/partial files. |
| GraphQL lists | Implemented | Partners, communication profiles, transfers, messages, and inbound files are served by persisted GraphQL connection queries; mutations and detail reads remain REST. |
| Template designer | Implemented | Draft/certify/activate/archive lifecycle, segment editing, X12 inspector, document preview and archive. |
| Operations dashboard | Implemented | `/edi/overview` (the module landing page) shows live grouped counts for delivery/ack status, inbound file status, and stuck inbound transfers, plus an overdue-ack tile and a recent-failures feed that deep-links into the message/inbound-file panels. Refreshes every 30 seconds via the `ediSummary` GraphQL query. |
| Failure alerting | Implemented | Dead-lettered messages and quarantined inbound files raise high-priority in-app notifications (org/BU-scoped, realtime push) with deep links; alerts are throttled per partner and event type within a 15-minute window via notification correlation IDs. |
| Test-case management | Implemented | `/edi/test-cases` provides full CRUD over certification scenarios (document profile + payload + expected diagnostics) with a Run Preview action that renders the payload through the partner template and opens the X12 inspector. |
| Audit logging | Implemented | Partner, connection, profile, transfer, and change actions log audit events with actor context. |

## Not Implemented

| Area | Status | Notes |
| --- | --- | --- |
| Automatic 214 lifecycle application | Not implemented | External carrier statuses are recorded as comments; automatic stop/lifecycle mutation requires a per-partner policy design. |

## Data Model

- `edi_partners`, `edi_connections`, `edi_communication_profiles`,
  `edi_mapping_profiles`/`edi_mapping_profile_items`: partner, handshake,
  transport, and translation configuration.
- `edi_templates`/`edi_template_versions`/`edi_template_segments` and the
  transaction-set dictionary tables: the template engine.
- `edi_partner_document_profiles` and `edi_control_number_sequences`: per
  transaction-set envelope/ack settings and control numbers.
- `edi_messages`: every generated or received X12 document with control
  numbers, delivery state, and acknowledgment state (`inbound_file_id` links
  received transactions to their source file).
- `edi_inbound_files`: staged mailbox files with checksums, ISA identity,
  processing status, and failure reasons.
- `edi_load_tender_transfers`, `edi_shipment_links`, `edi_tender_recipients`,
  `edi_tender_changes`, `edi_transfer_changes`: tender lifecycle, linkage, and
  change tracking (`inbound_message_id` marks external inbound transfers).

## Operations Runbook

### Stuck outbound deliveries

1. Open the Messages page and filter delivery status `Failed` or `DeadLettered`;
   the last error is shown in the detail panel.
2. In the Temporal UI, search for workflow ID `edi-deliver-message-{messageId}`
   on the EDI task queue to see attempt history.
3. Fix the cause (host key, credentials, directory permissions) on the
   communication profile, then use **Retry Delivery**. Retry restarts the same
   workflow ID, so a still-running retry cycle is never duplicated.

### Quarantined inbound files

1. Open the Inbound Files page and filter status `Quarantined` or
   `PartiallyProcessed`; the failure reason lists per-transaction warnings.
2. Common causes: unmapped entities (complete the partner mapping profile via
   the transfer approval screen), unmatched acknowledgments (control number
   mismatch), or malformed X12.
3. Use **Reprocess File** after fixing the cause. Duplicate interchanges are
   detected by ISA control number and marked `Duplicate` instead of
   reprocessing.

### Inbound polling

- Schedule ID `edi-inbound-poll` (Temporal), every 2 minutes, overlap policy
  skip. A profile is polled only when it is Active, method SFTP or VAN, has a
  partner assigned, and has a non-empty `inboundDirectory` config value.
- Processed remote files are moved to `archiveDirectory` (default
  `{inboundDirectory}/processed`). Archive failures are logged and tolerated —
  checksum dedup prevents double processing.

### Enabling a new external partner (checklist)

1. Create the partner (kind External) and, optionally, link the customer.
2. Create an SFTP or VAN communication profile for the partner, including the
   known host key, credentials, and inbound/outbound directories.
3. Certify and activate a template version (or rely on the seeded base
   templates) and create the partner document profiles per transaction set,
   with envelope identifiers.
4. Populate the partner mapping profile: `DEFAULT` keys for customer, service
   type, and rating formula, plus location/commodity codes as they appear in
   partner documents (unresolved entities can also be mapped inline during the
   first transfer approval).
5. Send a test 204 via Documents → Generate and verify delivery, then drop a
   test file into the inbound directory and verify it appears under Inbound
   Files.
