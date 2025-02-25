package audit

import (
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
)

// WithComment adds a comment to the audit entry
func WithComment(comment string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.Comment = comment
		return nil
	}
}

// WithDiff generates a detailed difference between two objects
func WithDiff(before, after any) services.LogOption {
	return func(entry *audit.Entry) error {
		opts := jsonutils.DefaultOptions()
		// Customize options as needed
		opts.IgnoreFields = []string{"updated_at", "version"}

		diff, err := jsonutils.JSONDiff(before, after, opts)
		if err != nil {
			return eris.Wrap(err, "failed to compute diff")
		}

		// Convert the structured diff to a simple map[string]any
		changes := make(map[string]any, len(diff))
		for path, change := range diff {
			changes[path] = map[string]any{
				"from":      change.From,
				"to":        change.To,
				"type":      change.Type,
				"fieldType": change.FieldType,
				"path":      change.Path,
			}
		}

		entry.Changes = changes
		return nil
	}
}

// WithCompactDiff generates a simplified difference format
// Useful for situations where the full diff would be too large
func WithCompactDiff(before, after any) services.LogOption {
	return func(entry *audit.Entry) error {
		// Convert to JSON and back to normalize the objects
		beforeJSON, err := sonic.Marshal(before)
		if err != nil {
			return eris.Wrap(err, "failed to marshal 'before' object")
		}

		afterJSON, err := sonic.Marshal(after)
		if err != nil {
			return eris.Wrap(err, "failed to marshal 'after' object")
		}

		var beforeMap, afterMap map[string]any
		if err := sonic.Unmarshal(beforeJSON, &beforeMap); err != nil {
			return eris.Wrap(err, "failed to unmarshal 'before' object")
		}

		if err := sonic.Unmarshal(afterJSON, &afterMap); err != nil {
			return eris.Wrap(err, "failed to unmarshal 'after' object")
		}

		// Find changed fields
		changes := make(map[string]any)
		for key, afterValue := range afterMap {
			if beforeValue, exists := beforeMap[key]; exists {
				// Compare values by marshaling to JSON and comparing strings
				beforeFieldJSON, err1 := sonic.Marshal(beforeValue)
				afterFieldJSON, err2 := sonic.Marshal(afterValue)

				if err1 != nil || err2 != nil {
					// Handle error or skip this field
					continue
				}

				if string(beforeFieldJSON) != string(afterFieldJSON) {
					changes[key] = map[string]any{
						"from": beforeValue,
						"to":   afterValue,
					}
				}
			} else {
				// Added field
				changes[key] = map[string]any{
					"from": nil,
					"to":   afterValue,
				}
			}
		}

		// Find removed fields
		for key, beforeValue := range beforeMap {
			if _, exists := afterMap[key]; !exists {
				changes[key] = map[string]any{
					"from": beforeValue,
					"to":   nil,
				}
			}
		}

		entry.Changes = changes
		return nil
	}
}

// WithMetadata adds custom metadata to the audit entry
func WithMetadata(metadata map[string]any) services.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		for k, v := range metadata {
			entry.Metadata[k] = v
		}
		return nil
	}
}

// WithUserAgent adds user agent information to the audit entry
func WithUserAgent(userAgent string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.UserAgent = userAgent
		return nil
	}
}

// WithCorrelationID generates a correlation ID for the audit entry
func WithCorrelationID() services.LogOption {
	return func(entry *audit.Entry) error {
		// Automatically generate a correlation ID.
		entry.CorrelationID = pulid.MustNew("corr_").String()
		return nil
	}
}

// WithCustomCorrelationID sets a specific correlation ID for the audit entry
func WithCustomCorrelationID(id string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.CorrelationID = id
		return nil
	}
}

// WithCategory sets the category for the audit entry
func WithCategory(category string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.Category = category
		return nil
	}
}

// WithCritical marks the audit entry as critical
// Critical entries are handled with higher priority and will be sent
// directly to storage if the buffer rejects them
func WithCritical() services.LogOption {
	return func(entry *audit.Entry) error {
		entry.Critical = true
		return nil
	}
}

// WithIP adds the IP address to the audit entry
func WithIP(ip string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.IPAddress = ip
		return nil
	}
}

// WithTimestamp sets a specific timestamp for the audit entry
func WithTimestamp(timestamp time.Time) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.Timestamp = timestamp.Unix()
		return nil
	}
}

// WithLocation adds the geographic location information to the audit entry
func WithLocation(location string) services.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		entry.Metadata["location"] = location
		return nil
	}
}

// WithSessionID adds a session ID to the audit entry
func WithSessionID(sessionID string) services.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		entry.Metadata["sessionId"] = sessionID
		return nil
	}
}

// WithTags adds tags to the audit entry for easier searching and filtering
func WithTags(tags ...string) services.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		entry.Metadata["tags"] = strings.Join(tags, ",")
		return nil
	}
}
