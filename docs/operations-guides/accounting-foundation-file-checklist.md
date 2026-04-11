# Accounting Foundation File-by-File Checklist

This checklist covers the remaining five implementation steps after the initial posting-engine slice:

1. Money minor-unit standardization
2. Manual journal workflow
3. Reversal workflow
4. Invoice-posted accounting
5. GL balance reads and period-close accounting guards

The checklist is repo-specific to `services/tms` and calls out DRY/SOLID cleanup that should happen while implementing each step.

## Cross-Cutting DRY / SOLID Fixes

- [ ] Remove direct journal table writes from operational services.
  Files:
  - `services/tms/internal/core/services/invoiceadjustmentservice/execution_helpers.go`
  - `services/tms/internal/core/services/invoiceadjustmentservice/service.go`
  Action:
  - Replace service-local `journalEntryRecord` / `journalEntryLineRecord` structs with accounting domain objects, repository ports, and a shared posting service.

- [ ] Centralize monetary conversion rules.
  Files:
  - `services/tms/internal/core/services/invoiceadjustmentservice/execution_helpers.go`
  - `services/tms/internal/core/domain/invoice/invoice.go`
  - `services/tms/internal/core/domain/invoiceadjustment/invoiceadjustment.go`
  Action:
  - Eliminate service-local `moneyToCents()` helpers.
  - Use one shared money package for rounding and minor-unit conversion.

- [ ] Centralize posting-period policy resolution.
  Files:
  - `services/tms/internal/core/services/invoiceservice/validator.go`
  - `services/tms/internal/core/services/invoiceadjustmentservice/service.go`
  - `services/tms/internal/core/services/fiscalperiodservice/validator.go`
  Action:
  - Move locked-period, closed-period, next-open-period, and approval-policy checks into a shared accounting policy service.

- [ ] Fix fiscal-period lifecycle semantics before manual journals and reversals.
  Files:
  - `services/tms/internal/core/domain/fiscalperiod/enums.go`
  - `services/tms/internal/core/domain/fiscalperiod/fiscalperiod.go`
  - `services/tms/internal/core/services/fiscalperiodservice/service.go`
  Action:
  - Align runtime transitions with documented lifecycle `Inactive -> Open -> Locked -> Closed -> PermanentlyClosed`.
  - Remove the hard-coded `PeriodNumber <= 12` assumption.

## Step 1: Money Minor-Unit Standardization

### New files

- [ ] Add `shared/money/amount.go`
  Action:
  - Define canonical accounting money type using `CurrencyCode` and `Minor int64`.

- [ ] Add `shared/money/convert.go`
  Action:
  - Add decimal-to-minor conversion with banker rounding.
  - Add helpers for sums, comparisons, and zero values.

- [ ] Add `shared/money/amount_test.go`
  Action:
  - Cover rounding edge cases and negative-value behavior.

### Existing files to modify

- [ ] Modify `services/tms/internal/core/domain/invoice/invoice.go`
  Action:
  - Add authoritative minor-unit fields:
    - `SubtotalAmountMinor`
    - `OtherAmountMinor`
    - `TotalAmountMinor`
    - `AppliedAmountMinor`
  - Add `AmountMinor` to invoice lines.
  - Keep current decimal fields only as transitional compatibility fields.

- [ ] Modify `services/tms/internal/core/domain/invoiceadjustment/invoiceadjustment.go`
  Action:
  - Add minor-unit fields for adjustment totals and line amounts.
  - Keep decimal fields only for compatibility with current UI/domain payloads during migration.

- [ ] Modify `services/tms/internal/infrastructure/postgres/migrations/20260408120000_add_invoices.tx.up.sql`
  Action:
  - Do not edit historical migration.
  - Use this file only as reference for current schema when adding a new forward migration.

