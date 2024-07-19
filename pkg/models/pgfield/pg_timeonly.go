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

package pgfield

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
)

// TimeOnly wraps a time.Time to provide custom scanning and formatting.
type TimeOnly struct {
	Time time.Time
}

// Scan implements the Scanner interface.
func (t *TimeOnly) Scan(value any) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("unsupported type %T, expected string", value)
	}
	parsedTime, err := time.Parse("15:04:05", str) // PostgreSQL 'time' format
	if err != nil {
		return fmt.Errorf("parse time error: %w", err)
	}
	t.Time = parsedTime
	return nil
}

// MarshalJSON converts the TimeOnly object to JSON.
func (t TimeOnly) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return sonic.Marshal(nil)
	}
	return sonic.Marshal(t.Time.Format("15:04:05"))
}

// UnmarshalJSON converts JSON data to a TimeOnly object.
func (t *TimeOnly) UnmarshalJSON(data []byte) error {
	var str string
	if err := sonic.Unmarshal(data, &str); err != nil {
		return err
	}
	if str == "" {
		t.Time = time.Time{}
		return nil
	}
	parsedTime, err := time.Parse("15:04:05", str)
	if err != nil {
		return err
	}
	t.Time = parsedTime
	return nil
}

// Value implements the driver Valuer interface.
func (t TimeOnly) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time.Format("15:04:05"), nil // PostgreSQL 'time' format
}
