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

package validator

import "fmt"

// DBValidationError is an error that occurs during database validation
type DBValidationError struct {
	Field   string
	Message string
}

func (e DBValidationError) Error() string {
	return fmt.Sprintf("field %s: %s", e.Field, e.Message)
}

// BusinessLogicError is an error that occurs during business logic validation
type BusinessLogicError struct {
	Message string
}

func (e BusinessLogicError) Error() string {
	return e.Message
}

// MultiValidationError is a collection of DBValidationErrors
type MultiValidationError struct {
	Errors []DBValidationError
}

func (m MultiValidationError) Error() string {
	return fmt.Sprintf("multiple validation errors occurred (%d)", len(m.Errors))
}
