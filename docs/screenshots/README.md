# Trenova — Feature Screenshots

Captured from the running application against the seeded development dataset.
Every record shown is seed data (`SEED-*` identifiers, fictional carriers and drivers) —
no production or customer information appears in these images.

- **Resolution:** 1600x1000 CSS px at 2x device pixel ratio (3200x2000 actual)
- **Light theme:** [`light/`](light) — every page below
- **Dark theme:** [`dark/`](dark) — the 12 marquee screens

Regenerate with the Playwright script described in [Regenerating](#regenerating).

## Contents

- [Overview](#overview)
- [Shipment management](#shipment-management)
- [Dispatch](#dispatch)
- [Billing & revenue](#billing--revenue)
- [Detention](#detention)
- [Accounting](#accounting)
- [Payroll & settlements](#payroll--settlements)
- [Carrier settlements](#carrier-settlements)
- [Human resources](#human-resources)
- [Equipment](#equipment)
- [EDI](#edi)
- [Reporting](#reporting)
- [Administration](#administration)

## Overview

### Home dashboard

Personalised landing page: what needs attention, quick actions, favourites and recent activity.

![Home dashboard](light/home-dashboard.png)

<details><summary>Dark theme</summary>

![Home dashboard (dark)](dark/home-dashboard.png)

</details>

### Sign in

Password sign-in with optional SSO providers, per-tenant branding.

![Sign in](light/sign-in.png)

## Shipment management

### Shipments

Operations command centre — revenue, on-time, at-risk and unassigned KPIs over a live shipment table.

![Shipments](light/shipments.png)

<details><summary>Dark theme</summary>

![Shipments (dark)](dark/shipments.png)

</details>

### Orders

Customer orders awaiting conversion into shipments.

![Orders](light/orders.png)

### Recurring shipments

Templates that spawn shipments on a schedule.

![Recurring shipments](light/recurring-shipments.png)

### Service failures

Missed-service log with reason codes and accountability.

![Service failures](light/service-failures.png)

## Dispatch

### Dispatch console

Cover open moves against available capacity without opening a shipment.

![Dispatch console](light/dispatch-console.png)

<details><summary>Dark theme</summary>

![Dispatch console (dark)](dark/dispatch-console.png)

</details>

### Locations

Facilities, terminals and customer sites.

![Locations](light/locations.png)

### Carriers

Partner carrier roster with authority and insurance state.

![Carriers](light/carriers.png)

### Routing guides

Ranked carrier fallbacks per lane.

![Routing guides](light/routing-guides.png)

## Billing & revenue

### Billing queue

Shipments ready to invoice, grouped by exception.

![Billing queue](light/billing-queue.png)

<details><summary>Dark theme</summary>

![Billing queue (dark)](dark/billing-queue.png)

</details>

### Invoices

Issued customer invoices and their lifecycle.

![Invoices](light/invoices.png)

### Invoice approvals

Invoices held for human review before release.

![Invoice approvals](light/invoice-approvals.png)

### Reconciliation exceptions

Invoices that failed automated reconciliation.

![Reconciliation exceptions](light/reconciliation-exceptions.png)

### Rate agreements

The contracts that decide what a shipment costs, and the lanes each one prices.

![Rate agreements](light/rate-agreements.png)

### Rate matrices

Lane/zone rate grids attached to an agreement.

![Rate matrices](light/rate-matrices.png)

### Rate zones

Geographic zone definitions used by matrices.

![Rate zones](light/rate-zones.png)

### Formula templates

Formula Studio — every rate is a versioned, reviewable formula template.

![Formula templates](light/formula-templates.png)

<details><summary>Dark theme</summary>

![Formula templates (dark)](dark/formula-templates.png)

</details>

### Accessorial charges

Reusable surcharge definitions.

![Accessorial charges](light/accessorial-charges.png)

### Customers

Customer master with billing profiles and document requirements.

![Customers](light/customers.png)

### Document types

Document taxonomy driving packet rules.

![Document types](light/document-types.png)

### Fuel management

Fuel surcharge programmes and index tracking.

![Fuel management](light/fuel-management.png)

## Detention

### Detention desk

Live free-time clocks, notice deadlines and accruing detention across every driver on a dock.

![Detention desk](light/detention-desk.png)

<details><summary>Dark theme</summary>

![Detention desk (dark)](dark/detention-desk.png)

</details>

### Detention intelligence

Where detention is being lost, by facility and customer.

![Detention intelligence](light/detention-intelligence.png)

### Detention policies

Free-time and billing rules per customer or facility.

![Detention policies](light/detention-policies.png)

## Accounting

### Accounting dashboard

Receivables health, collections and cash-flow at a glance.

![Accounting dashboard](light/accounting-dashboard.png)

<details><summary>Dark theme</summary>

![Accounting dashboard (dark)](dark/accounting-dashboard.png)

</details>

### Trial balance

Period trial balance.

![Trial balance](light/trial-balance.png)

### Income statement

P&L for the selected fiscal period.

![Income statement](light/income-statement.png)

### Balance sheet

Assets, liabilities and equity.

![Balance sheet](light/balance-sheet.png)

### AR aging

Receivables bucketed by age.

![AR aging](light/ar-aging.png)

### AR open items

Every unsettled receivable line.

![AR open items](light/ar-open-items.png)

### Customer payments

Payment capture and application.

![Customer payments](light/customer-payments.png)

### Bank receipts

Imported bank receipts awaiting matching.

![Bank receipts](light/bank-reconciliation.png)

### Reconciliation work queue

Unmatched receipts routed for human resolution.

![Reconciliation work queue](light/reconciliation-work-queue.png)

### Manual journals

Hand-entered journal entries.

![Manual journals](light/manual-journals.png)

### Journal reversals

Reversal entries and their originals.

![Journal reversals](light/journal-reversals.png)

## Payroll & settlements

### Settlement workspace

Run a whole pay period from one screen — queue, transfers, deductions, settlements.

![Settlement workspace](light/payroll-workspace.png)

<details><summary>Dark theme</summary>

![Settlement workspace (dark)](dark/payroll-workspace.png)

</details>

### Driver settlements

Settlement history per driver.

![Driver settlements](light/driver-settlements.png)

### Settlement disputes

Driver-raised pay disputes and their resolution.

![Settlement disputes](light/settlement-disputes.png)

### Driver expenses

Submitted expenses awaiting reimbursement.

![Driver expenses](light/driver-expenses.png)

### Pay profiles

Per-mile, percentage and hourly pay structures.

![Pay profiles](light/pay-profiles.png)

### Pay codes

Earning and deduction code definitions.

![Pay codes](light/pay-codes.png)

### Escrow accounts

Driver escrow balances under 49 CFR 376.12(k).

![Escrow accounts](light/escrow-accounts.png)

## Carrier settlements

### Carrier settlement workspace

Pay partner carriers from one screen.

![Carrier settlement workspace](light/carrier-settlement-workspace.png)

### Carrier settlements

Settlement records per carrier.

![Carrier settlements](light/carrier-settlements.png)

### Carrier invoice matching

Match inbound carrier invoices to cost events.

![Carrier invoice matching](light/carrier-invoice-matching.png)

## Human resources

### Workers

Worker roster with compliance, training and PTO roll-ups.

![Workers](light/workers.png)

<details><summary>Dark theme</summary>

![Workers (dark)](dark/workers.png)

</details>

### Scheduling

Rota derived on read, with swap requests and coverage.

![Scheduling](light/hr-scheduling.png)

<details><summary>Dark theme</summary>

![Scheduling (dark)](dark/hr-scheduling.png)

</details>

### Time & attendance

Timesheets whose totals freeze at submit.

![Time & attendance](light/time-attendance.png)

### Fleet safety

Safety events, CSA BASICs and the accountability ladder.

![Fleet safety](light/fleet-safety.png)

### OSHA log

Recordable injury log.

![OSHA log](light/osha-log.png)

### Random drug & alcohol testing

DOT random pools with reproducible HMAC draws.

![Random drug & alcohol testing](light/random-drug-testing.png)

### Org structure

Job positions, teams and delegation.

![Org structure](light/org-structure.png)

### Training courses

Required-course matrix per role.

![Training courses](light/training-courses.png)

### Policies

Policy documents and signature tracking, pinned to a version label.

![Policies](light/policies.png)

### Benefits

Benefit plans and enrolment.

![Benefits](light/benefits.png)

### PTO policies

Accrual policies, tenure tiers and caps.

![PTO policies](light/pto-policies.png)

### My team

Manager view scoped to direct reports.

![My team](light/my-team.png)

### Checklist templates

Onboarding and offboarding checklists.

![Checklist templates](light/worker-checklist-templates.png)

### Credential types

Licence and endorsement definitions driving compliance.

![Credential types](light/worker-credential-types.png)

## Equipment

### Tractors

Power unit roster.

![Tractors](light/tractors.png)

### Trailers

Trailer roster.

![Trailers](light/trailers.png)

### Equipment types

Equipment classification.

![Equipment types](light/equipment-types.png)

## EDI

### EDI overview

Trading partner health and transaction volume.

![EDI overview](light/edi-overview.png)

### EDI partners

Trading partner configuration.

![EDI partners](light/edi-partners.png)

### EDI mapping designer

Visual mapper between EDI segments and domain fields.

![EDI mapping designer](light/edi-designer.png)

<details><summary>Dark theme</summary>

![EDI mapping designer (dark)](dark/edi-designer.png)

</details>

### EDI messages

Generated and received transaction sets.

![EDI messages](light/edi-messages.png)

### Inbound transfers

Inbound file transfer log.

![Inbound transfers](light/edi-transfers-inbound.png)

## Reporting

### Reports

Report catalogue and saved dashboards.

![Reports](light/reports.png)

<details><summary>Dark theme</summary>

![Reports (dark)](dark/reports.png)

</details>

### Report explorer

Ad-hoc exploration over report definitions.

![Report explorer](light/report-explorer.png)

### Report builder

Build and publish a report definition.

![Report builder](light/report-builder.png)

### Report runs

Execution history and outputs.

![Report runs](light/report-runs.png)

## Administration

### Roles

Role definitions and the permissions each grants.

![Roles](light/admin-roles.png)

### Users

User accounts and role assignment.

![Users](light/admin-users.png)

### Audit logs

Immutable record of who changed what.

![Audit logs](light/audit-logs.png)

### GraphQL explorer

Built-in explorer over the typed GraphQL API.

![GraphQL explorer](light/graphql-explorer.png)

### API keys

Programmatic access credentials.

![API keys](light/api-keys.png)

### Integrations

Third-party integrations (Samsara, Google, PC*Miler).

![Integrations](light/integrations.png)

### Custom fields

Tenant-defined fields on core entities.

![Custom fields](light/custom-fields.png)

### Document intelligence

Automated document parsing rules.

![Document intelligence](light/document-intelligence.png)

### Shipment controls

Org-wide shipment behaviour switches.

![Shipment controls](light/shipment-controls.png)

### Billing controls

Org-wide billing behaviour switches.

![Billing controls](light/billing-controls.png)

### Organization settings

Tenant profile and preferences.

![Organization settings](light/organization-settings.png)

### Table change alerts

Row-level change notifications.

![Table change alerts](light/table-change-alerts.png)

### Hold reasons

Reason codes for placing a shipment on hold.

![Hold reasons](light/hold-reasons.png)

## Regenerating

1. Start the infrastructure and API: `cd services/tms && task docker-up && task run-watch`
2. Start the web client: `cd client && pnpm --filter @trenova/web dev`
3. Sign in as an org administrator. Note that the post-login **role activation gate**
   must be cleared before any route will render — an un-activated session returns
   `403 Missing <resource>:Read` on every page.
4. Drive the capture with Playwright at a 2x device pixel ratio.

If a page renders an error boundary mentioning a failed dynamic import, the Vite
dep cache is stale — stop the dev server, delete `client/node_modules/.vite`, and
restart it before re-capturing.
