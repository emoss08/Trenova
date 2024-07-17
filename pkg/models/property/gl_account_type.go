// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
