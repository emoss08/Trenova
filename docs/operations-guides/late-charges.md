# Late charges

Late charges are assessed by a nightly run, not typed by hand. The run looks at every posted invoice or debit memo that is still open past its due date plus the customer's grace period, charges the customer's rate on the open balance once per thirty-day period, and raises one debit memo per customer carrying a line per invoice and period.

## Controls

| Where | Setting | Effect |
|---|---|---|
| Billing control | `lateChargeAssessmentMode` | `Disabled` (default) skips the tenant. `Preview` computes the run and writes nothing, so the Late Charges page shows what `Automatic` would raise. `Automatic` raises the memos. |
| Billing control | `lateChargeMinimumAmount` | A customer whose total for the run is below this is skipped. |
| Billing control | `invoicePostingMode` | When posting is automatic the memo posts in the same run; otherwise it waits in the invoice workspace as a draft. |
| Customer billing profile | `applyLateCharges` | Opts the customer in. |
| Customer billing profile | `lateChargeRate` | Percent of the open balance charged per period. |
| Customer billing profile | `gracePeriodDays` | Days after the due date before the first period starts. |

## Rules

- The overdue clock starts at `dueDate + gracePeriodDays`. Period 1 runs from that instant for thirty days, period 2 for the next thirty, and so on. A period is assessable the moment it begins.
- Each (invoice, period) is assessed at most once. `late_charge_assessments` has a unique index on it, so two runs on the same day, or two workers in the same run, cannot double-charge; the second insert is a no-op and raises no memo.
- The charge is `openBalance × rate ÷ 100`, rounded half to even to the minor unit, on the balance open at run time. A rate change affects only periods assessed afterwards.
- Invoices that are disputed (flag or open dispute case), fully paid, voided, or are themselves late-charge memos are never candidates.
- The memo carries `memoKind = LateCharge`, the run key in its memo text, and one line per period: `Late charge on INV-123, period 2 (2026-05-01 to 2026-05-30), 1.5% of 1,200.00`.
- If raising the memo fails, the run deletes that customer's assessment rows for the run key so the next night tries again.

## Running it

- Schedule: `late-charge-assessment`, cron `30 2 * * *` on the billing task queue, overlap policy skip. One activity per tenant with assessment enabled.
- On demand: the Late Charges page under Accounts Receivable previews (`lateChargePreview`) and runs (`assessLateCharges`) for chosen customers and an as-of date. Running on demand while the mode is `Disabled` is refused; previewing is always allowed.
- Each memo raised is audited on the invoice and produces a `late_charges_assessed` notification.

## Where to look

- Domain rules: `internal/core/domain/latecharge/period.go`.
- Candidate query and idempotent insert: `internal/infrastructure/postgres/repositories/latechargerepository`.
- Run orchestration: `internal/core/services/latechargeservice`.
- Temporal workflow and schedule: `internal/core/temporaljobs/billingjobs/latecharge.go`, `schedules.go`.
