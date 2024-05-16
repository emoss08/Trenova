package util

import (
	"context"
	"fmt"
	"strings"

	gen "github.com/emoss08/trenova/internal/ent"
)

// ValidateUniqueness checks whether a given value already exists in a specified table
// under certain conditions and returns an error if the value is not unique.
//
// This function dynamically constructs a SQL query to check for the existence of a record
// in the specified table that matches the given conditions. The conditions are provided as
// a map where keys represent column names and values represent the corresponding values to
// be checked. The constructed query uses parameterized arguments to prevent SQL injection.
//
// Args:
//
//	ctx (context.Context): The context for controlling the query lifetime.
//	client (*gen.Client): The database client used for executing the query.
//	table (string): The name of the table to check for the uniqueness of the value.
//	attr (string): The name of the attribute being checked for uniqueness, used for error reporting.
//	conditions (map[string]string): A map of column-value pairs representing the conditions
//	  to be included in the WHERE clause of the SQL query.
//	excludeID (string): The ID of the current record to be excluded from the uniqueness check.
//
// Returns:
//
//	error: Returns a NewValidationError if a record matching the conditions exists,
//	  otherwise returns nil if the value is unique. Returns an error if any database
//	  operation fails during the execution.
//
// Example:
//
//	conditions := map[string]string{
//	    "name": "exampleName",
//	    "organization_id": "12345",
//	}
//	err := ValidateUniqueness(ctx, client, "location_categories", "name", conditions, "current_record_id")
//	if err != nil {
//	    return err
//	}
//
// The function ensures that the value provided in the 'conditions' map does not already
// exist in the specified table, excluding the current record. If a matching record is found,
// it returns a validation error specifying the 'attr' that caused the error.
func ValidateUniqueness(ctx context.Context, client *gen.Client, table, attr string, conditions map[string]string, excludeID string) error {
	// Construct the WHERE clause dynamically based on the conditions map.
	whereClauses := []string{}
	args := []interface{}{}
	i := 1
	for column, value := range conditions {
		if column == "organization_id" {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", column, i))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("LOWER(%s) = LOWER($%d)", column, i))
		}
		args = append(args, value)
		i++
	}

	// Add the exclusion of the current record's ID.
	if excludeID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("id != $%d", i))
		args = append(args, excludeID)
	}

	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE %s)`, table, strings.Join(whereClauses, " AND "))

	// Check if the value already exists in the table.
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var exists bool
	for rows.Next() {
		err = rows.Scan(&exists)
		if err != nil {
			return err
		}
	}

	// If the value already exists, return an error.
	if exists {
		return NewValidationError("Value already exists in the database. Please try again.", "invalid", attr)
	}

	return nil
}
