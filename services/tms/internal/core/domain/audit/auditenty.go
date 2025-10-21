package audit

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Entry)(nil)

type Entry struct {
	bun.BaseModel `bun:"table:audit_entries,alias:ae"`

	ID             pulid.ID             `json:"id"                      bun:"id,pk,type:VARCHAR(100)"`
	UserID         pulid.ID             `json:"userId"                  bun:"user_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID             `json:"businessUnitId"          bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID             `json:"organizationId"          bun:"organization_id,type:VARCHAR(100),notnull"`
	Timestamp      int64                `json:"timestamp"               bun:"timestamp,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Changes        map[string]any       `json:"changes,omitempty"       bun:"changes,type:JSONB,default:'{}'::jsonb"`
	PreviousState  map[string]any       `json:"previousState,omitempty" bun:"previous_state,type:JSONB,default:'{}'::jsonb"`
	CurrentState   map[string]any       `json:"currentState,omitempty"  bun:"current_state,type:JSONB,default:'{}'::jsonb"`
	Metadata       map[string]any       `json:"metadata,omitempty"      bun:"metadata,type:JSONB,default:'{}'::jsonb"`
	Resource       permission.Resource  `json:"resource"                bun:"resource,type:VARCHAR(50),notnull"` // Should be the same as the resource in the permission service
	Operation      permission.Operation `json:"operation"               bun:"operation,type:INT,notnull"`        // Should be the same as the operation in the permission service
	ResourceID     string               `json:"resourceId"              bun:"resource_id,type:VARCHAR(100),notnull"`
	CorrelationID  string               `json:"correlationId,omitempty" bun:"correlation_id,type:VARCHAR(100)"`
	UserAgent      string               `json:"userAgent,omitempty"     bun:"user_agent,type:VARCHAR(255)"`
	Comment        string               `json:"comment,omitempty"       bun:"comment,type:TEXT"`
	IPAddress      string               `json:"ipAddress,omitempty"     bun:"ip_address,type:VARCHAR(45)"` // IPv6 addresses need space
	Category       Category             `json:"category"                bun:"category,type:audit_category_enum,notnull,default:'System'"`
	SensitiveData  bool                 `json:"sensitiveData"           bun:"sensitive_data,notnull,default:false"`
	Critical       bool                 `json:"critical"                bun:"critical,notnull,default:false"`
	User           *tenant.User         `json:"user,omitempty"          bun:"rel:belongs-to,join:user_id=id"`
	Organization   *tenant.Organization `json:"-"                       bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit   *tenant.BusinessUnit `json:"-"                       bun:"rel:belongs-to,join:business_unit_id=id"`
}

func (e *Entry) Validate() error {
	return validation.ValidateStruct(
		e,
		validation.Field(
			&e.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&e.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),
		validation.Field(&e.Resource, validation.Required.Error("Resource is required")),
		validation.Field(&e.ResourceID, validation.Required.Error("Resource ID is required")),
		validation.Field(&e.Operation, validation.Required.Error("Operation is required")),
		validation.Field(&e.UserID, validation.Required.Error("User ID is required")),
		validation.Field(&e.Category, validation.Required.Error("Category is required")),
	)
}

func (e *Entry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("ae_")
		}

		if e.Timestamp == 0 {
			e.Timestamp = now
		}

		if e.Category == "" {
			e.Category = "system"
		}
	}
	return nil
}
