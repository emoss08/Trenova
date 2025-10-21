package variable

import (
	"strings"
	"testing"
)

func TestQueryValidator(t *testing.T) {
	validator := NewQueryValidator()

	tests := []struct {
		name    string
		query   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple select",
			query:   "SELECT name FROM customers WHERE id = :customerId",
			wantErr: false,
		},
		{
			name:    "valid select with join",
			query:   "SELECT c.name, b.name FROM customers c JOIN business_units b ON c.business_unit_id = b.id WHERE c.id = :customerId",
			wantErr: false,
		},
		{
			name:    "valid select with functions",
			query:   "SELECT UPPER(name), TO_CHAR(created_at, 'YYYY-MM-DD') FROM customers WHERE id = :customerId",
			wantErr: false,
		},
		{
			name:    "invalid - not a select",
			query:   "UPDATE customers SET name = 'test'",
			wantErr: true,
			errMsg:  "must be a select",
		},
		{
			name:    "invalid - contains semicolon",
			query:   "SELECT * FROM customers; DROP TABLE customers",
			wantErr: true,
			errMsg:  "semicolons",
		},
		{
			name:    "invalid - forbidden function",
			query:   "SELECT pg_sleep(10) FROM customers",
			wantErr: true,
			errMsg:  "forbidden function",
		},
		{
			name:    "invalid - unknown table",
			query:   "SELECT * FROM secret_table",
			wantErr: true,
			errMsg:  "not allowed",
		},
		{
			name:    "invalid - unknown parameter",
			query:   "SELECT * FROM customers WHERE id = :hackerId",
			wantErr: true,
			errMsg:  "not allowed",
		},
		{
			name:    "valid - subquery",
			query:   "SELECT name FROM customers WHERE id IN (SELECT customer_id FROM invoices WHERE organization_id = :orgId)",
			wantErr: false,
		},
		{
			name:    "valid - case expression",
			query:   "SELECT CASE WHEN is_active THEN 'Active' ELSE 'Inactive' END FROM customers WHERE id = :customerId",
			wantErr: false,
		},
		{
			name:    "invalid - contains comment",
			query:   "SELECT * FROM customers -- evil comment",
			wantErr: true,
			errMsg:  "comments",
		},
		{
			name:    "valid - complex query with aggregation",
			query:   "SELECT COUNT(*), MAX(created_at) FROM customers WHERE organization_id = :orgId",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errMsg)) {
					t.Errorf("Validate() error = %v, want error containing %s", err, tt.errMsg)
				}
			}
		})
	}
}

func TestFormatSQLValidator(t *testing.T) {
	validator := NewFormatSQLValidator()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid uppercase",
			sql:     "UPPER(:value)",
			wantErr: false,
		},
		{
			name:    "valid date format",
			sql:     "TO_CHAR(:value::date, 'Mon DD, YYYY')",
			wantErr: false,
		},
		{
			name:    "valid currency format",
			sql:     "TO_CHAR(:value::numeric, 'FM$999,999,999.00')",
			wantErr: false,
		},
		{
			name:    "valid boolean case",
			sql:     "CASE WHEN :value = 'true' THEN 'Active' ELSE 'Inactive' END",
			wantErr: false,
		},
		{
			name:    "valid complex expression",
			sql:     "CONCAT('INV-', LPAD(:value, 6, '0'), '-', TO_CHAR(NOW(), 'YYYY'))",
			wantErr: false,
		},
		{
			name:    "valid nested functions",
			sql:     "UPPER(TRIM(REPLACE(:value, '-', ' ')))",
			wantErr: false,
		},
		{
			name:    "invalid - no value placeholder",
			sql:     "UPPER('test')",
			wantErr: true,
			errMsg:  "must contain :value",
		},
		{
			name:    "invalid - contains SELECT",
			sql:     "SELECT UPPER(:value) FROM customers",
			wantErr: true,
			errMsg:  "cannot contain SELECT",
		},
		{
			name:    "invalid - contains semicolon",
			sql:     "UPPER(:value); DROP TABLE customers",
			wantErr: true,
			errMsg:  "semicolons",
		},
		{
			name:    "invalid - dangerous function",
			sql:     "pg_sleep(10)",
			wantErr: true,
			errMsg:  "dangerous function",
		},
		{
			name:    "invalid - contains FROM",
			sql:     "UPPER(:value) FROM customers",
			wantErr: true,
			errMsg:  "cannot contain FROM",
		},
		{
			name:    "invalid - contains comment",
			sql:     "UPPER(:value) -- comment",
			wantErr: true,
			errMsg:  "comments",
		},
		{
			name:    "invalid - unbalanced parentheses",
			sql:     "UPPER(:value))",
			wantErr: true,
			errMsg:  "unbalanced parentheses",
		},
		{
			name:    "invalid - subquery",
			sql:     "(SELECT name FROM customers WHERE id = :value)",
			wantErr: true,
			errMsg:  "subqueries",
		},
		{
			name:    "invalid - database function",
			sql:     "current_user()",
			wantErr: true,
			errMsg:  "dangerous function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errMsg)) {
					t.Errorf("Validate() error = %v, want error containing %s", err, tt.errMsg)
				}
			}
		})
	}
}

func TestFormatSQLValidator_Functions(t *testing.T) {
	validator := NewFormatSQLValidator()

	// Test allowed functions
	allowedFuncs := []string{"upper", "lower", "to_char", "concat", "round"}
	for _, fn := range allowedFuncs {
		if !validator.IsAllowedFunction(fn) {
			t.Errorf("Function %s should be allowed", fn)
		}
	}

	// Test adding new function
	validator.AddAllowedFunction("custom_func")
	if !validator.IsAllowedFunction("custom_func") {
		t.Error("custom_func should be allowed after adding")
	}

	// Test removing function
	validator.RemoveAllowedFunction("custom_func")
	if validator.IsAllowedFunction("custom_func") {
		t.Error("custom_func should not be allowed after removal")
	}

	// Test case insensitivity
	if !validator.IsAllowedFunction("UPPER") {
		t.Error("Function checking should be case insensitive")
	}
}