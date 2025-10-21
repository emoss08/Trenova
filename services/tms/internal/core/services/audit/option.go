package audit

import (
	"maps"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
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
			return err
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
		results, err := jsonutils.JSONDiff(before, after, jsonutils.DefaultOptions())
		if err != nil {
			return err
		}

		changes := make(map[string]any)
		for key, change := range results {
			changes[key] = map[string]any{
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

// WithMetadata adds custom metadata to the audit entry
func WithMetadata(metadata map[string]any) services.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		maps.Copy(entry.Metadata, metadata)
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
func WithCategory(category audit.Category) services.LogOption {
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
