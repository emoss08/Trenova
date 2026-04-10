# Finance Control Implementation Matrix

This matrix is based on the current code in `services/tms` and `client/src`.

Status values:

- `Fully enforced at runtime`
- `Validated + persisted, but not yet consumed at runtime`
- `UI/API only`
- `Reserved for future credit/rebill engine`

Legend:

- `V` = validation
- `P` = persistence
- `C` = runtime consumption

Common persistence paths:

- `AccountingControl`: `internal/core/domain/tenant/accountingcontrol.go`, `internal/infrastructure/postgres/repositories/accountingcontrolrepository/accountingcontrol.go`, migrations `20260408180000_finalize_finance_controls.*`
- `BillingControl`: `internal/core/domain/tenant/billingcontrol.go`, `internal/infrastructure/postgres/repositories/billingcontrolrepository/billingcontrol.go`, migrations `20260408180000_finalize_finance_controls.*`
- `InvoiceAdjustmentControl`: `internal/core/domain/tenant/invoiceadjustmentcontrol.go`, `internal/infrastructure/postgres/repositories/invoiceadjustmentcontrolrepository/invoiceadjustmentcontrol.go`, migrations `20260408180000_finalize_finance_controls.*`, `20260408193000_finance_control_hardening.*`

## AccountingControl

| Field | Status | V | C | If not consumed, why not |
| --- | --- | --- | --- | --- |
| `id` | UI/API only | domain shape only | handlers/services/repositories for CRUD identity | Metadata only |
| `businessUnitId` | UI/API only | auth-context binding | handlers/services/repositories for tenancy | Metadata only |
| `organizationId` | UI/API only | auth-context binding | handlers/services/repositories for tenancy | Metadata only |
| `accountingBasis` | Validated + persisted, but not yet consumed at runtime | `accountingcontrolservice/validator.go#createAccountingBasisRule` | none | No journal posting engine consumes basis yet |
| `revenueRecognitionPolicy` | Validated + persisted, but not yet consumed at runtime | `accountingcontrolservice/validator.go#createAccountingBasisRule` | none | No revenue journal engine consumes it yet |
| `expenseRecognitionPolicy` | Validated + persisted, but not yet consumed at runtime | `accountingcontrolservice/validator.go#createAccountingBasisRule` | none | No expense journal engine consumes it yet |
| `journalPostingMode` | Validated + persisted, but not yet consumed at runtime | `accountingcontrolservice/validator.go#createJournalPostingRule` | none | Automatic journal creation is not implemented yet |
| `autoPostSourceEvents` | Validated + persisted, but not yet consumed at runtime | `accountingcontrolservice/validator.go#createJournalPostingRule` | none | Journal source-event automation is not implemented yet |
| `manualJournalEntryPolicy` | Validated + persisted, but not yet consumed at runtime | `accountingcontrolservice/validator.go#createManualJournalEntryRule` | none | Manual JE workflow is not implemented yet |
| `requireManualJEApproval` | Validated + persisted, but not yet consumed at runtime | `accountingcontrolservice/validator.go#createManualJournalEntryRule` | none | Manual JE approval workflow is not implemented yet |
| `journalReversalPolicy` | Validated + persisted, but not yet consumed at runtime | domain required + service validator context | none | Journal reversal workflow is not implemented yet |
| `periodCloseMode` | Fully enforced at runtime | `accountingcontrolservice/validator.go#createPeriodCloseRule` | `accountingcontrolrepository.ListWithScheduledPeriodClose`, `core/temporaljobs/fiscaljobs/activities.go` | |
| `requirePeriodCloseApproval` | Validated + persisted, but not yet consumed at runtime | `accountingcontrolservice/validator.go#createPeriodCloseRule` | none | No period-close approval workflow exists yet |
| `lockedPeriodPostingPolicy` | Fully enforced at runtime | domain required + service validator context | `invoiceservice/validator.go#validatePostingPeriodPolicy` | |
| `closedPeriodPostingPolicy` | Fully enforced at runtime | domain required + service validator context | `invoiceservice/validator.go#validatePostingPeriodPolicy` | |
| `requireReconciliationToClose` | Fully enforced at runtime | `accountingcontrolservice/validator.go#createReconciliationRule` | `fiscalperiodservice/validator.go#ValidateClose` | |
| `reconciliationMode` | Fully enforced at runtime | `accountingcontrolservice/validator.go#createReconciliationRule` | `invoiceservice/validator.go#validatePostingReconciliation`, `fiscalperiodservice/validator.go#ValidateClose` | |
| `reconciliationToleranceAmount` | Fully enforced at runtime | `accountingcontrolservice/validator.go#createReconciliationRule` | `invoiceservice/validator.go#validatePostingReconciliation`, `invoicerepository.CountPostedReconciliationDiscrepancies` | |
| `notifyOnReconciliationException` | Fully enforced at runtime | domain required + service validator context | `invoiceservice/service.go#notifyReconciliationWarning` | |
| `currencyMode` | Validated + persisted, but not yet consumed at runtime | `accountingcontrolservice/validator.go#createCurrencyRule` | none | Multi-currency accounting behavior is not implemented yet |
| `functionalCurrencyCode` | Validated + persisted, but not yet consumed at runtime | `AccountingControl.Validate`, `accountingcontrolservice/validator.go#createCurrencyRule` | none | Functional-currency posting is not implemented yet |
| `exchangeRateDatePolicy` | Validated + persisted, but not yet consumed at runtime | domain required + service validator context | none | FX rate selection engine is not implemented yet |
| `exchangeRateOverridePolicy` | Validated + persisted, but not yet consumed at runtime | `accountingcontrolservice/validator.go#createCurrencyRule` | none | FX override workflow is not implemented yet |
| `defaultRevenueAccountId` | Validated + persisted, but not yet consumed at runtime | GL ref checks + `createJournalPostingRule` | none | Waiting on automatic journal posting |
| `defaultExpenseAccountId` | Validated + persisted, but not yet consumed at runtime | GL ref checks + `createJournalPostingRule` | none | Waiting on automatic journal posting |
| `defaultArAccountId` | Validated + persisted, but not yet consumed at runtime | GL ref checks + `createJournalPostingRule` | none | Waiting on automatic journal posting |
| `defaultApAccountId` | Validated + persisted, but not yet consumed at runtime | GL ref checks + `createJournalPostingRule` | none | Waiting on automatic journal posting |
| `defaultTaxLiabilityAccountId` | Validated + persisted, but not yet consumed at runtime | GL ref checks | none | Tax posting engine is not implemented yet |
| `defaultWriteOffAccountId` | Reserved for future credit/rebill engine | GL ref checks | none | This account is intended for future write-off posting in adjustment workflows |
| `defaultRetainedEarningsAccountId` | Validated + persisted, but not yet consumed at runtime | GL ref checks | none | Retained earnings close posting is not implemented yet |
| `realizedFxGainAccountId` | Validated + persisted, but not yet consumed at runtime | GL ref checks + `createCurrencyRule` | none | FX realization posting is not implemented yet |
| `realizedFxLossAccountId` | Validated + persisted, but not yet consumed at runtime | GL ref checks + `createCurrencyRule` | none | FX realization posting is not implemented yet |
| `version` | UI/API only | optimistic concurrency | repositories update paths | Metadata only |
| `createdAt` | UI/API only | Bun hooks | repositories/JSON/API | Metadata only |
| `updatedAt` | UI/API only | Bun hooks | repositories/JSON/API | Metadata only |

