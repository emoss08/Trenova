// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
