package canned

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

const (
	categoryAccounting = "Accounting"
	categoryPayroll    = "Payroll"

	entityGLAccountBalance  = "gl_account_balance"
	entityJournalEntryLine  = "journal_entry_line"
	entityCustomerLedger    = "customer_ledger_entry"
	entityDriverSettlement  = "driver_settlement"
	glAccountEdge           = "glAccount"
	accountTypeEdge         = "accountType"
	fiscalPeriodEdge        = "fiscalPeriod"
	fiscalYearEdge          = "fiscalYear"
	journalEntryEdge        = "journalEntry"
	accountCodeField        = "accountCode"
	accountingDateField     = "accountingDate"
	settlementNumberField   = "settlementNumber"
	accountColumnID         = "account_code"
	accountNameColumnID     = "account_name"
	accountCategoryColumnID = "account_category"

	tagLedger      = "ledger"
	tagAccounting  = "accounting"
	tagSettlements = "settlements"
	tagPayroll     = "payroll"
)

// trialBalance reads the pre-aggregated period balances rather than the journal
// lines, which is what makes it cheap enough to run across a whole year. The
// entity has no id column — its grain is the account-period — so it carries no
// row count: a count here would count account-periods, not documents.
func trialBalance() *Entry {
	return &Entry{
		Key:     "trial-balance",
		Version: initialVersion,
		Name:    "Trial Balance",
		Description: "Debits, credits and net change per GL account for the current " +
			"fiscal year, grouped by account category",
		Category:      categoryAccounting,
		Tags:          []string{tagLedger, tagAccounting, "trial-balance"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityGLAccountBalance,
			Columns: []report.ColumnSpec{
				dimCol(accountColumnID, "Account Code", accountCodeField, glAccountEdge),
				dimCol(accountNameColumnID, "Account Name", nameField, glAccountEdge),
				dimCol(accountCategoryColumnID, "Category", "category",
					glAccountEdge, accountTypeEdge),
				dimCol("period", "Period", nameField, fiscalPeriodEdge),
				sumMinor("period_debit", "Debits", "periodDebitMinor"),
				sumMinor("period_credit", "Credits", "periodCreditMinor"),
				sumMinor("net_change", "Net Change", "netChangeMinor"),
			},
			Filters: andFilters(report.FieldFilter{
				Ref:      report.FieldRef{Path: []string{fiscalYearEdge}, Field: "isCurrent"},
				Operator: dbtype.OpEqual,
				Value:    true,
			}),
			Sort: []report.SortSpec{asc(accountColumnID)},
		},
	}
}

// revenueByGLAccount is the report the ledger exists to answer: what posted to
// each account, for which customer, in which month. It reads the line rather
// than the entry because the account and the customer both live on the line.
func revenueByGLAccount() *Entry {
	return &Entry{
		Key:     "revenue-by-gl-account",
		Version: initialVersion,
		Name:    "Posted Activity by GL Account",
		Description: "Monthly posted debits, credits and net movement per GL account " +
			"and customer",
		Category:      categoryAccounting,
		Tags:          []string{tagLedger, tagAccounting, tagRevenue},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityJournalEntryLine,
			Columns: []report.ColumnSpec{
				dimCol(accountColumnID, "Account Code", accountCodeField, glAccountEdge),
				dimCol(accountNameColumnID, "Account Name", nameField, glAccountEdge),
				dimCol(accountCategoryColumnID, "Category", "category",
					glAccountEdge, accountTypeEdge),
				dimCol(customerColumnID, customerLabel, nameField, customerEdge),
				sumMinor("debit_total", "Debits", "debitAmount"),
				sumMinor("credit_total", "Credits", "creditAmount"),
				sumMinor("net_total", "Net", "netAmount"),
			},
			Filters: andFilters(
				report.FieldFilter{
					Ref: report.FieldRef{
						Path:  []string{journalEntryEdge},
						Field: "isPosted",
					},
					Operator: dbtype.OpEqual,
					Value:    true,
				},
				windowFilter(accountingDateField, journalEntryEdge),
			),
			Sort:       []report.SortSpec{desc("net_total")},
			Parameters: []report.ParameterDef{windowParam(90)},
		},
	}
}

// arAgingByDocument complements the invoice-grain aging report: the subledger
// carries credits, payments and unapplied cash, none of which appear on an
// invoice, so this is the only view that reconciles to the AR control account.
func arAgingByDocument() *Entry {
	return &Entry{
		Key:     "ar-aging-by-document",
		Version: initialVersion,
		Name:    "AR Subledger by Document",
		Description: "Every document that moved a customer balance — invoices, credits, " +
			"payments and adjustments — with its signed amount",
		Category:      categoryAccounting,
		Tags:          []string{tagAR, tagAccounting, "collections"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityCustomerLedger,
			Columns: []report.ColumnSpec{
				dimCol(customerColumnID, customerLabel, nameField, customerEdge),
				dimCol(customerCodeColumnID, customerCodeLabel, codeField, customerEdge),
				dimCol("source_object", "Document Type", "sourceObjectType"),
				dimCol("document_number", "Document Number", "documentNumber"),
				dateDim("transaction_date", "Transaction Date", "transactionDate"),
				sumMinor("balance_movement", "Amount", "amountMinor"),
			},
			Filters:    andFilters(windowFilter("transactionDate")),
			Sort:       []report.SortSpec{desc("balance_movement")},
			Parameters: []report.ParameterDef{windowParam(180)},
		},
	}
}

func settlementRegister() *Entry {
	return &Entry{
		Key:     "settlement-register",
		Version: initialVersion,
		Name:    "Driver Settlement Register",
		Description: "Gross earnings, deductions and net pay per settlement, with the " +
			"miles and shipment count behind each one",
		Category:      categoryPayroll,
		Tags:          []string{tagSettlements, tagPayroll, tagDrivers},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityDriverSettlement,
			Columns: []report.ColumnSpec{
				dimCol("driver_first_name", "First Name", firstNameField, workerEdge),
				dimCol("driver_last_name", "Last Name", lastNameField, workerEdge),
				dimCol("settlement_number", "Settlement", settlementNumberField),
				dimCol("classification", "Classification", "classification"),
				dimCol(statusFieldKey, statusLabel, statusFieldKey),
				dateDim("pay_date", "Pay Date", "payDate"),
				sumMinor("gross_earnings", "Gross Earnings", "grossEarningsMinor"),
				sumMinor("reimbursements", "Reimbursements", "reimbursementsMinor"),
				sumMinor("deductions", "Deductions", "deductionsMinor"),
				sumMinor("net_pay", "Net Pay", "netPayMinor"),
				newMeasure(&measureSpec{
					id:      totalMilesColumnID,
					label:   totalMilesLabel,
					agg:     reportcatalog.AggSum,
					field:   "totalMiles",
					display: counted(),
				}),
				countMeasure("settlement_count", "Settlements"),
			},
			Filters:    andFilters(windowFilter("payDate")),
			Sort:       []report.SortSpec{desc("net_pay")},
			Parameters: []report.ParameterDef{windowParam(90)},
		},
	}
}
