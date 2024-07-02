package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type GLAccountSubType string

const (
	GLAccountSubTypeCurrentAsset      = GLAccountSubType("CurrentAsset")
	GLAccountSubTypeFixedAsset        = GLAccountSubType("FixedAsset")
	GLAccountSubTypeOtherAsset        = GLAccountSubType("OtherAsset")
	GLAccountSubTypeCurrentLiability  = GLAccountSubType("CurrentLiability")
	GLAccountSubTypeLongTermLiability = GLAccountSubType("LongTermLiability")
	GLAccountSubTypeEquity            = GLAccountSubType("Equity")
	GLAccountSubTypeRevenue           = GLAccountSubType("Revenue")
	GLAccountSubTypeExpense           = GLAccountSubType("Expense")
	GLAccountSubTypeOtherIncome       = GLAccountSubType("OtherIncome")
	GLAccountSubTypeOtherExpense      = GLAccountSubType("OtherExpense")
	GlAccountSubTypeCostOfGoodsSold   = GLAccountSubType("CostOfGoodsSold")
)

func (s GLAccountSubType) String() string {
	return string(s)
}

func (GLAccountSubType) Values() []string {
	return []string{
		"CurrentAsset",
		"FixedAsset",
		"OtherAsset",
		"CurrentLiability",
		"LongTermLiability",
		"Equity",
		"Revenue",
		"Expense",
		"OtherIncome",
		"OtherExpense",
		"CostOfGoodsSold",
	}
}

func (s GLAccountSubType) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *GLAccountSubType) Scan(value any) error {
	if value == nil {
		return errors.New("GLAccountSubType: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*s = GLAccountSubType(v)
	case []byte:
		*s = GLAccountSubType(string(v))
	default:
		return fmt.Errorf("GLAccountSubType: cannot scan type %T into GLAccountSubType", value)
	}
	return nil
}
