package cdcutils

import (
	"github.com/emoss08/trenova/pkg/types/cdctypes"
	"github.com/rotisserie/eris"
)

// ExtractTenantInformation extracts organization and business unit IDs from CDC events.
// For DELETE operations, it uses the 'before' data; for others, it uses 'after' data.
// This ensures tenant isolation can be maintained across all operation types.
func ExtractTenantInformation(event *cdctypes.CDCEvent) (orgID, buID string, err error) {
	// Determine which data to use based on operation type
	var data map[string]any
	switch event.Operation {
	case "delete":
		if event.Before == nil {
			return "", "", eris.New("delete event missing 'before' data for tenant extraction")
		}
		data = event.Before
	default:
		if event.After == nil {
			return "", "", eris.New("event missing 'after' data for tenant extraction")
		}
		data = event.After
	}

	// Extract organization ID
	orgID, ok := data["organization_id"].(string)
	if !ok {
		return "", "", cdctypes.ErrOrganizationIDMissing
	}

	// Extract business unit ID
	buID, ok = data["business_unit_id"].(string)
	if !ok {
		return "", "", cdctypes.ErrBusinessUnitIDMissing
	}

	return orgID, buID, nil
}

// ExtractStringField safely extracts a string field from a map with optional Avro handling
func ExtractStringField(data map[string]any, field string) string {
	if data == nil {
		return ""
	}

	value := data[field]
	if value == nil {
		return ""
	}

	// Handle direct string
	if str, ok := value.(string); ok {
		return str
	}

	// Handle Avro optional format {"string": "value"}
	if m, ok := value.(map[string]any); ok {
		if str, ok := m["string"].(string); ok {
			return str
		}
	}

	return ""
}

// ExtractIntField safely extracts an integer field from a map with optional Avro handling
func ExtractIntField(data map[string]any, field string) *int64 {
	if data == nil {
		return nil
	}

	value := data[field]
	if value == nil {
		return nil
	}

	// Handle direct numeric types
	switch v := value.(type) {
	case int64:
		return &v
	case float64:
		i := int64(v)
		return &i
	case int:
		i := int64(v)
		return &i
	}

	// Handle Avro optional format {"int": 123} or {"long": 123}
	if m, ok := value.(map[string]any); ok {
		for _, key := range []string{"int", "long"} {
			if val, exists := m[key]; exists {
				switch v := val.(type) {
				case int64:
					return &v
				case float64:
					i := int64(v)
					return &i
				case int:
					i := int64(v)
					return &i
				}
			}
		}
	}

	return nil
}
