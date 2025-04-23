package audit

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Entry)(nil)

type Entry struct {
	bun.BaseModel `bun:"table:audit_entries,alias:ae"`

	ID             pulid.ID            `json:"id" bun:",pk,type:VARCHAR(100)"`
	UserID         pulid.ID            `json:"userId" bun:",type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" bun:",type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID            `json:"organizationId" bun:",type:VARCHAR(100),notnull"`
	Timestamp      int64               `json:"timestamp" bun:",notnull,default:extract(epoch from current_timestamp)::bigint"`
	Changes        map[string]any      `json:"changes,omitempty" bun:"type:JSONB,default:'{}'::jsonb"`
	PreviousState  map[string]any      `json:"previousState,omitempty" bun:"type:JSONB,default:'{}'::jsonb"`
	CurrentState   map[string]any      `json:"currentState,omitempty" bun:"type:JSONB,default:'{}'::jsonb"`
	Metadata       map[string]any      `json:"metadata,omitempty" bun:"type:JSONB,default:'{}'::jsonb"`
	Resource       permission.Resource `json:"resource" bun:",type:VARCHAR(50),notnull"` // Should be the same as the resource in the permission service
	Action         permission.Action   `json:"action" bun:",type:VARCHAR(50),notnull"`   // Should be the same as the action in the permission service
	ResourceID     string              `json:"resourceId" bun:",type:VARCHAR(100),notnull"`
	CorrelationID  string              `json:"correlationId,omitempty" bun:",type:VARCHAR(100)"`
	UserAgent      string              `json:"userAgent,omitempty" bun:",type:VARCHAR(255)"`
	Comment        string              `json:"comment,omitempty" bun:",type:TEXT"`
	IPAddress      string              `json:"ipAddress,omitempty" bun:",type:VARCHAR(45)"` // IPv6 addresses need space
	Category       string              `json:"category" bun:",type:VARCHAR(50),notnull,default:'system'"`
	SensitiveData  bool                `json:"sensitiveData" bun:",notnull,default:false"`
	Critical       bool                `json:"critical" bun:",notnull,default:false"`

	// Relationships
	User         *user.User                 `json:"user,omitempty" bun:"rel:belongs-to,join:user_id=id"`
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
		validation.Field(&e.Category, validation.Required.Error("Category is required")),
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

		// Set default category if empty
		if e.Category == "" {
			e.Category = "system"
		}
	}
	return nil
}
