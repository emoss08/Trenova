# EDI Module Feature Inventory

This document inventories the EDI module as it exists today and separates current
capabilities from future-state EDI expectations. Current implementation is
primarily internal organization-to-organization load tender transfer inside
Trenova. Future support can expand this foundation into standards-based external
EDI exchange.

## Current Scope

The current module supports internal EDI-style shipment tendering between
Trenova organizations. It models partners, organization-to-organization
connections, communication profiles, mapping profiles, load tender transfer
review, target shipment creation, and transfer lifecycle visibility.

It is not yet a full external EDI gateway. AS2, SFTP, and VAN profile data can be
configured, but the module does not yet serialize, transmit, receive, parse, or
acknowledge X12 documents end to end.

## Current Feature Matrix

| Area | Status | Current capability |
| --- | --- | --- |
| EDI navigation | Implemented | EDI module navigation exposes partners, communication profiles, mapping profiles, inbound transfers, and outbound transfers. |
| Permission model | Implemented | The module uses the restricted `edi` resource with standard create, read, update, and delete operations. |
| Partner management | Implemented | Stores internal and external partners with code, name, status, contact details, country, inbound/outbound enablement, settings, linked customer, internal organization, and default profile references. |
| Internal partner pairs | Implemented | Creates reciprocal internal partners through an internal connection request and acceptance flow. |
| Connection lifecycle | Implemented | Tracks connection method, capabilities, source/target organizations, source/target partner configuration, pending acceptance, active, suspended, rejected, and revoked states. |
| Pending connection review | Implemented | Target organizations can accept or reject pending internal connection requests from the EDI partner workspace. |
| Communication profiles | Implemented | Stores transport profile metadata, method, active/inactive status, config, encrypted secrets, and secret presence state. |
| Communication profile validation | Partial | Validates required fields for Internal, AS2, SFTP, and VAN profile configurations. Transport execution is not implemented. |
| Secret handling | Implemented | Communication profile secrets are encrypted when an encryption service is configured and are returned only as presence metadata. |
| Mapping profiles | Implemented | Maintains per-partner mapping profiles and mapping items from source IDs to target IDs. |
| Supported mapping entities | Implemented | Supports customer, service type, shipment type, formula template, location, commodity, and accessorial charge mappings. |
| Mapping preview | Implemented | Shows resolved and unresolved mappings for a transfer before approval. |
| Inline mapping during approval | Implemented | Inbound transfer review can submit missing mapping items while approving a transfer. |
| Load tender submission | Implemented | New eligible shipments can be submitted as outbound internal load tenders to enabled internal partners with an active internal connection and active internal profiles. |
| Tender eligibility | Implemented | Only `New` shipments without an active or accepted tender can be tendered; rejected, expired, and canceled tenders can be retried. |
| Tender payload snapshot | Implemented | Captures shipment, customer, service, rating, BOL, pieces, weight, temperature, charge, move, stop, commodity, and accessorial data into a JSON payload. |
| Transfer queues | Implemented | Lists inbound and outbound load tender transfers separately. |
| Transfer statuses | Implemented | Supports submitted, mapping required, pending approval, processing, approved, rejected, expired, canceled, and failed lifecycle states. |
| Inbound review | Implemented | Inbound users can inspect tender, freight, route, stop, and mapping details before accepting or rejecting. |
| Outbound review | Implemented | Outbound users can inspect sent tenders and cancel actionable transfers. |
| Approval workflow | Implemented | Approval starts a Temporal workflow on the EDI task queue and processes the target shipment creation asynchronously. |
| Target shipment creation | Implemented | Approved transfers create a target shipment with mapped entities, EDI entry method, accepted tender status, moves, stops, commodities, and additional charges. |
| Source tender status updates | Implemented | Source shipments move through tendered, accepted, rejected, expired, and canceled tender statuses. |
| Shipment comments | Implemented | System comments are written for tender submission, acceptance, rejection, cancellation, expiration, and transfer-change review events. |
| Shipment links | Implemented | Accepted tenders create links between source and target shipments with sync policy, field ownership, and link status. |
| Field ownership model | Implemented | Default ownership separates source-owned commercial tender fields from target-owned operational execution fields. |
| Transfer changes | Partial | Transfer change records can be listed, inspected, applied, or rejected, with conflict status and idempotency fields. Automatic change detection/application is not fully represented in the current service flow. |
| Audit logging | Implemented | EDI partner, connection, profile, transfer, and transfer-change actions log audit events when audit service and actor context are available. |
| Search/list support | Implemented | Partners, profiles, links, and changes include searchable fields or list repositories for table views. |

