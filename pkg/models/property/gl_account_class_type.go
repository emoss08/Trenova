package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type GLAccountClassificationType string

const (
	GLAccountClassificationTypeBank               = GLAccountClassificationType("Bank")
	GLAccountClassificationTypeCash               = GLAccountClassificationType("Cash")
	GLAccountClassificationTypeAccountsReceivable = GLAccountClassificationType("AccountsReceivable")
	GLAccountClassificationTypeAccountsPayable    = GLAccountClassificationType("AccountsPayable")
	GLAccountClassificationTypeInventory          = GLAccountClassificationType("Inventory")
	GLAccountClassificationTypePrepaidExpenses    = GLAccountClassificationType("PrepaidExpenses")
	GLAccountClassificationTypeAccruedExpenses    = GLAccountClassificationType("AccruedExpenses")
	GLAccountClassificationTypeOtherCurrentAsset  = GLAccountClassificationType("OtherCurrentAsset")
	GLAccountClassificationTypeFixedAsset         = GLAccountClassificationType("FixedAsset")
)

func (s GLAccountClassificationType) String() string {
	return string(s)
}

func (GLAccountClassificationType) Values() []string {
	return []string{
		"Bank",
		"Cash",
		"AccountsReceivable",
		"AccountsPayable",
		"Inventory",
		"PrepaidExpenses",
		"AccruedExpenses",
		"OtherCurrentAsset",
		"FixedAsset",
	}
}

func (s GLAccountClassificationType) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *GLAccountClassificationType) Scan(value any) error {
	if value == nil {
		return errors.New("GLAccountClassificationType: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*s = GLAccountClassificationType(v)
	case []byte:
		*s = GLAccountClassificationType(string(v))
	default:
		return fmt.Errorf("GLAccountClassificationType: cannot scan type %T into GLAccountClassificationType", value)
	}
	return nil
}
