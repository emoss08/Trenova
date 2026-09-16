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

## Supporting documents

Supporting documents are optional for every adjustment kind, including credit and rebill, full reversals, write-offs and voids. Nothing refuses an adjustment for having no documents attached. The organization-wide `adjustmentAttachmentRequirement` control and the per-customer `invoiceAdjustmentSupportingDocumentPolicy` billing profile setting were retired because they forced documentation on customers that have no documentation requirements.

When documents are attached, each one must exist in the tenant, must not be archived, and must belong to a shipment the invoice bills. The GraphQL field `CustomerBillingProfile.invoiceAdjustmentSupportingDocumentPolicy` remains for compatibility, is deprecated, and always returns `Optional`.

## Full reversal voids the original

A `FullReversal` adjustment retires the invoice it reverses. When the reversal executes, the engine:

- creates the posted credit memo as before, reversing the receivable and the customer ledger entry;
- marks the original invoice `Voided`, stamping `voided_at`, `voided_by_id`, `voided_by_adjustment_id`, the reason and the disposition. The invoice keeps its number and lines so the audit trail stays readable, and its open balance is zero, so aging and the collections worklist no longer carry it;
- releases the billing queue items behind it according to the disposition. `Rebill` puts each item back to `Approved` with a fresh number and clears its invoice link, and its shipments return to `ReadyToInvoice`. `DoNotRebill` cancels the items and settles the shipments as `Completed`. Order charges and charge allocations invoiced by the voided invoice are released in the same transaction.

The disposition and reason are recorded on the invoice when the void is requested (`voidInvoice`), so an approver sees them and the engine has them when a pending reversal executes. A reversal submitted directly through the adjustment API voids with `DoNotRebill` and the adjustment reason.

A `FullReversal` is refused while the invoice has any cash or credit memo applied. Unapply the payments and credit memo applications first, then void.

Drafts never reach the ledger and are voided in place by `voidInvoice`, releasing their queue items the same way without an adjustment.

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
