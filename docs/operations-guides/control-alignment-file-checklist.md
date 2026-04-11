# Control Alignment File Checklist

This checklist covers the control-enforcement work for this pass.

Scope included now:

1. Centralize `AccountingControl` runtime interpretation
2. Centralize `BillingControl` runtime interpretation
3. Enforce accounting recognition and posting policy in invoice posting flows
4. Enforce billing auto-post/manual-review policy in invoice runtime
5. Enforce manual vs scheduled/approval close behavior in fiscal periods
6. Add tests for the above

Scope intentionally left out for now:

- Full billing schedule/batch execution workflows
- Full multi-currency / FX runtime support
- Invoice presentation-only fields (`DefaultInvoiceTerms`, `DefaultInvoiceFooter`, `ShowDueDateOnInvoice`, `ShowBalanceDueOnInvoice`)

## Checklist

- [x] Add `services/tms/internal/core/services/accountingcontrolpolicyservice/service.go`
- [x] Add `services/tms/internal/core/services/accountingcontrolpolicyservice/service_test.go`
- [x] Add `services/tms/internal/core/services/billingcontrolpolicyservice/service.go`
- [x] Add `services/tms/internal/core/services/billingcontrolpolicyservice/service_test.go`
- [x] Wire policy services in `services/tms/internal/bootstrap/modules/api/services.go`

- [x] Update `services/tms/internal/core/services/invoiceservice/accounting_helpers.go`
  Implement `AccountingControl`-driven invoice ledger eligibility
- [x] Update `services/tms/internal/core/services/invoiceservice/service.go`
  Enforce `BillingControl.InvoicePostingMode` for auto-post paths
- [ ] Update `services/tms/internal/core/services/invoiceservice/validator.go`
  Keep invoice posting validation aligned to accounting control policy
- [x] Update `services/tms/internal/core/services/invoiceservice/service_test.go`
  Add accounting + billing control enforcement coverage
- [x] Update `services/tms/internal/core/services/invoiceservice/service_integration_test.go`
  Cover invoice-post recognition gating when needed

- [x] Update `services/tms/internal/core/services/fiscalperiodservice/service.go`
  Enforce `PeriodCloseMode` and `RequirePeriodCloseApproval`
- [x] Update `services/tms/internal/core/services/fiscalperiodservice/service_integration_test.go`
  Add control-enforcement tests for manual close restrictions

- [x] Update `services/tms/internal/core/services/accountingcontrolservice/validator.go`
  Strengthen control compatibility rules for supported runtime behavior
- [x] Update `services/tms/internal/core/services/accountingcontrolservice/validator_test.go`

- [x] Update `services/tms/internal/core/services/billingcontrolservice/validator_test.go`
  Add tests for runtime-supported billing combinations

- [x] Run targeted unit and integration tests for:
  - `accountingcontrolpolicyservice`
  - `billingcontrolpolicyservice`
  - `invoiceservice`
  - `fiscalperiodservice`
  - `accountingcontrolservice`
  - `billingcontrolservice`