- [ ] Add new migration under `services/tms/internal/infrastructure/postgres/migrations/`
  Suggested scope:
  - `invoices`: add `*_amount_minor BIGINT`
  - `invoice_lines`: add `amount_minor BIGINT`
  - `invoice_adjustments`: add `*_amount_minor BIGINT`
  - `invoice_adjustment_lines`: add `credit_amount_minor`, `rebill_amount_minor`
  - backfill from existing decimal columns using shared conversion rule expressed in SQL

- [ ] Modify `services/tms/internal/infrastructure/postgres/repositories/invoicerepository/invoice.go`
  Action:
  - Persist and read the new minor-unit fields.
  - Ensure create/update logic keeps decimal and minor fields in sync during transition.

- [ ] Modify `services/tms/internal/infrastructure/postgres/repositories/invoiceadjustmentrepository/invoiceadjustment.go`
  Action:
  - Persist and read adjustment minor-unit totals and lines.

- [ ] Modify `services/tms/internal/core/services/invoiceadjustmentservice/execution_helpers.go`
  Action:
  - Remove `moneyToCents()`.
  - Use shared money package and authoritative minor-unit totals.

- [ ] Modify `services/tms/internal/core/services/invoiceservice/validator.go`
  Action:
  - Add consistency validation so posted invoices cannot drift between decimal and minor-unit totals.

- [ ] Modify `services/tms/internal/core/services/invoiceadjustmentservice/service.go`
  Action:
  - Build adjustment previews from authoritative posted monetary totals, not ad hoc conversion at posting time.

### Tests

- [ ] Modify `services/tms/internal/core/services/invoiceadjustmentservice/service_integration_test.go`
  Action:
  - Assert journal output uses minor-unit authoritative values.

- [ ] Add repository tests where needed:
  - `services/tms/internal/infrastructure/postgres/repositories/invoicerepository/`
  - `services/tms/internal/infrastructure/postgres/repositories/invoiceadjustmentrepository/`

## Step 2: Manual Journal Workflow

### New domain files

- [ ] Add `services/tms/internal/core/domain/manualjournal/manualjournal.go`
- [ ] Add `services/tms/internal/core/domain/manualjournal/enums.go`
- [ ] Add `services/tms/internal/core/domain/manualjournal/line.go`
  Action:
  - Model request lifecycle separately from posted journal entries.

### New repository port files

- [ ] Add `services/tms/internal/core/ports/repositories/manualjournal.go`
- [ ] Add `services/tms/internal/core/ports/services/manualjournal.go`

### New service files

- [ ] Add `services/tms/internal/core/services/manualjournalservice/service.go`
- [ ] Add `services/tms/internal/core/services/manualjournalservice/validator.go`
- [ ] Add `services/tms/internal/core/services/manualjournalservice/testing.go`
  Action:
  - Implement `CreateDraft`, `UpdateDraft`, `Submit`, `Approve`, `Reject`, `Post`, `Cancel`.
  - Enforce `AccountingControl.ManualJournalEntryPolicy`.
  - Enforce `GLAccount.AllowManualJE`.

### New postgres repository files

- [ ] Add `services/tms/internal/infrastructure/postgres/repositories/manualjournalrepository/manualjournal.go`
- [ ] Add `services/tms/internal/infrastructure/postgres/repositories/manualjournalrepository/manualjournal_test.go`

### New migration files

- [ ] Add a migration for:
  - `manual_journal_requests`
  - `manual_journal_request_lines`

### Sequence and numbering files to modify

- [ ] Modify `services/tms/internal/core/domain/tenant/enums.go`
  Action:
  - Add `SequenceTypeJournalEntry`
  - Add `SequenceTypeJournalBatch`
  - Add `SequenceTypeManualJournalRequest`

- [ ] Modify `services/tms/internal/core/domain/tenant/sequenceconfig.go`
  Action:
  - Support new sequence types.

- [ ] Modify `services/tms/pkg/seqgen/generator.go`
  Action:
  - Add generation helpers for journal batch, journal entry, and manual journal request numbering.

- [ ] Modify `services/tms/pkg/seqgen/format_provider.go`
  Action:
  - Add default formats for the new accounting sequence types.

