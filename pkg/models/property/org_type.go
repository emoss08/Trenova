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

type OrganizationType string

const (
	OrganizationTypeAsset     = OrganizationType("Asset")
	OrganizationTypeBrokerage = OrganizationType("Brokerage")
	OrganizationTypeBoth      = OrganizationType("Both")
)

func (o OrganizationType) String() string {
	return string(o)
}

func (OrganizationType) Values() []string {
	return []string{"Asset", "Brokerage", "Both"}
}

func (o OrganizationType) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *OrganizationType) Scan(value any) error {
	if value == nil {
		return errors.New("organizationtype: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = OrganizationType(v)
	case []byte:
		*o = OrganizationType(string(v))
	default:
		return fmt.Errorf("organizationtype: cannot can type %T into OrganizationType", value)
	}
	return nil
}
