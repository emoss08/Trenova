package auditservice

import (
	"context"
	"maps"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func WithComment(comment string) auditports.LogOption {
	return func(entry *audit.Entry) error {
		entry.Comment = comment
		return nil
	}
}

func WithDiff(before, after any) auditports.LogOption {
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

func WithCompactDiff(before, after any) auditports.LogOption {
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

func WithMetadata(metadata map[string]any) auditports.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		maps.Copy(entry.Metadata, metadata)
		return nil
	}
}

func WithUserAgent(userAgent string) auditports.LogOption {
	return func(entry *audit.Entry) error {
		entry.UserAgent = userAgent
		return nil
	}
}

func WithCorrelationID() auditports.LogOption {
	return func(entry *audit.Entry) error {
		entry.CorrelationID = pulid.MustNew("corr_").String()
		return nil
	}
}

func WithCustomCorrelationID(id string) auditports.LogOption {
	return func(entry *audit.Entry) error {
		entry.CorrelationID = id
		return nil
	}
}

func WithCategory(category audit.Category) auditports.LogOption {
	return func(entry *audit.Entry) error {
		entry.Category = category
		return nil
	}
}

func WithCritical() auditports.LogOption {
	return func(entry *audit.Entry) error {
		entry.Critical = true
		return nil
	}
}

func WithIP(ip string) auditports.LogOption {
	return func(entry *audit.Entry) error {
		entry.IPAddress = ip
		return nil
	}
}

func WithTimestamp(timestamp time.Time) auditports.LogOption {
	return func(entry *audit.Entry) error {
		entry.Timestamp = timestamp.Unix()
		return nil
	}
}

func WithLocation(location string) auditports.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		entry.Metadata["location"] = location
		return nil
	}
}

func WithSessionID(sessionID string) auditports.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		entry.Metadata["sessionId"] = sessionID
		return nil
	}
}

func WithTags(tags ...string) auditports.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		entry.Metadata["tags"] = strings.Join(tags, ",")
		return nil
	}
}

func NewBulkLogEntry(
	params *auditports.LogActionParams,
	opts ...auditports.LogOption,
) auditports.BulkLogEntry {
	return auditports.BulkLogEntry{
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
	Opts      []auditports.LogOption
}

func BuildBulkLogEntries[T validationframework.TenantedEntity](
	params *BulkLogEntriesParams[T],
	opts ...auditports.LogOption,
) []auditports.BulkLogEntry {
	if len(params.Updated) == 0 {
		return nil
	}

	originalByID := make(map[string]T, len(params.Originals))
	for _, orig := range params.Originals {
		originalByID[orig.GetID().String()] = orig
	}

	entries := make([]auditports.BulkLogEntry, 0, len(params.Updated))
	for _, entity := range params.Updated {
		var previousState map[string]any
		var entryOpts []auditports.LogOption

		if orig, ok := originalByID[entity.GetID().String()]; ok {
			previousState = jsonutils.MustToJSON(orig)
			entryOpts = append(entryOpts, WithDiff(orig, entity))
		}

		entryOpts = append(entryOpts, opts...)

		entry := auditports.BulkLogEntry{
			Params: &auditports.LogActionParams{
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

func WithRequest(ctx context.Context) auditports.LogOption {
	return func(entry *audit.Entry) error {
		meta, ok := requestmeta.From(ctx)
		if !ok {
			return nil
		}

		if entry.IPAddress == "" {
			entry.IPAddress = meta.ClientIP
		}
		if entry.UserAgent == "" {
			entry.UserAgent = meta.UserAgent
		}
		if entry.CorrelationID == "" {
			entry.CorrelationID = meta.RequestID
		}
		if meta.RequestID != "" {
			if entry.Metadata == nil {
				entry.Metadata = make(map[string]any)
			}
			entry.Metadata["requestId"] = meta.RequestID
		}

		return nil
	}
}
