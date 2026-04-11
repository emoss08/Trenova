# Accounting Foundation: Done vs Deferred

This document captures the current implementation boundary for the accounting foundation in `services/tms`.

It is intended to answer two questions:

1. What is implemented and supported now?
2. What is intentionally deferred to a later phase?

## Done Now

### Posting Backbone

- Shared accounting posting path for journal persistence
- Journal batches
- Journal entries
- Journal entry lines
- Source-to-ledger traceability via `journal_sources`
- Source-to-entry linkage via `source_journal_links`
- Period-scoped balances via `gl_account_balances_by_period`
- Running GL account balance updates on `gl_accounts`

### Supported Posting Flows

- Manual journal requests through draft, submit, approve, reject, cancel, and post
- Invoice-posted accounting
- Credit memo and debit memo invoice-posted accounting
- Invoice write-off accounting
- Explicit journal reversal workflow with reversal-by-new-entry behavior

### Accounting Controls Enforced Now

- `JournalPostingMode`
- `AutoPostSourceEvents`
- `ManualJournalEntryPolicy`
- `RequireManualJEApproval`
- `JournalReversalPolicy`
- `LockedPeriodPostingPolicy`
- `ClosedPeriodPostingPolicy`
- `ReconciliationMode`
- `ReconciliationToleranceAmount`
- `RequireReconciliationToClose`
- `PeriodCloseMode` for manual close blocking
- `RequirePeriodCloseApproval` for manual close blocking
- `AccountingBasis` and `RevenueRecognitionPolicy` for invoice-post ledger eligibility
- Default AR / revenue / write-off account requirements used by supported posting flows

### Billing Controls Enforced Now

- `DefaultPaymentTerm`
- `ReadyToBillAssignmentMode`
- `BillingQueueTransferMode` in readiness evaluation
- `ShipmentBillingRequirementEnforcement`
- `RateValidationEnforcement`
- `BillingExceptionDisposition`
- `NotifyOnBillingExceptions`
- `RateVarianceTolerancePercent`
- `RateVarianceAutoResolutionMode`
- `InvoicePostingMode` for auto-post gating
- `InvoiceDraftCreationMode` for auto-post eligibility gating

### Close / Period Controls

- Fiscal close blocked by pending approved manual journals
- Fiscal close blocked by unposted accounting sources
- Fiscal close blocked by reconciliation rules when enabled
- Locked-period and closed-period posting behavior enforced in supported accounting flows

### Read Side / Reporting

- Trial balance read API
- Income statement read API
- Balance sheet read API
- Journal entry detail read API
- Source-to-journal drill-down read API

### Testing Done

- Unit tests for core accounting policy and service slices
- Integration tests for supported posting flows
- Integration tests for balances and traceability
- Integration tests for fiscal close blockers
- Integration tests for reporting repositories

## Deferred Intentionally

### Billing Automation Execution

These fields are validated, but the full scheduled/batch execution workflow is not yet implemented:

- `BillingQueueTransferSchedule`
- `BillingQueueTransferBatchSize`
- `AutoInvoiceBatchSize`
- `NotifyOnAutoInvoiceCreation`

Current state:

- The application can evaluate readiness and transfer intent.
- The application does not yet execute full schedule-driven transfer and invoice-draft automation from these settings.

### Multi-Currency / FX Runtime

These controls are not fully operationalized yet:

- `CurrencyMode` beyond currently supported single-currency accounting behavior
- `FunctionalCurrencyCode` beyond current validation/runtime checks
- `ExchangeRateDatePolicy`
- `ExchangeRateOverridePolicy`
- `RealizedFXGainAccountID`
- `RealizedFXLossAccountID`

Current state:

- Validation exists.
- Full exchange-rate selection, override workflow, and FX posting behavior are deferred.

### Billing Presentation Fields

These fields exist in `BillingControl`, but are not treated as implemented application behavior yet:

