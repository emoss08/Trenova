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

type ShipmentRatingMethod string

const (
	ShipmentRatingMethodFlatRate         = ShipmentRatingMethod("FlatRate")
	ShipmentRatingMethodPerMile          = ShipmentRatingMethod("PerMile")
	ShipmentRatingMethodPerHundredWeight = ShipmentRatingMethod("PerHundredWeight")
	ShipmentRatingMethodPerStop          = ShipmentRatingMethod("PerStop")
	ShipmentRatingMethodPerPound         = ShipmentRatingMethod("PerPound")
	ShipmentRatingMethodOther            = ShipmentRatingMethod("Other")
)

func (o ShipmentRatingMethod) String() string {
	return string(o)
}

func (ShipmentRatingMethod) Values() []string {
	return []string{"New", "InProgress", "Completed", "Hold", "Billed", "Voided"}
}

func (o ShipmentRatingMethod) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *ShipmentRatingMethod) Scan(value any) error {
	if value == nil {
		return errors.New("ShipmentRatingMethod: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = ShipmentRatingMethod(v)
	case []byte:
		*o = ShipmentRatingMethod(string(v))
	default:
		return fmt.Errorf("ShipmentRatingMethodType: cannot can type %T into ShipmentRatingMethodType", value)
	}
	return nil
}
