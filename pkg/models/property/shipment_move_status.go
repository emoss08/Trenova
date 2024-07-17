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

type ShipmentMoveStatus string

const (
	ShipmentMoveStatusNew        = ShipmentMoveStatus("New")
	ShipmentMoveStatusInProgress = ShipmentMoveStatus("InProgress")
	ShipmentMoveStatusCompleted  = ShipmentMoveStatus("Completed")
	ShipmentMoveStatusVoided     = ShipmentMoveStatus("Voided")
)

func (o ShipmentMoveStatus) String() string {
	return string(o)
}

func (ShipmentMoveStatus) Values() []string {
	return []string{"New", "InProgress", "Completed", "Voided"}
}

func (o ShipmentMoveStatus) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *ShipmentMoveStatus) Scan(value any) error {
	if value == nil {
		return errors.New("ShipmentMoveStatus: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = ShipmentMoveStatus(v)
	case []byte:
		*o = ShipmentMoveStatus(string(v))
	default:
		return fmt.Errorf("ShipmentMoveStatusType: cannot can type %T into ShipmentMoveStatusType", value)
	}
	return nil
}