- `DefaultInvoiceTerms`
- `DefaultInvoiceFooter`
- `ShowDueDateOnInvoice`
- `ShowBalanceDueOnInvoice`

Current state:

- They are stored in the model.
- They are not yet wired into a dedicated invoice-rendering or document-generation surface.

### Broader Subledgers

Deferred to later phases:

- Customer payment posting and full AR subledger behavior
- Vendor bill posting and full AP subledger behavior
- Vendor payment posting
- Write-off and settlement flows beyond the current supported accounting foundation

### Approval Workflow Objects Beyond Blocking

Deferred:

- Dedicated fiscal period close approval request workflow objects
- Full accounting approval inbox / orchestration beyond current service-level enforcement

## Phase 2 Checklist: Customer Payments and AR Subledger

This checklist is the recommended next implementation slice after the accounting foundation.

### Scope

- [ ] Customer payment posting
- [ ] Invoice application / settlement
- [ ] Unapplied cash handling
- [ ] Overpayment handling
- [ ] Short-pay handling
- [ ] Payment reversal handling
- [ ] Cash-basis revenue recognition support through payment events

### Core Domain / Persistence

- [ ] Add `customer_payments` table and domain model
- [ ] Add `customer_payment_applications` table and domain model
- [ ] Add `customer_ledger_entries` or equivalent customer subledger domain model
- [ ] Add repository ports and postgres repositories for customer payments and applications
- [ ] Add source-to-ledger linkage for payment events through the shared posting path

### Posting Events

- [ ] Support `CustomerPaymentPosted`
- [ ] Support `CustomerPaymentApplied`
- [ ] Support `CustomerPaymentUnapplied`
- [ ] Support `CustomerShortPayRecognized`
- [ ] Support `CustomerPaymentReversed`

### Accounting Behavior

- [ ] Post cash receipt to cash / unapplied cash or AR as appropriate
- [ ] Post invoice application to reduce AR correctly
- [ ] Post overpayment and unapplied cash correctly
- [ ] Post short-pay / write-off interaction correctly
- [ ] Route cash-basis revenue recognition through payment events instead of invoice-post events

### Control Alignment

- [ ] Enforce `AccountingBasis` in payment posting behavior
- [ ] Enforce `RevenueRecognitionPolicy` in payment posting behavior
- [ ] Enforce `AutoPostSourceEvents` for customer payment events
- [ ] Enforce reconciliation controls for payment-linked settlement behavior where applicable

### Auditability

- [ ] Record payment source to journal linkage
- [ ] Record payment application lineage to invoice
- [ ] Record unapplied-to-applied transitions with audit history
- [ ] Record reversal lineage for payment corrections

### Reporting / Read Side

- [ ] Expose customer payment detail reads
- [ ] Expose invoice application detail reads
- [ ] Expose customer subledger / settlement drill-down reads
- [ ] Add AR-aging input model or derived read support

### Tests Required

- [ ] Unit tests for customer payment policy and posting decisions
- [ ] Integration test: full payment against single invoice
- [ ] Integration test: partial payment against single invoice
- [ ] Integration test: unapplied cash then later application
- [ ] Integration test: overpayment handling
- [ ] Integration test: short-pay handling
- [ ] Integration test: payment reversal
- [ ] Integration test: cash-basis revenue recognition on payment

## Supported Scope Boundary

The current accounting foundation should be treated as:

- backend-first
- source-traceable
- double-entry
- period-aware
- auditable for supported posting flows

It should not yet be treated as:

- a complete AR subledger
- a complete AP subledger
- a full multi-currency accounting engine
- a complete billing scheduler/orchestration system

## Recommended Next Phase

When work resumes beyond the current boundary, the recommended order is:

1. Customer payment posting and AR ledger behavior
2. Vendor bill posting and AP ledger behavior
3. Vendor payment posting
4. Multi-currency / FX runtime support
5. Billing schedule/batch automation execution
6. Optional close approval workflow objects
