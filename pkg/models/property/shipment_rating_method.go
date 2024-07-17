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