## Current Data Model

The current module persists these main records:

- `edi_partners`: partner identity, direction flags, contact data, linked
  internal organization/customer, and default profile references.
- `edi_connections`: source/target organization relationship, method,
  capabilities, partner configs, and connection lifecycle metadata.
- `edi_communication_profiles`: method-specific transport/envelope config,
  encrypted secrets, partner/connection links, and active/inactive state.
- `edi_mapping_profiles` and `edi_mapping_profile_items`: per-partner entity ID
  translation rules.
- `edi_load_tender_transfers`: source/target shipment tender payload,
  lifecycle state, mapping snapshot, workflow IDs, approval/rejection/cancel
  metadata, and failure reason.
- `edi_shipment_links`: linkage between source and target shipments after
  approval, sync policy, field ownership, and link status.
- `edi_transfer_changes`: reviewed change records with direction, type,
  conflict status, idempotency key, payload, diff, and review/apply metadata.

## Future-State Gap Checklist

Transportation EDI commonly centers on X12 transaction sets such as 204 Motor
Carrier Load Tender, 990 Response to a Load Tender, 214 Shipment Status, 210
Motor Carrier Freight Details and Invoice, and 997/999 acknowledgments. The
current module has strong internal tender workflow foundations, but the items
below are still future-state unless explicitly implemented later.

| Future capability | Status | Notes |
| --- | --- | --- |
| X12 204 outbound generation | Not implemented | Convert Trenova load tender data into partner-specific X12 204 envelopes and segments. |
| X12 204 inbound parsing | Not implemented | Parse external customer/broker load tenders into draft or reviewable Trenova shipments. |
| X12 990 tender response | Not implemented | Send and receive accept/decline responses tied to load tender transfers. |
| X12 214 shipment status | Not implemented | Send and receive pickup, departure, arrival, delivery, delay, and exception status events. |
| X12 210 freight invoice | Not implemented | Generate freight invoice EDI from billing/invoice data and ingest partner invoice responses where applicable. |
| 997/999 acknowledgments | Not implemented | Generate and process functional or implementation acknowledgments for received X12 documents. |
| ISA/GS/ST control number management | Not implemented | Allocate, persist, validate, and reconcile interchange, group, and transaction control numbers. |
| External transport execution | Not implemented | Run AS2, SFTP, or VAN send/receive jobs using communication profile config and secrets. |
| Inbound mailbox polling | Not implemented | Poll SFTP/VAN mailboxes or receive AS2 messages and hand documents to validation/parsing. |
| AS2 MDN handling | Not implemented | Support signed/encrypted AS2 payloads and synchronous/asynchronous MDNs. |
| Document validation | Not implemented | Validate syntax, envelope values, required segments, partner rules, duplicate control numbers, and business constraints. |
| Message archive | Not implemented | Store raw inbound/outbound payloads, parsed payloads, acknowledgments, errors, and replay metadata. |
| Retry and dead-letter handling | Not implemented | Retry transient transport failures and isolate poison messages for manual recovery. |
| Partner certification/testing | Not implemented | Provide test-mode profiles, sample messages, validation reports, and partner onboarding evidence. |
| Operational monitoring | Not implemented | Add dashboards or alerts for failed transmissions, missing acknowledgments, stale tenders, and partner outages. |
| Exception workbench | Not implemented | Centralize mapping errors, validation errors, duplicate messages, rejected acknowledgments, and replay actions. |
| Multi-document partner capabilities | Planned | Existing connection capability fields already anticipate load tender, shipment status, and invoice features. |

## References

- X12 transaction set catalog:
  <https://x12.org/products/transaction-sets>
- Stedi X12 transaction set reference:
  <https://www.stedi.com/edi/x12-003020/transaction-set>
- EDI transportation cycle overview:
  <https://ediacademy.com/blog/edi-transportation-cycle/>
- EDI 204 Motor Carrier Load Tender overview:
  <https://ediacademy.com/blog/edi-204-motor-carrier-load-tender/>
