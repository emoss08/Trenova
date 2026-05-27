# Trenova Enterprise TMS Gap Analysis

Date: 2026-05-26

## Executive Summary

Trenova is not a basic CRUD starter. The codebase already has a serious foundation for an enterprise transportation management system:

- Go hexagonal backend with domain, service, port, repository, and adapter separation.
- PostgreSQL/PostGIS system of record with Redis, MinIO/R2-style object storage, Meilisearch, and GTC CDC.
- Temporal workers for async jobs and long-running processes.
- Shipment, movement, assignment, equipment, worker, customer, billing, invoice, accounting, document, EDI, audit, RBAC, SSO, integration, search, and analytics surfaces.
- A React client with meaningful operational/admin/accounting/document/EDI screens.

The primary gaps are enterprise TMS depth and operational maturity, not the absence of a platform. The highest-risk areas are:

1. The operating model still centers heavily on `Shipment`; enterprise TMS products need clearer separation between order, load/shipment execution, trip/movement plan, tender, and financial artifacts.
2. Carrier/broker procurement, settlements/payables, claims, maintenance, and compliance evidence workflows are missing or shallow compared with the research benchmark.
3. Integrations exist, especially EDI and Samsara, but the integration hub is not yet broad enough for enterprise partner connectivity.
4. The infrastructure is strong for local development and early SaaS, but production deployment, DR, event contracts, tenant operations, and analytics/warehouse patterns need explicit hardening.

The best next move is not a rewrite or microservice split. Keep the current modular monolith, strengthen the domain boundaries, add durable event contracts, and fill the missing TMS product domains in priority order.

## Current Strengths

### Architecture

Trenova is already aligned with the report's recommended architecture in several important ways:

- The backend follows a ports-and-adapters structure under `services/tms/internal`.
- Domain code lives under `internal/core/domain`, services under `internal/core/services`, and infrastructure under `internal/infrastructure`.
- Dependency injection is centralized through Uber FX bootstrap modules.
- PostgreSQL is the relational core, with PostGIS support for location/geospatial workflows.
- Redis is used for caching/buffering/session-style support.
- MinIO/R2-style object storage supports documents.
- Meilisearch plus GTC CDC gives a read/search projection path.
- Temporal is already present for async jobs, including audit, billing, document intelligence, document uploads, EDI, exchange rates, fiscal jobs, invoice adjustments, Samsara sync, shipments, SMS, thumbnails, and weather alerts.

This is a strong foundation. The codebase should stay modular while the business domains mature.

### Product Domains Already Present

The repository already contains meaningful coverage for:

- Shipments, shipment moves, stops, assignment, shipment events, comments, holds, and billing readiness.
- Customers, billing profiles, email profiles, credit controls, invoice requirements, and consolidation settings.
- Workers/drivers with CDL, license, medical card, hazmat, MVR, drug test, ELD-exempt, short-haul-exempt, and qualification-ish fields.
- Tractors, trailers, equipment types, equipment manufacturers, fleet codes, and equipment continuity helpers.
- Accessorial charges, formulas/rating templates, shipment commercial calculation, detention charge support, and distance overrides.
- Billing queue, invoices, invoice adjustments, customer payments, bank receipts, customer ledger, journal entries, fiscal periods, fiscal years, GL accounts, and balances.
- Documents, document types, document packets, parsing rules, document intelligence, document upload sessions, document content, thumbnails, and document search projections.
- EDI partners, EDI connections, document profiles, mapping profiles, X12 rendering, test cases, messages, validation errors, source contexts, templates, transfer approval workflow, and load tender transfer models.
- Audit logging with buffering, sensitive data handling, DLQ support, realtime invalidation, and searchability.
- RBAC, permissions, route policy, SSO configuration, API keys, sessions, CSRF, rate limiting, and security controls.
- Observability foundations: OpenTelemetry tracing, Prometheus-style metrics, monitoring server, health/readiness/liveness, pprof, HTTP/database/document/error/Temporal metrics.
- Client screens for shipments, workers, tractors, trailers, customers, billing, invoices, accounting, documents, EDI, admin, integrations, audit logs, roles, users, and analytics.

This gives Trenova a credible base for an asset-carrier-first TMS.

## Material Gaps

### 1. Order, Shipment, Load, Trip, and Financial Boundaries

The research report correctly warns against modeling "loads" as the only first-class object. Trenova currently has `Shipment`, `ShipmentMove`, `Stop`, and `Assignment`, but it does not appear to have explicit first-class domains for:

- Customer order or commercial request.
- Load as an executable shipment unit separate from order intent.
- Trip or movement plan as the asset/driver execution plan.
- Tender as a durable workflow aggregate separate from EDI transfer status.
- Financial artifacts as evidence-linked outputs from execution.