## BillingControl

| Field | Status | V | C | If not consumed, why not |
| --- | --- | --- | --- | --- |
| `id` | UI/API only | domain shape only | handlers/services/repositories for CRUD identity | Metadata only |
| `businessUnitId` | UI/API only | auth-context binding | handlers/services/repositories for tenancy | Metadata only |
| `organizationId` | UI/API only | auth-context binding | handlers/services/repositories for tenancy | Metadata only |
| `defaultPaymentTerm` | Fully enforced at runtime | domain required + client schema | `invoiceservice/service.go#resolvePaymentTerm` | |
| `defaultInvoiceTerms` | Validated + persisted, but not yet consumed at runtime | client schema and form | none | Invoice presentation engine does not read this value yet |
| `defaultInvoiceFooter` | Validated + persisted, but not yet consumed at runtime | client schema and form | none | Invoice presentation engine does not read this value yet |
| `showDueDateOnInvoice` | Validated + persisted, but not yet consumed at runtime | client schema and form | none | Invoice rendering does not read this value yet |
| `showBalanceDueOnInvoice` | Validated + persisted, but not yet consumed at runtime | client schema and form | none | Invoice rendering does not read this value yet |
| `readyToBillAssignmentMode` | Fully enforced at runtime | `billingcontrolservice/validator.go#createTransferAutomationRule`, readiness policy resolution | `shipmentservice/billing_readiness.go` | |
| `billingQueueTransferMode` | Fully enforced at runtime | `billingcontrolservice/validator.go#createTransferAutomationRule`, readiness policy resolution | `shipmentservice/billing_readiness.go`, `shipmentservice/TransferToBilling` | |
| `billingQueueTransferSchedule` | Validated + persisted, but not yet consumed at runtime | `billingcontrolservice/validator.go#createTransferAutomationRule` | none | No scheduled transfer worker consumes this yet |
| `billingQueueTransferBatchSize` | Validated + persisted, but not yet consumed at runtime | `billingcontrolservice/validator.go#createTransferAutomationRule` | none | No scheduled transfer worker consumes this yet |
| `invoiceDraftCreationMode` | Validated + persisted, but not yet consumed at runtime | `billingcontrolservice/validator.go#createInvoiceAutomationRule` | none | Draft creation remains billing-queue approval driven today |
| `invoicePostingMode` | Fully enforced at runtime | `billingcontrolservice/validator.go#createInvoiceAutomationRule` | `invoiceservice/service.go#shouldAutoPost` | |
| `autoInvoiceBatchSize` | Validated + persisted, but not yet consumed at runtime | `billingcontrolservice/validator.go#createInvoiceAutomationRule` | none | No automatic draft-batch worker consumes this yet |
| `notifyOnAutoInvoiceCreation` | Validated + persisted, but not yet consumed at runtime | `billingcontrolservice/validator.go#createInvoiceAutomationRule` | none | Auto-invoice notification emission is not implemented yet |
| `shipmentBillingRequirementEnforcement` | Fully enforced at runtime | `billingcontrolservice/validator.go#createRequirementEnforcementRule` | `shipmentservice/billing_readiness.go`, `shipmentservice/validateBillingTransferPolicy`, status-change readiness validation | |
| `rateValidationEnforcement` | Fully enforced at runtime | `billingcontrolservice/validator.go#createRateValidationRule` | `shipmentservice/billing_readiness.go`, `shipmentservice/validateBillingTransferPolicy` | |
| `billingExceptionDisposition` | Fully enforced at runtime | `billingcontrolservice/validator.go#createRequirementEnforcementRule` | `shipmentservice/billing_readiness.go`, `shipmentservice/validateBillingTransferPolicy` | |
| `notifyOnBillingExceptions` | Fully enforced at runtime | domain required + client schema | `shipmentservice/billing_readiness.go#notifyBillingExceptions` during transfer attempts | |
| `rateVarianceTolerancePercent` | Fully enforced at runtime | `billingcontrolservice/validator.go#createRateValidationRule` | `shipmentservice/evaluateRateValidation` | |
| `rateVarianceAutoResolutionMode` | Fully enforced at runtime | `billingcontrolservice/validator.go#createRateValidationRule` | `shipmentservice/evaluateRateValidation` | Bypass suppresses review only; it does not change amounts |
| `version` | UI/API only | optimistic concurrency | repositories update paths | Metadata only |
| `createdAt` | UI/API only | Bun hooks | repositories/JSON/API | Metadata only |
| `updatedAt` | UI/API only | Bun hooks | repositories/JSON/API | Metadata only |

