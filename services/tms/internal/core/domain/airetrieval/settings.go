package airetrieval

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Settings)(nil)
	_ validationframework.TenantedEntity = (*Settings)(nil)
)

var (
	DefaultMonthlyIndexingBudgetUSD = decimal.NewFromInt(10)
	MaxMonthlyIndexingBudgetUSD     = decimal.NewFromInt(100000)
)

var indexingBudgetRule = domainvalidation.BudgetUSD(MaxMonthlyIndexingBudgetUSD, "100,000")

type Settings struct {
	bun.BaseModel `bun:"table:ai_retrieval_settings,alias:airs" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	MemoryEnabled          bool `json:"memoryEnabled"          bun:"memory_enabled,type:BOOLEAN,notnull"`
	DocumentsEnabled       bool `json:"documentsEnabled"       bun:"documents_enabled,type:BOOLEAN,notnull"`
	InboundMessagesEnabled bool `json:"inboundMessagesEnabled" bun:"inbound_messages_enabled,type:BOOLEAN,notnull"`

	MonthlyIndexingBudgetUSD decimal.Decimal `json:"monthlyIndexingBudgetUsd" bun:"monthly_indexing_budget_usd,type:NUMERIC(14,2),notnull"`

	Paused       bool        `json:"paused"       bun:"paused,type:BOOLEAN,notnull,default:false"`
	PausedReason PauseReason `json:"pausedReason" bun:"paused_reason,type:VARCHAR(20),nullzero"`
	PausedAt     *int64      `json:"pausedAt"     bun:"paused_at,type:BIGINT,nullzero"`

	ActiveModelKey    string `json:"activeModelKey"    bun:"active_model_key,type:VARCHAR(300),nullzero"`
	Dimensions        int    `json:"dimensions"        bun:"dimensions,type:INTEGER,nullzero"`
	PendingModelKey   string `json:"pendingModelKey"   bun:"pending_model_key,type:VARCHAR(300),nullzero"`
	PendingDimensions int    `json:"pendingDimensions" bun:"pending_dimensions,type:INTEGER,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func DefaultSettings(orgID, buID pulid.ID) *Settings {
	return &Settings{
		OrganizationID:           orgID,
		BusinessUnitID:           buID,
		MemoryEnabled:            true,
		DocumentsEnabled:         true,
		InboundMessagesEnabled:   true,
		MonthlyIndexingBudgetUSD: DefaultMonthlyIndexingBudgetUSD,
	}
}

func (s *Settings) SourceEnabled(sourceType SourceType) bool {
	switch sourceType {
	case SourceTypeMemory:
		return s.MemoryEnabled
	case SourceTypeDocument:
		return s.DocumentsEnabled
	case SourceTypeInboundMessage:
		return s.InboundMessagesEnabled
	default:
		return false
	}
}

func (s *Settings) EnabledSourceTypes() []SourceType {
	enabled := make([]SourceType, 0, len(AllSourceTypes()))
	for _, sourceType := range AllSourceTypes() {
		if s.SourceEnabled(sourceType) {
			enabled = append(enabled, sourceType)
		}
	}

	return enabled
}

func (s *Settings) HasActiveModel() bool { return s.ActiveModelKey != "" }

func (s *Settings) HasPendingModel() bool { return s.PendingModelKey != "" }

func (s *Settings) IndexedModelKeys() []string {
	keys := make([]string, 0, 2)
	if s.HasActiveModel() {
		keys = append(keys, s.ActiveModelKey)
	}
	if s.HasPendingModel() {
		keys = append(keys, s.PendingModelKey)
	}

	return keys
}

func (s *Settings) PausedByBudget() bool {
	return s.Paused && s.PausedReason == PauseReasonBudget
}

func (s *Settings) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&s.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&s.MonthlyIndexingBudgetUSD, indexingBudgetRule),
		validation.Field(&s.PausedReason,
			validation.When(s.Paused,
				validation.Required.Error("Say why indexing is paused"),
			).Else(
				validation.Empty.Error("Only paused indexing has a reason"),
			),
			domainvalidation.ValidEnum[PauseReason]("Pause reason is invalid"),
		),
		validation.Field(&s.ActiveModelKey,
			validation.Length(0, MaxModelKeyChars).Error("A model key is at most 300 characters"),
			validation.When(s.Dimensions != 0,
				validation.Required.Error("An active model needs a model key"),
			),
		),
		validation.Field(&s.Dimensions,
			validation.When(s.ActiveModelKey != "",
				validation.Required.Error("An active model needs its dimensions"),
			),
			validation.By(validateDimensions),
		),
		validation.Field(&s.PendingModelKey,
			validation.Length(0, MaxModelKeyChars).Error("A model key is at most 300 characters"),
			validation.When(s.PendingDimensions != 0,
				validation.Required.Error("A pending model needs a model key"),
			),
			validation.When(s.PendingModelKey != "" && s.PendingModelKey == s.ActiveModelKey,
				validation.Empty.Error("The pending model is already the active one"),
			),
		),
		validation.Field(&s.PendingDimensions,
			validation.When(s.PendingModelKey != "",
				validation.Required.Error("A pending model needs its dimensions"),
			),
			validation.By(validateDimensions),
		),
	))

	if strings.TrimSpace(s.ActiveModelKey) != s.ActiveModelKey {
		multiErr.Add("activeModelKey", errortypes.ErrInvalid,
			"A model key cannot start or end with spaces")
	}
	if strings.TrimSpace(s.PendingModelKey) != s.PendingModelKey {
		multiErr.Add("pendingModelKey", errortypes.ErrInvalid,
			"A model key cannot start or end with spaces")
	}
}

func validateDimensions(value any) error {
	dimensions, ok := value.(int)
	if !ok || dimensions == 0 || IsAllowedDimension(dimensions) {
		return nil
	}

	return validation.NewError(
		"validation_embedding_dimensions",
		"Embedding dimensions must be 768, 1024 or 1536",
	)
}

func (s *Settings) GetID() pulid.ID { return s.ID }

func (s *Settings) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *Settings) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *Settings) GetTableName() string { return "ai_retrieval_settings" }

func (s *Settings) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("airs_")
		}
		s.CreatedAt = now
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}
