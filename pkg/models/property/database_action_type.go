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

type DatabaseAction string

const (
	DatabaseActionInsert = DatabaseAction("Insert")
	DatabaseActionUpdate = DatabaseAction("Update")
	DatabaseActionDelete = DatabaseAction("Delete")
	DatabaseActionAll    = DatabaseAction("All")
)

func (o DatabaseAction) String() string {
	return string(o)
}

func (DatabaseAction) Values() []string {
	return []string{"Insert", "Update", "Delete", "All"}
}

func (o DatabaseAction) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *DatabaseAction) Scan(value any) error {
	if value == nil {
		return errors.New("databaseaction: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = DatabaseAction(v)
	case []byte:
		*o = DatabaseAction(string(v))
	default:
		return fmt.Errorf("databaseaction: cannot can type %T into DatabaseAction", value)
	}
	return nil
}
