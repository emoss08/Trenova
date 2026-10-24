package tablechangealert

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*TCASubscription)(nil)
	_ domaintypes.PostgresSearchable     = (*TCASubscription)(nil)
)

type TCASubscription struct {
	keel.Table `                                                  keel:"tca_subscriptions"`

	ID             pulid.ID           `json:"id" keel:"id,pk,autoid:tcas_"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID         pulid.ID           `json:"userId" keel:"user_id"`
	Name           string             `json:"name" keel:"name"`
	TableName      string             `json:"tableName" keel:"table_name"`
	RecordID       *string            `json:"recordId" keel:"record_id"`
	EventTypes     []string           `json:"eventTypes" keel:"event_types,jsonb"`
	Conditions     []Condition        `json:"conditions" keel:"conditions,jsonb"`
	ConditionMatch string             `json:"conditionMatch" keel:"condition_match"`
	WatchedColumns []string           `json:"watchedColumns" keel:"watched_columns,jsonb"`
	CustomTitle    string             `json:"customTitle" keel:"custom_title"`
	CustomMessage  string             `json:"customMessage" keel:"custom_message"`
	Topic          string             `json:"topic" keel:"topic"`
	Priority       string             `json:"priority" keel:"priority"`
	Status         SubscriptionStatus `json:"status" keel:"status"`
	Version        int64              `json:"version" keel:"version,version"`
	CreatedAt      int64              `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	User         *tenant.User         `json:"user,omitempty" keel:"belongs:user_id"`
}

func (s *TCASubscription) BeforeCreate(_ context.Context) error {
	if s.ConditionMatch == "" {
		s.ConditionMatch = "all"
	}
	if s.Priority == "" {
		s.Priority = "medium"
	}
	if s.Status == "" {
		s.Status = SubscriptionStatusActive
	}

	return nil
}

func (s *TCASubscription) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		s,
		validation.Field(&s.Name, validation.Required, validation.Length(1, 255)),
		validation.Field(&s.TableName, validation.Required, validation.Length(1, 100)),
		validation.Field(&s.EventTypes, validation.Required, validation.Length(1, 3)),
		validation.Field(&s.ConditionMatch, validation.Required, validation.In("all", "any")),
		validation.Field(
			&s.Priority,
			validation.Required,
			validation.In("critical", "high", "medium", "low"),
		),
		validation.Field(&s.CustomTitle, validation.Length(0, 500)),
		validation.Field(&s.CustomMessage, validation.Length(0, 5000)),
		validation.Field(&s.Topic, validation.Length(0, 100)),
		validation.Field(&s.Status, validation.Required, validation.In(
			SubscriptionStatusActive,
			SubscriptionStatusPaused,
		)),
	))

	for _, et := range s.EventTypes {
		if !ValidEventType(et) {
			multiErr.Add("eventTypes", errortypes.ErrInvalid, "Invalid event type: {0}", et)
		}
	}

	for i, cond := range s.Conditions {
		prefix := fmt.Sprintf("conditions[%d]", i)
		if !ValidConditionOperator(string(cond.Operator)) {
			multiErr.Add(
				prefix+".operator",
				errortypes.ErrInvalid,
				"Invalid condition operator: {0}", string(cond.Operator),
			)
		}
		if !IsUnaryOperator(cond.Operator) && cond.Value == nil {
			multiErr.Add(
				prefix+".value",
				errortypes.ErrRequired,
				"Value is required for operator: {0}", string(cond.Operator),
			)
		}
	}
}

func (s *TCASubscription) GetID() pulid.ID {
	return s.ID
}

func (s *TCASubscription) GetCreatedAt() int64 {
	return s.CreatedAt
}

func (s *TCASubscription) GetOrganizationID() pulid.ID {
	return s.OrganizationID
}

func (s *TCASubscription) GetBusinessUnitID() pulid.ID {
	return s.BusinessUnitID
}

func (s *TCASubscription) GetTableName() string {
	return "tca_subscriptions"
}

func (s *TCASubscription) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "tcas",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "table_name", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}
