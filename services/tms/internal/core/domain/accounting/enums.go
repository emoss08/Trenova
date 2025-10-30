package accounting

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