Current shipment fields include commercial, operational, tender, billing, rating, and execution state in one aggregate. That is workable early, but it will become a bottleneck for complex enterprise flows:

- One customer order producing multiple loads.
- Multiple orders consolidated into one load.
- Split, merge, relay, cross-dock, or interline movements.
- Re-rating after operational changes.
- Billing or settlement corrections after execution.
- Brokered versus asset-executed capacity decisions.

Recommended direction:

- Introduce `Order` as the commercial commitment/intake object.
- Keep `Shipment` or rename/evolve toward `Load` for the executable movement unit.
- Promote `Trip` or `MovementPlan` for asset/driver execution planning.
- Treat invoice, settlement, payable, claim, and adjustment records as financial artifacts linked to operational evidence.

### 2. Carrier and Broker Procurement

The codebase has EDI partners and load tender transfer support, but a full carrier procurement domain appears missing.

Missing or shallow capabilities:

- Carrier master separate from customer/vendor/EDI partner concepts.
- Carrier onboarding workflow.
- Operating authority, DOT/MC identity, insurance policies, W-9, contracts, lane preferences, and document expirations.
- Carrier compliance status and safety qualification.
- Carrier scorecards and service history.
- Routing guide, waterfall tendering, spot tendering, tender retry, and fallback rules.
- Tender attempts and tender responses as a canonical workflow independent of EDI transport.
- Carrier portal or API for accepting/rejecting tenders.
- Load-board integrations such as DAT or Truckstop.

Recommended direction:

- Add a canonical tender/procurement model first.
- Hang EDI, API, webhook, portal, and email channels off that model.
- Treat EDI 204/990 as one protocol implementation, not the tender domain itself.

### 3. Settlements, Payables, and Carrier/Driver Pay

Trenova has strong AR/invoicing/accounting direction, but the research benchmark expects both receivables and payables.

Missing or shallow capabilities:

- Driver settlement.
- Owner-operator settlement.
- Carrier payable.
- Pay rules by driver, equipment, fleet, lane, accessorial, detention, fuel, or contract.
- Deductions, advances, escrow, chargebacks, reimbursements, lumper handling, fines, and adjustment approvals.
- Settlement statements and audit trails.
- Payable approval workflow.
- Integration/export to accounting/AP systems.
- Linkage between operational evidence, invoice lines, and settlement/payable lines.

Recommended direction:

- Add settlements/payables after the order/load/trip split is clarified.
- Reuse the existing formula engine where it fits, but do not force all pay logic into shipment rating formulas.
- Make every pay artifact traceable to shipment events, POD/documents, accessorial evidence, and approval history.

### 4. Claims and OS&D

Claims are called out in the research report but do not appear to be a first-class domain in the current codebase.

Missing capabilities:

- OS&D events: overage, shortage, damage, refusal, concealed damage.
- Claim case lifecycle.
- Claim reserves and recovery tracking.
- Claim documents/photos/evidence.
- Internal responsibility/root-cause tagging.
- Customer/carrier communication history.
- Links to invoice adjustments, credit memos, and insurance recovery.
- Claim analytics.

Recommended direction:

- Add claims as a separate bounded context, not just a shipment note type.
- Allow claims to reference shipments, stops, documents, invoices, carriers, workers, equipment, and customers.

### 5. Maintenance and Equipment Compliance

Tractors and trailers exist, and trailers have `lastInspectionDate`, but a real maintenance context appears missing.

Missing or shallow capabilities:

- Maintenance work orders.
- Preventive maintenance schedules.
- Inspection records.
- Defects and defect resolution.
- Out-of-service workflow.
- Repair vendor/facility tracking.
- Annual inspection evidence.
- Maintenance documents and search.
- Cost history by tractor/trailer.
- Maintenance-driven dispatch holds.

Recommended direction:

- Add a maintenance domain with work orders, inspections, defects, PM schedules, and equipment holds.
- Integrate it with dispatch assignment eligibility and document storage.

### 6. Driver Compliance Evidence

Worker profiles include several important compliance fields, but evidence and workflow depth are not yet enterprise-grade.

Missing or shallow capabilities:

- Driver Qualification File document inventory.
- Prior employer investigation tracking.
- Annual MVR tasks and evidence.
- Clearinghouse query evidence.
- Drug/alcohol program evidence.
- Expiration task queues and compliance workflows.
- Compliance document retention policy.
- Audit-ready compliance views.
- HOS data ingestion beyond flags.

Recommended direction:

- Keep ELD/HOS certified output external.
- Integrate ELD/HOS availability and exception signals from providers.
- Add evidence capture, expiration workflows, compliance holds, and audit views inside Trenova.

### 7. Terminal, Yard, Dock, and Gate Operations

Locations and location categories exist, including facility-style categories and geofences. The deeper yard/terminal domain appears missing.

