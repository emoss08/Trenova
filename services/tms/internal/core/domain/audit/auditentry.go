package audit

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Entry)(nil)
	_ domaintypes.PostgresSearchable = (*Entry)(nil)
)

type Entry struct {
	bun.BaseModel `bun:"table:audit_entries,alias:ae"`

	ID             pulid.ID             `json:"id"                      bun:"id,pk,type:VARCHAR(100)"`
	UserID         pulid.ID             `json:"userId,omitempty"        bun:"user_id,type:VARCHAR(100),nullzero"`
	PrincipalType  string               `json:"principalType"           bun:"principal_type,type:VARCHAR(50),notnull,default:'session_user'"`
	PrincipalID    pulid.ID             `json:"principalId"             bun:"principal_id,type:VARCHAR(100),notnull"`
	APIKeyID       pulid.ID             `json:"apiKeyId,omitempty"      bun:"api_key_id,type:VARCHAR(100),nullzero"`
	BusinessUnitID pulid.ID             `json:"businessUnitId"          bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID             `json:"organizationId"          bun:"organization_id,type:VARCHAR(100),notnull"`
	Timestamp      int64                `json:"timestamp"               bun:"timestamp,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Changes        map[string]any       `json:"changes,omitempty"       bun:"changes,type:JSONB,default:'{}'::jsonb"`
	PreviousState  map[string]any       `json:"previousState,omitempty" bun:"previous_state,type:JSONB,default:'{}'::jsonb"`
	CurrentState   map[string]any       `json:"currentState,omitempty"  bun:"current_state,type:JSONB,default:'{}'::jsonb"`
	Metadata       map[string]any       `json:"metadata,omitempty"      bun:"metadata,type:JSONB,default:'{}'::jsonb"`
	Resource       permission.Resource  `json:"resource"                bun:"resource,type:VARCHAR(50),notnull"`  // Should be the same as the resource in the permission service
	Operation      permission.Operation `json:"operation"               bun:"operation,type:VARCHAR(50),notnull"` // Should be the same as the operation in the permission service
	ResourceID     string               `json:"resourceId"              bun:"resource_id,type:VARCHAR(100),notnull"`
	CorrelationID  string               `json:"correlationId,omitempty" bun:"correlation_id,type:VARCHAR(100)"`
	UserAgent      string               `json:"userAgent,omitempty"     bun:"user_agent,type:VARCHAR(255)"`
	Comment        string               `json:"comment,omitempty"       bun:"comment,type:TEXT"`
	IPAddress      string               `json:"ipAddress,omitempty"     bun:"ip_address,type:VARCHAR(45)"` // IPv6 addresses need space
	Category       Category             `json:"category"                bun:"category,type:audit_category_enum,notnull,default:'System'"`
	SensitiveData  bool                 `json:"sensitiveData"           bun:"sensitive_data,notnull,default:false"`
	Critical       bool                 `json:"critical"                bun:"critical,notnull,default:false"`
	User           *tenant.User         `json:"user,omitempty"          bun:"rel:belongs-to,join:user_id=id"`
	APIKey         *apikey.Key          `json:"apiKey,omitempty"        bun:"rel:belongs-to,join:api_key_id=id"`
	Organization   *tenant.Organization `json:"-"                       bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit   *tenant.BusinessUnit `json:"-"                       bun:"rel:belongs-to,join:business_unit_id=id"`
}

func (e *Entry) Validate() error {
	if e.PrincipalType == "" && e.UserID.IsNotNil() {
		e.PrincipalType = "session_user"
	}

	if e.PrincipalID.IsNil() && e.UserID.IsNotNil() {
		e.PrincipalID = e.UserID
	}

	if e.PrincipalType == "api_key" {
		if e.APIKeyID.IsNil() {
			e.APIKeyID = e.PrincipalID
		}
		e.UserID = pulid.Nil
	}

	if e.PrincipalType == "session_user" {
		e.APIKeyID = pulid.Nil
		if e.UserID.IsNil() {
			e.UserID = e.PrincipalID
		}
	}

	if err := validation.ValidateStruct(
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
		validation.Field(&e.PrincipalType, validation.Required.Error("Principal type is required")),
		validation.Field(&e.PrincipalID, validation.Required.Error("Principal ID is required")),
		validation.Field(&e.Category, validation.Required.Error("Category is required")),
	); err != nil {
		return err
	}

	switch e.PrincipalType {
	case "session_user":
		if e.UserID.IsNil() {
			return fmt.Errorf("user audit entries require a user ID")
		}
		if e.APIKeyID.IsNotNil() {
			return fmt.Errorf("user audit entries cannot include an api key ID")
		}
		if e.PrincipalID != e.UserID {
			return fmt.Errorf("user audit entries must use user ID as principal ID")
		}
	case "api_key":
		if e.APIKeyID.IsNil() {
			return fmt.Errorf("api key audit entries require an api key ID")
		}
		if e.UserID.IsNotNil() {
			return fmt.Errorf("api key audit entries cannot include a user ID")
		}
		if e.PrincipalID != e.APIKeyID {
			return fmt.Errorf("api key audit entries must use api key ID as principal ID")
		}
	default:
		return fmt.Errorf("unsupported principal type %q", e.PrincipalType)
	}

	return nil
}

func (e *Entry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("ae_")
		}

		if e.Timestamp == 0 {
			e.Timestamp = now
		}

		if e.Category == "" {
			e.Category = CategorySystem
		}
	}
	return nil
}

func (e *Entry) GetTableName() string {
	return "audit_entries"
}

func (e *Entry) RealtimeBatchKey() string {
	if e == nil {
		return ""
	}

	return fmt.Sprintf(
		"%s:%s:%s:%s:%s",
		e.OrganizationID,
		e.BusinessUnitID,
		e.PrincipalType,
		e.PrincipalID,
		e.APIKeyID,
	)
}

func (e *Entry) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ae",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "comment", Type: domaintypes.FieldTypeText},
			{Name: "ip_address", Type: domaintypes.FieldTypeText},
			{Name: "user_agent", Type: domaintypes.FieldTypeText},
			{Name: "correlation_id", Type: domaintypes.FieldTypeText},
			{Name: "resource", Type: domaintypes.FieldTypeText},
			{Name: "resource_id", Type: domaintypes.FieldTypeText},
			{Name: "operation", Type: domaintypes.FieldTypeText},
			{Name: "category", Type: domaintypes.FieldTypeEnum},
			{Name: "critical", Type: domaintypes.FieldTypeBoolean},
			{Name: "sensitive_data", Type: domaintypes.FieldTypeBoolean},
		},
	}
}
