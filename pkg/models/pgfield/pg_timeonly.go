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
