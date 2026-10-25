package audit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const principalSessionUser = "session_user"

var (
	_ domaintypes.PostgresSearchable = (*Entry)(nil)
	_ pagination.CursorEntity        = (*Entry)(nil)
)

type Entry struct {
	keel.Table `keel:"audit_entries"`

	ID             pulid.ID             `json:"id"                      keel:"id,pk,autoid:ae_"`
	UserID         pulid.ID             `json:"userId,omitempty"        keel:"user_id,nullzero"`
	PrincipalType  string               `json:"principalType"           keel:"principal_type"`
	PrincipalID    pulid.ID             `json:"principalId"             keel:"principal_id"`
	APIKeyID       pulid.ID             `json:"apiKeyId,omitempty"      keel:"api_key_id,nullzero"`
	BusinessUnitID pulid.ID             `json:"businessUnitId"          keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID             `json:"organizationId"          keel:"organization_id,pk,tenant"`
	Timestamp      int64                `json:"timestamp"               keel:"timestamp"`
	Changes        map[string]any       `json:"changes,omitempty"       keel:"changes,jsonb"`
	PreviousState  map[string]any       `json:"previousState,omitempty" keel:"previous_state,jsonb"`
	CurrentState   map[string]any       `json:"currentState,omitempty"  keel:"current_state,jsonb"`
	Metadata       map[string]any       `json:"metadata,omitempty"      keel:"metadata,jsonb"`
	Resource       permission.Resource  `json:"resource"                keel:"resource"`  // Should be the same as the resource in the permission service
	Operation      permission.Operation `json:"operation"               keel:"operation"` // Should be the same as the operation in the permission service
	ResourceID     string               `json:"resourceId"              keel:"resource_id"`
	CorrelationID  string               `json:"correlationId,omitempty" keel:"correlation_id,nullzero"`
	UserAgent      string               `json:"userAgent,omitempty"     keel:"user_agent,nullzero"`
	Comment        string               `json:"comment,omitempty"       keel:"comment,nullzero"`
	IPAddress      string               `json:"ipAddress,omitempty"     keel:"ip_address,nullzero"` // IPv6 addresses need space
	Category       Category             `json:"category"                keel:"category"`
	SensitiveData  bool                 `json:"sensitiveData"           keel:"sensitive_data"`
	Critical       bool                 `json:"critical"                keel:"critical"`
	User           *tenant.User         `json:"user,omitempty"          keel:"belongs:user_id"`
	APIKey         *apikey.Key          `json:"apiKey,omitempty"        keel:"belongs:api_key_id"`
	Organization   *tenant.Organization `json:"-"`
	BusinessUnit   *tenant.BusinessUnit `json:"-"`
}

func (e *Entry) BeforeCreate(_ context.Context) error {
	if e.Category == "" {
		e.Category = CategorySystem
	}
	if e.PrincipalType == "" {
		e.PrincipalType = principalSessionUser
	}
	if e.Timestamp == 0 {
		e.Timestamp = timeutils.NowUnix()
	}

	return nil
}

func (e *Entry) Validate() error {
	if e.PrincipalType == "" && e.UserID.IsNotNil() {
		e.PrincipalType = principalSessionUser
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

	if e.PrincipalType == principalSessionUser {
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
	case principalSessionUser:
		if e.UserID.IsNil() {
			return errors.New("user audit entries require a user ID")
		}
		if e.APIKeyID.IsNotNil() {
			return errors.New("user audit entries cannot include an api key ID")
		}
		if e.PrincipalID != e.UserID {
			return errors.New("user audit entries must use user ID as principal ID")
		}
	case "api_key":
		if e.APIKeyID.IsNil() {
			return errors.New("api key audit entries require an api key ID")
		}
		if e.UserID.IsNotNil() {
			return errors.New("api key audit entries cannot include a user ID")
		}
		if e.PrincipalID != e.APIKeyID {
			return errors.New("api key audit entries must use api key ID as principal ID")
		}
	case "system":
		if e.UserID.IsNotNil() {
			return errors.New("system audit entries cannot include a user ID")
		}
		if e.APIKeyID.IsNotNil() {
			return errors.New("system audit entries cannot include an api key ID")
		}
	case "agent":
		if e.APIKeyID.IsNotNil() {
			return errors.New("agent audit entries cannot include an api key ID")
		}
		if e.UserID.IsNotNil() && e.UserID == e.PrincipalID {
			return errors.New("agent audit entries cannot use the user ID as principal ID")
		}
	default:
		return fmt.Errorf("unsupported principal type %q", e.PrincipalType)
	}

	return nil
}

func (e *Entry) CreditAgent(agentName string) {
	credit := AgentCredit(agentName)
	if strings.TrimSpace(e.Comment) == "" {
		e.Comment = credit
		return
	}

	e.Comment = e.Comment + " (" + credit + ")"
}

func AgentCredit(agentName string) string {
	if name := strings.TrimSpace(agentName); name != "" {
		return "Ran by " + name
	}

	return "Ran by an agent"
}

func (e *Entry) GetTableName() string {
	return "audit_entries"
}

func (e *Entry) GetID() pulid.ID {
	return e.ID
}

func (e *Entry) GetCreatedAt() int64 {
	return e.Timestamp
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

var entityReferenceKeys = []string{
	"proNumber",
	"orderNumber",
	"invoiceNumber",
	"number",
	"code",
	"name",
	"username",
	"emailAddress",
	"reference",
	"title",
	"label",
}

func (e *Entry) EntityReference() string {
	if e == nil {
		return ""
	}

	for _, state := range []map[string]any{e.CurrentState, e.PreviousState, e.Metadata} {
		if ref := firstStringValue(state, entityReferenceKeys); ref != "" {
			return ref
		}
	}

	return ""
}

func firstStringValue(values map[string]any, keys []string) string {
	if len(values) == 0 {
		return ""
	}

	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		str, ok := raw.(string)
		if !ok {
			continue
		}
		if trimmed := strings.TrimSpace(str); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
