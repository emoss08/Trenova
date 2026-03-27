package tablechangealert

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*TCASubscription)(nil)
	_ validationframework.TenantedEntity = (*TCASubscription)(nil)
	_ domaintypes.PostgresSearchable     = (*TCASubscription)(nil)
)

type TCASubscription struct {
	bun.BaseModel `bun:"table:tca_subscriptions,alias:tcas" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID           `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Name           string             `json:"name"           bun:"name,type:VARCHAR(255),notnull"`
	TableName      string             `json:"tableName"      bun:"table_name,type:VARCHAR(100),notnull"`
	RecordID       *string            `json:"recordId"       bun:"record_id,type:VARCHAR(100)"`
	EventTypes     []string           `json:"eventTypes"     bun:"event_types,type:JSONB,notnull"`
	Conditions     []Condition        `json:"conditions"     bun:"conditions,type:JSONB,notnull,default:'[]'"`
	ConditionMatch string             `json:"conditionMatch" bun:"condition_match,type:VARCHAR(10),notnull,default:'all'"`
	WatchedColumns []string           `json:"watchedColumns" bun:"watched_columns,type:JSONB,notnull,default:'[]'"`
	CustomTitle    string             `json:"customTitle"    bun:"custom_title,type:VARCHAR(500),notnull,default:''"`
	CustomMessage  string             `json:"customMessage"  bun:"custom_message,type:TEXT,notnull,default:''"`
	Topic          string             `json:"topic"          bun:"topic,type:VARCHAR(100),notnull,default:''"`
	Priority       string             `json:"priority"       bun:"priority,type:VARCHAR(20),notnull,default:'medium'"`
	Status         SubscriptionStatus `json:"status"         bun:"status,type:tca_subscription_status_enum,notnull,default:'Active'"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	User         *tenant.User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
}

func (s *TCASubscription) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(s,
		validation.Field(&s.Name, validation.Required, validation.Length(1, 255)),
		validation.Field(&s.TableName, validation.Required, validation.Length(1, 100)),
		validation.Field(&s.EventTypes, validation.Required, validation.Length(1, 3)),
		validation.Field(&s.ConditionMatch, validation.Required, validation.In("all", "any")),
		validation.Field(&s.Priority, validation.Required, validation.In("critical", "high", "medium", "low")),
		validation.Field(&s.CustomTitle, validation.Length(0, 500)),
		validation.Field(&s.CustomMessage, validation.Length(0, 5000)),
		validation.Field(&s.Topic, validation.Length(0, 100)),
		validation.Field(&s.Status, validation.Required, validation.In(
			SubscriptionStatusActive,
			SubscriptionStatusPaused,
		)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	for _, et := range s.EventTypes {
		if !ValidEventType(et) {
			multiErr.Add("eventTypes", errortypes.ErrInvalid, "Invalid event type: "+et)
		}
	}

	for i, cond := range s.Conditions {
		prefix := fmt.Sprintf("conditions[%d]", i)
		if !ValidConditionOperator(string(cond.Operator)) {
			multiErr.Add(prefix+".operator", errortypes.ErrInvalid, "Invalid condition operator: "+string(cond.Operator))
		}
		if !IsUnaryOperator(cond.Operator) && cond.Value == nil {
			multiErr.Add(prefix+".value", errortypes.ErrRequired, "Value is required for operator: "+string(cond.Operator))
		}
	}
}

func (s *TCASubscription) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("tcas_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}

func (s *TCASubscription) GetID() pulid.ID {
	return s.ID
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
		},
	}
}