- [ ] Modify `services/tms/internal/infrastructure/postgres/repositories/sequenceconfigrepository/sequenceconfig.go`
  Action:
  - Seed defaults for the new sequence types.
  - Refactor hard-coded defaults so the file is open for extension.

- [ ] Modify `services/tms/internal/core/services/sequenceconfigservice/validator.go`
  Action:
  - Stop validating against exactly four sequence configs.
  - Use a required-sequence registry instead of an inline fixed map.

- [ ] Modify `services/tms/internal/infrastructure/database/seeds/base/01_adminaccount.go`
  Action:
  - Seed default sequence configs for new accounting sequence types if needed.

### Wiring and API files

- [ ] Modify `services/tms/internal/bootstrap/modules/repositories.go`
  Action:
  - Register `manualjournalrepository.New`.

- [ ] Modify `services/tms/internal/bootstrap/modules/api/services.go`
  Action:
  - Register `manualjournalservice.New`.

- [ ] Modify `services/tms/internal/bootstrap/modules/validators.go`
  Action:
  - Register `manualjournalservice.NewValidator`.

- [ ] Add `services/tms/internal/api/handlers/manualjournalhandler/handler.go`
  Action:
  - Add draft/submit/approve/post endpoints.

- [ ] Modify `services/tms/internal/bootstrap/modules/api/handlers.go`
  Action:
  - Register `manualjournalhandler.New`.

- [ ] Modify `services/tms/internal/api/router.go`
  Action:
  - Wire manual journal routes into the protected API group.

### Tests

- [ ] Add `services/tms/internal/core/services/manualjournalservice/service_test.go`
- [ ] Add `services/tms/internal/core/services/manualjournalservice/service_integration_test.go`

## Step 3: Reversal Workflow

### New domain files

- [ ] Add `services/tms/internal/core/domain/journalreversal/journalreversal.go`
- [ ] Add `services/tms/internal/core/domain/journalreversal/enums.go`

### New repository port files

- [ ] Add `services/tms/internal/core/ports/repositories/journalreversal.go`
- [ ] Add `services/tms/internal/core/ports/services/journalreversal.go`

### New service files

- [ ] Add `services/tms/internal/core/services/journalreversalservice/service.go`
- [ ] Add `services/tms/internal/core/services/journalreversalservice/validator.go`
  Action:
  - Implement reversal request, approval, rejection, posting, and cancellation.
  - Enforce one active reversal per original entry.

### New repository files

- [ ] Add `services/tms/internal/infrastructure/postgres/repositories/journalreversalrepository/journalreversal.go`
- [ ] Add `services/tms/internal/infrastructure/postgres/repositories/journalreversalrepository/journalreversal_test.go`

### New migration files

- [ ] Add a migration for `journal_reversals`.
  Action:
  - Add explicit linkage to original entry, reversal entry, status, accounting period, actors, and batch.

### Existing files to modify

- [ ] Modify `services/tms/internal/core/domain/tenant/enums.go`
  Action:
  - Ensure `JournalReversalPolicyType` is fully consumed by runtime services.

- [ ] Modify `services/tms/internal/core/services/invoiceadjustmentservice/service.go`
  Action:
  - Remove any temptation to reuse `KindFullReversal` as the accounting reversal workflow.
  - Keep invoice-adjustment reversal semantics separate from journal-entry reversal semantics.

- [ ] Modify `services/tms/internal/core/domain/invoiceadjustment/invoiceadjustment.go`
  Action:
  - Clarify `KindFullReversal` as an invoice-adjustment artifact, not a general ledger reversal.

### Wiring and API files

- [ ] Modify `services/tms/internal/bootstrap/modules/repositories.go`
  Action:
  - Register `journalreversalrepository.New`.

- [ ] Modify `services/tms/internal/bootstrap/modules/api/services.go`
  Action:
  - Register `journalreversalservice.New`.