## InvoiceAdjustmentControl

| Field | Status | V | C | If not consumed, why not |
| --- | --- | --- | --- | --- |
| `id` | UI/API only | domain shape only | handlers/services/repositories for CRUD identity | Metadata only |
| `businessUnitId` | UI/API only | auth-context binding | handlers/services/repositories for tenancy | Metadata only |
| `organizationId` | UI/API only | auth-context binding | handlers/services/repositories for tenancy | Metadata only |
| `partiallyPaidInvoiceAdjustmentPolicy` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createEligibilityRule` | none | Requires the adjustment engine to evaluate payment state during credit/rebill |
| `paidInvoiceAdjustmentPolicy` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createEligibilityRule` | none | Requires the adjustment engine to evaluate payment state during credit/rebill |
| `disputedInvoiceAdjustmentPolicy` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createEligibilityRule` | none | Requires dispute-aware adjustment workflow |
| `adjustmentAccountingDatePolicy` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createEligibilityRule` | none | Requires adjustment posting workflow |
| `closedPeriodAdjustmentPolicy` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createEligibilityRule` | none | Requires adjustment posting workflow |
| `adjustmentReasonRequirement` | Reserved for future credit/rebill engine | domain required + client schema | none | Requires adjustment completion workflow |
| `adjustmentAttachmentRequirement` | Reserved for future credit/rebill engine | domain required + client schema | none | Requires adjustment completion workflow |
| `standardAdjustmentApprovalPolicy` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createApprovalThresholdRule` | none | Requires adjustment approval workflow |
| `standardAdjustmentApprovalThreshold` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createApprovalThresholdRule`, migration hardening backfill | none | Requires adjustment approval workflow |
| `writeOffApprovalPolicy` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createApprovalThresholdRule` | none | Requires write-off workflow |
| `writeOffApprovalThreshold` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createApprovalThresholdRule`, migration hardening backfill | none | Requires write-off workflow |
| `rerateVarianceTolerancePercent` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createApprovalThresholdRule` | none | Requires rerate/rebill comparison workflow |
| `replacementInvoiceReviewPolicy` | Reserved for future credit/rebill engine | domain required + client schema | none | Requires replacement-invoice generation workflow |
| `customerCreditBalancePolicy` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createCreditBalanceRule` | none | Requires adjustment settlement and customer-credit workflow |
| `overCreditPolicy` | Reserved for future credit/rebill engine | `invoiceadjustmentcontrolservice/validator.go#createCreditBalanceRule` | none | Requires adjustment settlement and customer-credit workflow |
| `supersededInvoiceVisibilityPolicy` | Reserved for future credit/rebill engine | domain required + client schema | none | Requires supersession-aware invoice presentation workflow |
| `version` | UI/API only | optimistic concurrency | repositories update paths | Metadata only |
| `createdAt` | UI/API only | Bun hooks | repositories/JSON/API | Metadata only |
| `updatedAt` | UI/API only | Bun hooks | repositories/JSON/API | Metadata only |

## Over-Credit Note

`OverCreditPolicy` does not authorize credit beyond true eligible commercial scope.

- Credit above the remaining eligible invoice line or item scope is always blocked as a hard integrity rule.
- `OverCreditPolicy` only governs whether an otherwise valid credit outcome may create unapplied customer credit because of payment state or settlement state.
