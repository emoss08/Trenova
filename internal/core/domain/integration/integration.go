package integration

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Integration)(nil)
	_ domain.Validatable        = (*Integration)(nil)
)

// Integration represents an integration with an external service
type Integration struct {
	bun.BaseModel `bun:"table:integrations,alias:i" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`

	// Integration details
	Type        Type     `bun:"type,type:integration_type,notnull" json:"type"`
	Name        string   `bun:"name,type:VARCHAR(100),notnull" json:"name"`
	Description string   `bun:"description,type:TEXT" json:"description"`
	Enabled     bool     `bun:"enabled,type:BOOLEAN,notnull,default:true" json:"enabled"`
	BuiltBy     string   `bun:"built_by,type:VARCHAR(100)" json:"builtBy"`
	Overview    string   `bun:"overview,type:TEXT" json:"overview"`
	Screenshots []string `bun:"screenshots,type:JSONB" json:"screenshots"`
	Features    []string `bun:"features,type:JSONB" json:"features"`
	Category    Category `bun:"category,type:integration_category,notnull" json:"category"`

	// Type-specific configuration options and settings
	ConfigFields     map[string]Field  `bun:"config_fields,type:JSONB" json:"configFields"`
	EventTriggers    []EventTrigger    `bun:"event_triggers,type:JSONB" json:"eventTriggers"`
	WebhookEndpoints []WebhookEndpoint `bun:"webhook_endpoints,type:JSONB" json:"webhookEndpoints"`

	// Configuration stored as JSON
	Configuration map[string]any `bun:"configuration,type:JSONB,default:'{}'::jsonb" json:"configuration"`

	// Usage statistics
	LastUsed    int64  `bun:"last_used,type:BIGINT" json:"lastUsed"`
	UsageCount  int64  `bun:"usage_count,type:BIGINT,notnull,default:0" json:"usageCount"`
	ErrorCount  int64  `bun:"error_count,type:BIGINT,notnull,default:0" json:"errorCount"`
	LastError   string `bun:"last_error,type:TEXT" json:"lastError"`
	LastErrorAt int64  `bun:"last_error_at,type:BIGINT" json:"lastErrorAt"`

	EnabledByID *pulid.ID `bun:"enabled_by_id,type:VARCHAR(100)" json:"enabledById"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	EnabledBy    *user.User                 `json:"enabledBy,omitempty" bun:"rel:belongs-to,join:enabled_by_id=id"`
}

// Validate validates the Integration entity
func (i *Integration) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, i,
		validation.Field(&i.Type, validation.Required.Error("Integration type is required")),
		validation.Field(&i.Name, validation.Required.Error("Integration name is required")),
		// validation.Field(&i.Status, validation.Required.Error("Integration status is required")),
	)

	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (i *Integration) GetID() string {
	return i.ID.String()
}

func (i *Integration) GetTableName() string {
	return "integrations"
}

func (i *Integration) GetVersion() int64 {
	return i.Version
}

func (i *Integration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("int_")
		}

		i.CreatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

// GetLastUsedTime returns the last used time as a time.Time
func (i *Integration) GetLastUsedTime() time.Time {
	if i.LastUsed == 0 {
		return time.Time{}
	}
	return time.Unix(i.LastUsed, 0)
}

// GetLastErrorTime returns the last error time as a time.Time
func (i *Integration) GetLastErrorTime() time.Time {
	if i.LastErrorAt == 0 {
		return time.Time{}
	}
	return time.Unix(i.LastErrorAt, 0)
}

// RecordUsage increments the usage count and updates the LastUsed timestamp
func (i *Integration) RecordUsage() {
	i.UsageCount++
	i.LastUsed = timeutils.NowUnix()
}

// RecordError records an error occurrence
func (i *Integration) RecordError(errorMessage string) {
	i.ErrorCount++
	i.LastError = errorMessage
	i.LastErrorAt = timeutils.NowUnix()

	// If there's an error, change status to error
	i.Enabled = false
}

// ClearError clears the error state
func (i *Integration) ClearError() {
	i.LastError = ""
	i.Enabled = true
}
