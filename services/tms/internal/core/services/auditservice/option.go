package auditservice

import (
	"maps"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func WithComment(comment string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.Comment = comment
		return nil
	}
}

func WithDiff(before, after any) services.LogOption {
	return func(entry *audit.Entry) error {
		opts := jsonutils.DefaultOptions()
		opts.IgnoreFields = []string{"updated_at", "version"}

		diff, err := jsonutils.JSONDiff(before, after, opts)
		if err != nil {
			return err
		}

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

func WithMetadata(metadata map[string]any) services.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		maps.Copy(entry.Metadata, metadata)
		return nil
	}
}

func WithUserAgent(userAgent string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.UserAgent = userAgent
		return nil
	}
}

func WithCorrelationID() services.LogOption {
	return func(entry *audit.Entry) error {
		entry.CorrelationID = pulid.MustNew("corr_").String()
		return nil
	}
}

func WithCustomCorrelationID(id string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.CorrelationID = id
		return nil
	}
}

func WithCategory(category audit.Category) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.Category = category
		return nil
	}
}

func WithCritical() services.LogOption {
	return func(entry *audit.Entry) error {
		entry.Critical = true
		return nil
	}
}

func WithIP(ip string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.IPAddress = ip
		return nil
	}
}

func WithTimestamp(timestamp time.Time) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.Timestamp = timestamp.Unix()
		return nil
	}
}

func WithLocation(location string) services.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		entry.Metadata["location"] = location
		return nil
	}
}

func WithSessionID(sessionID string) services.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		entry.Metadata["sessionId"] = sessionID
		return nil
	}
}

func WithTags(tags ...string) services.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		entry.Metadata["tags"] = strings.Join(tags, ",")
		return nil
	}
}

func NewBulkLogEntry(
	params *services.LogActionParams,
	opts ...services.LogOption,
) services.BulkLogEntry {
	return services.BulkLogEntry{
		Params:  params,
		Options: opts,
	}
}

type BulkLogEntriesParams[T validationframework.TenantedEntity] struct {
	Resource  permission.Resource
	Operation permission.Operation
	UserID    pulid.ID
	Updated   []T
	Originals []T
	Opts      []services.LogOption
}

func BuildBulkLogEntries[T validationframework.TenantedEntity](
	params *BulkLogEntriesParams[T],
	opts ...services.LogOption,
) []services.BulkLogEntry {
	if len(params.Updated) == 0 {
		return nil
	}

	originalByID := make(map[string]T, len(params.Originals))
	for _, orig := range params.Originals {
		originalByID[orig.GetID().String()] = orig
	}

	entries := make([]services.BulkLogEntry, 0, len(params.Updated))
	for _, entity := range params.Updated {
		var previousState map[string]any
		var entryOpts []services.LogOption

		if orig, ok := originalByID[entity.GetID().String()]; ok {
			previousState = jsonutils.MustToJSON(orig)
			entryOpts = append(entryOpts, WithDiff(orig, entity))
		}

		entryOpts = append(entryOpts, opts...)

		entry := services.BulkLogEntry{
			Params: &services.LogActionParams{
				Resource:       params.Resource,
				ResourceID:     entity.GetID().String(),
				Operation:      params.Operation,
				UserID:         params.UserID,
				CurrentState:   jsonutils.MustToJSON(entity),
				PreviousState:  previousState,
				OrganizationID: entity.GetOrganizationID(),
				BusinessUnitID: entity.GetBusinessUnitID(),
			},
			Options: entryOpts,
		}

		entries = append(entries, entry)
	}

	return entries
}
