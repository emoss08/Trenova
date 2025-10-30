package accounting

import "errors"

type Category string

const (
	CategoryAsset         = Category("Asset")
	CategoryLiability     = Category("Liability")
	CategoryEquity        = Category("Equity")
	CategoryRevenue       = Category("Revenue")
	CategoryCostOfRevenue = Category("CostOfRevenue")
	CategoryExpense       = Category("Expense")
)

func (c Category) String() string {
	return string(c)
}

func (c Category) IsValid() bool {
	switch c {
	case CategoryAsset, CategoryLiability, CategoryEquity,
		CategoryRevenue, CategoryCostOfRevenue, CategoryExpense:
		return true
	}
	return false
}

func (c Category) GetDescription() string {
	switch c {
	case CategoryAsset:
		return "Resources owned by the company"
	case CategoryLiability:
		return "Obligations owed by the company"
	case CategoryEquity:
		return "Owner's stake in the company"
	case CategoryRevenue:
		return "Income from operations"
	case CategoryCostOfRevenue:
		return "Direct costs of providing service"
	case CategoryExpense:
		return "Operating expenses"
	default:
		return "Unknown category"
	}
}

type FiscalYearStatus string

const (
	FiscalYearStatusDraft  = FiscalYearStatus("Draft")
	FiscalYearStatusOpen   = FiscalYearStatus("Open")
	FiscalYearStatusClosed = FiscalYearStatus("Closed")
	FiscalYearStatusLocked = FiscalYearStatus("Locked")
)

func (s FiscalYearStatus) String() string {
	return string(s)
}

func (s FiscalYearStatus) IsValid() bool {
	switch s {
	case FiscalYearStatusDraft, FiscalYearStatusOpen,
		FiscalYearStatusClosed, FiscalYearStatusLocked:
		return true
	}
	return false
}

func (s FiscalYearStatus) GetDescription() string {
	switch s {
	case FiscalYearStatusDraft:
		return "Year is being set up, not yet active"
	case FiscalYearStatusOpen:
		return "Year is active and accepting transactions"
	case FiscalYearStatusClosed:
		return "Year-end closing complete, only adjusting entries allowed"
	case FiscalYearStatusLocked:
		return "Year is locked, no transactions allowed"
	default:
		return "Unknown status"
	}
}

func FiscalYearStatusFromString(status string) (FiscalYearStatus, error) {
	switch status {
	case "Draft":
		return FiscalYearStatusDraft, nil
	case "Open":
		return FiscalYearStatusOpen, nil
	case "Closed":
		return FiscalYearStatusClosed, nil
	case "Locked":
		return FiscalYearStatusLocked, nil
	default:
		return "", errors.New("invalid fiscal year status")
	}
}