- [ ] Modify `services/tms/internal/bootstrap/modules/validators.go`
  Action:
  - Register `journalreversalservice.NewValidator`.

- [ ] Add `services/tms/internal/api/handlers/journalreversalhandler/handler.go`

- [ ] Modify `services/tms/internal/bootstrap/modules/api/handlers.go`
  Action:
  - Register the new reversal handler.

- [ ] Modify `services/tms/internal/api/router.go`
  Action:
  - Add reversal routes.

### Tests

- [ ] Add `services/tms/internal/core/services/journalreversalservice/service_test.go`
- [ ] Add `services/tms/internal/core/services/journalreversalservice/service_integration_test.go`

## Step 4: Invoice-Posted Accounting

### Posting engine files to modify

- [ ] Modify or add accounting posting service files introduced in the initial posting slice.
  Expected files:
  - `services/tms/internal/core/services/postingservice/service.go`
  - `services/tms/internal/core/services/postingservice/rules/invoice_posted.go`
  - `services/tms/internal/core/services/postingservice/validators/*.go`
  Action:
  - Add `InvoicePosted` source-event handler.
  - Map invoice bill types to AR/revenue posting entries.

### Existing operational files to modify

- [ ] Modify `services/tms/internal/core/services/invoiceservice/service.go`
  Action:
  - Capture a `journal_source` during `Post()`.
  - If accounting automation is enabled for `InvoicePosted`, call the posting service in the same transaction.
  - Keep invoice, shipment, billing queue, and accounting source state transactionally aligned.

- [ ] Modify `services/tms/internal/core/services/invoiceservice/validator.go`
  Action:
  - Replace service-local period-policy logic with shared accounting policy service.

- [ ] Modify `services/tms/internal/core/domain/tenant/accountingcontrol.go`
  Action:
  - No schema change required unless additional source events are needed.
  - Ensure current defaults are sufficient for invoice-posted automation.

- [ ] Modify `services/tms/internal/core/services/accountingcontrolservice/validator.go`
  Action:
  - Make sure automatic posting validation includes runtime-required accounts for invoice posting.

### Existing repository files to modify

- [ ] Modify `services/tms/internal/infrastructure/postgres/repositories/invoicerepository/invoice.go`
  Action:
  - Expose any additional fields needed for invoice posting snapshots and authoritative minor totals.

### Tests

- [ ] Modify `services/tms/internal/core/services/invoiceservice/service_test.go`
  Action:
  - Add cases for accounting source capture and auto-post policy behavior.

- [ ] Add `services/tms/internal/core/services/invoiceservice/service_integration_test.go`
  Action:
  - Verify invoice posting creates the expected source, batch, journal entry, lines, and balance updates.

## Step 5: GL Balance Reads and Period-Close Accounting Guards

### New domain and repository files

- [ ] Add `services/tms/internal/core/domain/glbalance/glbalance.go`
- [ ] Add `services/tms/internal/core/ports/repositories/glbalance.go`
- [ ] Add `services/tms/internal/infrastructure/postgres/repositories/glbalancerepository/glbalance.go`
- [ ] Add `services/tms/internal/infrastructure/postgres/repositories/glbalancerepository/glbalance_test.go`

### New service files

- [ ] Add `services/tms/internal/core/services/glbalanceservice/service.go`
- [ ] Add `services/tms/internal/core/services/glbalanceservice/queries.go`
  Action:
  - Implement trial-balance and account-balance-by-period reads against `gl_account_balances_by_period`.

### Existing files to modify

- [ ] Modify `services/tms/internal/core/domain/glaccount/glaccount.go`
  Action:
  - Clarify that `CurrentBalance`, `DebitBalance`, and `CreditBalance` are derived projections or deprecate them from CRUD semantics.

- [ ] Modify `services/tms/internal/core/services/glaccountservice/rules.go`
  Action:
  - Stop treating mutable master-data balance fields as authoritative.
  - Move deactivation checks to engine-owned balance projections.

