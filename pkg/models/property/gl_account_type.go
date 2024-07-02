package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type GLAccountType string

const (
	GLAccountTypeAsset     = GLAccountType("Asset")
	GLAccountTypeLiability = GLAccountType("Liability")
	GLAccountTypeEquity    = GLAccountType("Equity")
	GLAccountTypeRevenue   = GLAccountType("Revenue")
	GLAccountTypeExpense   = GLAccountType("Expense")
)

func (s GLAccountType) String() string {
	return string(s)
}

func (GLAccountType) Values() []string {
	return []string{"Asset", "Liability", "Equity", "Revenue", "Expense"}
}

func (s GLAccountType) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *GLAccountType) Scan(value any) error {
	if value == nil {
		return errors.New("GLAccountType: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*s = GLAccountType(v)
	case []byte:
		*s = GLAccountType(string(v))
	default:
		return fmt.Errorf("GLAccountType: cannot scan type %T into GLAccountType", value)
	}
	return nil
}

func (s GLAccountType) List() []GLAccountType {
	return []GLAccountType{
		GLAccountTypeAsset,
		GLAccountTypeLiability,
		GLAccountTypeEquity,
		GLAccountTypeRevenue,
		GLAccountTypeExpense,
	}
}
