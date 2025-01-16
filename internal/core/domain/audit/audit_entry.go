package audit

import (
	"context"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/trenova-app/transport/internal/core/domain/businessunit"
	"github.com/trenova-app/transport/internal/core/domain/organization"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/domain/user"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Entry)(nil)

type Entry struct {
	bun.BaseModel `bun:"table:audit_entries,alias:ae"`

	// Primary identifiers
	ID             pulid.ID `json:"id" bun:",pk,type:VARCHAR(100)"`
	ResourceID     string   `json:"resourceId" bun:",type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId" bun:",type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:",type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:",type:VARCHAR(100),notnull"`
	CorrelationID  pulid.ID `json:"correlationId,omitempty" bun:",type:VARCHAR(100)"`

	// Core fields
	Timestamp     int64               `json:"timestamp" bun:",notnull,default:extract(epoch from current_timestamp)::bigint"`
	Changes       map[string]any      `json:"changes,omitempty" bun:"type:JSONB,default:'{}'::jsonb"`
	PreviousState map[string]any      `json:"previousState,omitempty" bun:"type:JSONB,default:'{}'::jsonb"`
	CurrentState  map[string]any      `json:"currentState,omitempty" bun:"type:JSONB,default:'{}'::jsonb"`
	Metadata      map[string]any      `json:"metadata,omitempty" bun:"type:JSONB,default:'{}'::jsonb"`
	Resource      permission.Resource `json:"resource" bun:",type:VARCHAR(50),notnull"` // Should be the same as the resource in the permission service
	Action        permission.Action   `json:"action" bun:",type:VARCHAR(50),notnull"`   // Should be the same as the action in the permission service
	UserAgent     string              `json:"userAgent,omitempty" bun:",type:VARCHAR(255)"`
	Comment       string              `json:"comment,omitempty" bun:",type:TEXT"`
	SensitiveData bool                `json:"sensitiveData" bun:",notnull,default:false"`

	// Relationships
	User         *user.User                 `json:"-" bun:"rel:belongs-to,join:user_id=id"`
	Organization *organization.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *businessunit.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
}

func (e *Entry) Validate() error {
	return validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID, validation.Required.Error("Organization ID is required")),
		validation.Field(&e.BusinessUnitID, validation.Required.Error("Business Unit ID is required")),
		validation.Field(&e.Resource, validation.Required.Error("Resource is required")),
		validation.Field(&e.ResourceID, validation.Required.Error("Resource ID is required")),
		validation.Field(&e.Action, validation.Required.Error("Action is required")),
		validation.Field(&e.UserID, validation.Required.Error("User ID is required")),
	)
}

func (e *Entry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := time.Now().Unix()
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID == "" {
			e.ID = pulid.MustNew("ae_")
		}

		if e.Timestamp == 0 {
			e.Timestamp = now
		}
	}
	return nil
}