Missing capabilities:

- Terminal as an operational entity with dispatch/accounting/compliance boundaries.
- Yard and dock/door inventory.
- Parking spots.
- Gate in/out events.
- Trailer/container/chassis yard movements.
- Yard checks.
- Drop/hook visibility.
- Dwell tracking by yard/dock.

Recommended direction:

- Start with terminal and yard modeling only where it supports dispatch, trailer visibility, detention, and dwell.
- Avoid building a full YMS too early unless target customers require it.

### 8. Pricing, Contracts, Tariffs, and Rating Depth

Trenova has accessorial charges, formula templates, rating details, base rate, distance overrides, and shipment commercial calculation. That is useful, but enterprise TMS products usually need more explicit commercial contract structure.

Missing or shallow capabilities:

- Customer contracts/rate agreements.
- Tariffs.
- Lane rates.
- Fuel surcharge programs.
- Effective dates, expiration, versioning, and approval.
- Routing guide rate commitments.
- Minimum charges, deficit rating, discounts, stop charges, commodity/hazmat modifiers, and multi-currency/tax policy depth.
- Rate audit and quote history.

Recommended direction:

- Keep formula templates for configurable calculations.
- Add explicit rate agreement/tariff/fuel surcharge entities so the system can explain why a rate applied.
- Version commercial agreements and preserve rating snapshots for auditability.

### 9. Integration Hub Maturity

The current EDI work is meaningful, especially X12 rendering, templates, partner profiles, validation, and transfer workflows. The missing layer is a broader integration hub that handles external enterprise connectivity.

Missing or shallow capabilities:

- AS2 transport.
- SFTP batch drops.
- VAN support or VAN-facing adapters.
- Webhook subscriptions and delivery attempts.
- API partner credentials/scopes separate from internal API keys.
- Idempotency keys and replay-safe inbound processing.
- Partner-specific DLQs and replay tooling.
- 997 functional acknowledgments.
- 214 shipment status flows.
- 210 freight invoice flows.
- External ERP/accounting connectors.
- Load board, visibility provider, mapping/routing, payment/factoring, and carrier qualification integrations.
- Partner simulators and certification suites.

Recommended direction:

- Define canonical integration events and commands first.
- Build adapters around those contracts.
- Add durable inbound/outbound message state, retry policy, idempotency, replay, and operator tooling.

### 10. Event Contracts and Outbox

The repo has CDC through GTC and Temporal workflows, but CDC is not the same as a business event contract.

Missing capabilities:

- Transactional outbox for material domain events.
- Versioned event schemas.
- Event idempotency and deduplication conventions.
- Event replay policy.
- Consumer contract tests.
- Explicit event taxonomy for shipment lifecycle, tender, assignment, billing, settlement, document, compliance, and integration events.

Recommended direction:

- Keep GTC for search/read projection CDC.
- Add an application-level outbox for domain events that represent business meaning.
- Use Temporal for workflows and the outbox for durable integration/event publication.
- Do not add Kafka/RabbitMQ until outbox semantics and event volume justify it.

### 11. Analytics and Warehouse Posture

Analytics service and client surfaces exist, but the enterprise report expects deeper operational intelligence.

Missing or shallow capabilities:

- Warehouse export path.
- Historical fact tables or stable BI models.
- Lane profitability.
- Cost-per-mile.
- Dwell and detention analytics.
- On-time pickup/delivery.
- Tender acceptance/rejection performance.
- Carrier scorecards.
- Driver/equipment utilization.
- Billing variance and adjustment analytics.
- Customer scorecards.
- Production-like replay datasets for analytics validation.

Recommended direction:

- Start with operational read models in Postgres/Meilisearch where needed.
- Add a warehouse/export pattern once the core event model stabilizes.
- Make analytics traceable back to operational and financial source records.

### 12. Production Deployment and Operations

Local infrastructure is rich, and CI exists for TMS, client, releases, and simulator tests. Production posture needs more explicit assets.

Missing or shallow capabilities:

- Kubernetes/Helm or equivalent deployment manifests.
- Separate API and worker deployment topology.
- GTC deployment and monitoring runbook.
- Temporal production persistence and namespace strategy.
- Secrets management and key rotation.
- Production config profiles with hardened defaults.
- Backup/restore drills.
- DR runbooks and RPO/RTO targets.
- Migration rollout strategy.
- Canary/blue-green rollout controls.
- Centralized log shipping and alert definitions.
- SLOs and dashboards.

Recommended direction:

- Document and implement a production reference architecture before heavy customer onboarding.
- Treat API, workers, GTC, Temporal, Postgres, Redis, search, and object storage as separately observable components.

### 13. Tenant, Legal Entity, and Enterprise Controls

