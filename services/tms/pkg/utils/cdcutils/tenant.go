package cdcutils

import (
	"github.com/emoss08/trenova/pkg/cdctypes"
)

// ExtractTenantInformation extracts organization and business unit IDs from CDC events.
// For DELETE operations, it uses the 'before' data; for others, it uses 'after' data.
// This ensures tenant isolation can be maintained across all operation types.
func ExtractTenantInformation(event *cdctypes.CDCEvent) (orgID, buID string, err error) {
	var data map[string]any
	switch event.Operation {
	case "delete":
		if event.Before == nil {
			return "", "", ErrDeleteEventMissingBeforeData
		}
		data = event.Before
	default:
		if event.After == nil {
			return "", "", ErrEventMissingAfterData
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

	if str, ok := value.(string); ok {
		return str
	}

	if m, ok := value.(map[string]any); ok {
		if str, strOk := m["string"].(string); strOk {
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