- [ ] Modify `services/tms/internal/core/services/fiscalperiodservice/validator.go`
  Action:
  - Block close when there are:
    - failed accounting sources in the period
    - pending approved/manual journals not posted
    - pending reversals
  - Keep invoice reconciliation validation, but do not make it the only financial-close check.

- [ ] Modify `services/tms/internal/core/services/fiscalperiodservice/service.go`
  Action:
  - Align lock/close transitions with documented accounting lifecycle.
  - Use shared accounting close validator.

- [ ] Modify `services/tms/internal/core/domain/fiscalperiod/fiscalperiod.go`
  Action:
  - Remove the 12-period hard stop so adjusting periods can exist.

- [ ] Modify `services/tms/internal/core/domain/fiscalperiod/enums.go`
  Action:
  - Keep comments and lifecycle behavior aligned with implementation.

### Wiring and API files

- [ ] Modify `services/tms/internal/bootstrap/modules/repositories.go`
  Action:
  - Register `glbalancerepository.New`.

- [ ] Modify `services/tms/internal/bootstrap/modules/api/services.go`
  Action:
  - Register `glbalanceservice.New`.

- [ ] Add `services/tms/internal/api/handlers/glbalancehandler/handler.go`
  Action:
  - Add endpoints for trial balance and account balance detail.

- [ ] Modify `services/tms/internal/bootstrap/modules/api/handlers.go`
  Action:
  - Register `glbalancehandler.New`.

- [ ] Modify `services/tms/internal/api/router.go`
  Action:
  - Add GL balance routes.

### Tests

- [ ] Add `services/tms/internal/core/services/glbalanceservice/service_test.go`
- [ ] Add `services/tms/internal/core/services/fiscalperiodservice/service_integration_test.go`
  Action:
  - Add close-blocker tests for failed accounting sources, pending manual journals, and reversals.

## Accounting Engine Files To Expect If Not Already Added

If the initial posting slice has not yet created these files, add them before continuing with Steps 2-5:

- [ ] `services/tms/internal/core/domain/journalbatch/journalbatch.go`
- [ ] `services/tms/internal/core/domain/journalentry/journalentry.go`
- [ ] `services/tms/internal/core/domain/journalsource/journalsource.go`
- [ ] `services/tms/internal/core/domain/journalrule/journalrule.go`
- [ ] `services/tms/internal/core/ports/repositories/journalbatch.go`
- [ ] `services/tms/internal/core/ports/repositories/journalentry.go`
- [ ] `services/tms/internal/core/ports/repositories/journalsource.go`
- [ ] `services/tms/internal/core/ports/repositories/journalrule.go`
- [ ] `services/tms/internal/core/services/postingservice/service.go`
- [ ] `services/tms/internal/core/services/accountingpolicyservice/service.go`
- [ ] `services/tms/internal/infrastructure/postgres/repositories/journalbatchrepository/journalbatch.go`
- [ ] `services/tms/internal/infrastructure/postgres/repositories/journalentryrepository/journalentry.go`
- [ ] `services/tms/internal/infrastructure/postgres/repositories/journalsourcerepository/journalsource.go`
- [ ] `services/tms/internal/infrastructure/postgres/repositories/journalrulerepository/journalrule.go`

## Recommended Implementation Order

- [ ] Step 1: money minor-unit standardization
- [ ] Step 2a: sequence config refactor for accounting sequence types
- [ ] Step 2b: manual journal workflow
- [ ] Step 3: reversal workflow
- [ ] Step 4: invoice-posted accounting
- [ ] Step 5: GL balance reads and accounting-aware close validation

## Definition of Done Guardrails

- [ ] No operational service writes journal tables directly.
- [ ] No service-local monetary conversion helpers remain.
- [ ] Posted journal entries and lines are immutable.
- [ ] Every posted invoice/manual journal/reversal has source-to-entry traceability.
- [ ] Period locking behavior is consistent across invoices, manual journals, reversals, and period close.
- [ ] New sequence types are registry-driven, not hard-coded to a fixed count.