The codebase has organization and business-unit scoping, RBAC, route policy, data retention, and platform/control-plane concepts. Enterprise customers will push this further.

Missing or shallow capabilities:

- Legal entity as a distinct accounting/compliance boundary.
- Tenant-safe background job guarantees.
- Per-tenant rate limits and quotas.
- Tenant-level data export.
- Tenant-level data deletion/retention workflows.
- Customer-configurable audit retention.
- Sandbox/UAT tenants.
- Tenant migration tooling.
- Tenant-aware search/CDC verification.

Recommended direction:

- Formalize tenant, organization, business unit, and future legal entity semantics.
- Add tenant isolation tests across API, repositories, jobs, CDC/search, documents, and audit.

### 14. Portals and External User Workflows

The internal app is broad, but enterprise TMS products usually include external collaboration surfaces.

Missing capabilities:

- Customer portal for tender/order visibility, documents, invoices, status, and disputes.
- Carrier portal for tender acceptance, document upload, status updates, invoice/pay visibility.
- Driver mobile workflow or integration-backed driver app strategy.
- Public API onboarding and developer docs.
- Webhook subscription management.

Recommended direction:

- Build portals only after the canonical order/load/tender/document/invoice models are stable.
- Start with narrow portal workflows: customer shipment visibility and carrier tender response/document upload.

## Priority Roadmap

### P0: Foundation Fixes Before More Breadth

These should happen before adding more large feature areas:

- Define the canonical operating model: `Order`, `Load/Shipment`, `Trip/MovementPlan`, `Tender`, and `FinancialArtifact`.
- Add transactional outbox and versioned domain events for material lifecycle changes.
- Formalize production deployment topology and runbooks.
- Add tenant isolation tests across API, jobs, documents, search, and audit.
- Clarify how GTC CDC, Temporal workflows, realtime invalidation, and future integration events each fit.

### P1: Enterprise TMS Core Depth

These close the largest market gaps:

- Carrier procurement and tendering.
- Settlements/payables.
- Compliance evidence workflows.
- Maintenance and inspection workflows.
- Contract/rate/tariff/fuel surcharge depth.
- External EDI/API/webhook transport and replay tooling.

### P2: Operational Expansion

These deepen the product after the quote-to-cash spine is solid:

- Claims/OS&D.
- Terminal/yard/dock/gate operations.
- Carrier and customer portals.
- Warehouse/BI export.
- Carrier scorecards and lane profitability.
- Load board and visibility provider integrations.

### P3: Optimization and Intelligence

These should come after the data model and event history are reliable:

- Dispatch optimization.
- Routing guide automation.
- Predictive ETA and exception prediction.
- Anomaly detection for billing/settlement.
- Pricing intelligence.
- Advanced network optimization.

## Architecture Recommendation

Do not split into microservices now. Trenova is at the stage where domain clarity is more valuable than deployment fragmentation.

Recommended architecture path:

1. Keep the modular monolith.
2. Strengthen bounded contexts inside the repo.
3. Introduce explicit event contracts and an outbox.
4. Use Temporal for long-running workflows.
5. Keep PostgreSQL as the source of truth.
6. Use GTC/Meilisearch for search and read projection support.
7. Add a broker only when partner fanout, event throughput, or cross-service boundaries demand it.
8. Split services only when one domain has a clear operational scaling, ownership, deployment, or compliance reason.

## Acceptance Criteria for "Enterprise Ready"

Trenova should not be considered enterprise-ready until it can demonstrate:

- A quote-to-cash flow with order intake, load planning, assignment, status events, POD, billing queue, invoice, payment, and adjustment.
- A tender-to-carrier flow with retries, acceptance/rejection, audit, and external transport.
- A payables flow with settlement/payable generation from operational evidence.
- Compliance evidence workflows with expiration queues and audit views.
- Maintenance workflows that can block dispatch when equipment is not eligible.
- Integration replay and idempotency for inbound/outbound partner messages.
- Tenant isolation across API, database access, documents, search, jobs, audit, and realtime events.
- Production deployment documentation with backups, restores, monitoring, alerts, and DR targets.
- Scenario and contract tests for duplicate, out-of-order, delayed, corrected, and replayed business events.

## Bottom Line

Trenova is set up well architecturally for a serious TMS. The foundation is stronger than the average early-stage product: domain modules, Temporal, documents, EDI, audit, billing, accounting, RBAC, observability, and CDC/search are already present.

The missing work is to mature it from a strong shipment/billing/document platform into a complete enterprise trucking operating system. The most important adjustments are domain-model clarity, carrier/tender depth, settlement/payables, compliance evidence, maintenance, claims, external integration hardening, and production operations.

The strategic recommendation is to dominate the asset-carrier quote-to-cash spine first, then add brokerage, portals, claims, yard, analytics, and optimization on top of a stable domain/event foundation.
