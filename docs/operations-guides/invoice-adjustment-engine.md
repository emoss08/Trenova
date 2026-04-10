# Invoice Adjustment Engine

This document describes the runtime invoice credit / rebill engine introduced for `services/tms`.

## Runtime-consumed controls

The engine consumes these `InvoiceAdjustmentControl` fields at runtime:

- `partiallyPaidInvoiceAdjustmentPolicy`
- `paidInvoiceAdjustmentPolicy`
- `disputedInvoiceAdjustmentPolicy`
- `adjustmentAccountingDatePolicy`
- `closedPeriodAdjustmentPolicy`
- `adjustmentReasonRequirement`
- `adjustmentAttachmentRequirement`
- `standardAdjustmentApprovalPolicy`
- `standardAdjustmentApprovalThreshold`
- `writeOffApprovalPolicy`
- `writeOffApprovalThreshold`
- `rerateVarianceTolerancePercent`
- `replacementInvoiceReviewPolicy`
- `customerCreditBalancePolicy`
- `overCreditPolicy`
- `supersededInvoiceVisibilityPolicy`

The engine also consumes related finance controls:

- `AccountingControl.lockedPeriodPostingPolicy`
- `AccountingControl.closedPeriodPostingPolicy`
- `AccountingControl.reconciliationMode`
- `AccountingControl.reconciliationToleranceAmount`
- `AccountingControl.notifyOnReconciliationException`
- `AccountingControl.defaultWriteOffAccountId`
- `BillingControl.invoicePostingMode`

## Workflow semantics

- Posted invoices remain immutable.
- Credit memos are created as posted, immutable invoice artifacts.
- Replacement invoices are seeded through `billing_queue_items` and remain editable drafts until posted.
- Approval-required adjustments persist as `invoice_adjustments` in `PendingApproval` and do not mutate financial artifacts until approved.
- Approval re-runs policy and eligibility checks under lock before creating credit memo or rebill artifacts.
- Paid and partially paid invoices do not mutate cash application state. The engine creates reconciliation exceptions and explicit follow-up artifacts instead.

## Over-credit distinction

`OverCreditPolicy` never authorizes commercial over-credit.

- Credit beyond remaining eligible invoice-line scope is always blocked.
- Remaining eligible scope is reduced by previously executed partial credits tracked in `invoice_adjustment_lines`.
- `OverCreditPolicy` only applies when an otherwise valid commercial credit would create unapplied customer credit because the invoice settlement state makes the open balance smaller than the requested credit.
- If `CustomerCreditBalancePolicy` disallows unapplied credit outcomes, the engine blocks execution even when the requested credit is still within true line eligibility.

## Lineage model

- `invoice_correction_groups` stores the root original invoice and current active invoice pointer.
- `invoice_adjustments` links original invoice, credit memo, replacement invoice, and rebill queue item.
- `invoices` and `billing_queue_items` now carry correction metadata so invoice detail screens can traverse lineage directly.

## Snapshots

Every submitted adjustment stores immutable snapshots in `invoice_adjustment_snapshots`.

- `Submission` snapshots capture the original invoice and finance-sensitive source state before approval/execution.
- `Execution` snapshots capture the source state plus the created artifact linkage after execution.
