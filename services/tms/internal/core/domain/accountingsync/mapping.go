package accountingsync

import (
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	PrecheckConfidence  = 0.95
	ConfidentConfidence = 0.70
	MaxModelConfidence  = 0.90
	MaxCandidates       = 3
	maxReasonLength     = 500
)

var (
	ErrMappingNotProposed = errors.New("only a proposed mapping can be rejected")
	ErrMappingNotSet      = errors.New("the mapping has no QuickBooks record to clear")
)

var _ bun.BeforeAppendModelHook = (*AccountingMapping)(nil)

type MappingCandidate struct {
	ExternalID string  `json:"externalId"`
	Name       string  `json:"name"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason"`
}

type MatcherSignal struct {
	Matcher string  `json:"matcher"`
	Score   float64 `json:"score"`
}

type MappingSignals struct {
	Matchers            []MatcherSignal    `json:"matchers,omitempty"`
	Candidates          []MappingCandidate `json:"candidates,omitempty"`
	RejectedExternalIDs []string           `json:"rejectedExternalIds,omitempty"`
}

type AccountingMapping struct {
	bun.BaseModel             `bun:"table:accounting_mappings,alias:acctm" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                json:"-"`

	ID              pulid.ID          `json:"id"              bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID  pulid.ID          `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID          `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ConnectionID    pulid.ID          `json:"connectionId"    bun:"connection_id,type:VARCHAR(100),notnull"`
	TargetType      MappingTargetType `json:"targetType"      bun:"target_type,type:VARCHAR(30),notnull"`
	TrenovaObjectID pulid.ID          `json:"trenovaObjectId" bun:"trenova_object_id,type:VARCHAR(100),nullzero"`
	TrenovaKey      string            `json:"trenovaKey"      bun:"trenova_key,type:VARCHAR(50),nullzero"`
	TargetLabel     string            `json:"targetLabel"     bun:"target_label,type:VARCHAR(500),notnull"`
	SearchLabel     string            `json:"-"               bun:"search_label,type:VARCHAR(500),notnull"`
	ProviderKind    ReferenceKind     `json:"providerKind"    bun:"provider_kind,type:VARCHAR(30),notnull"`
	ExternalID      string            `json:"externalId"      bun:"external_id,type:VARCHAR(100),nullzero"`
	ExternalName    string            `json:"externalName"    bun:"external_name,type:VARCHAR(1000),nullzero"`
	State           MappingState      `json:"state"           bun:"state,type:VARCHAR(20),notnull"`
	Source          MappingSource     `json:"source"          bun:"source,type:VARCHAR(30),nullzero"`
	Confidence      *float64          `json:"confidence"      bun:"confidence,type:NUMERIC(5,4),nullzero"`
	Reason          string            `json:"reason"          bun:"reason,type:TEXT,nullzero"`
	Signals         MappingSignals    `json:"signals"         bun:"signals,type:JSONB,notnull"`
	ConfirmedByID   pulid.ID          `json:"confirmedById"   bun:"confirmed_by_id,type:VARCHAR(100),nullzero"`
	ConfirmedAt     *int64            `json:"confirmedAt"     bun:"confirmed_at,type:BIGINT,nullzero"`
	Version         int64             `json:"version"         bun:"version,type:BIGINT"`
	CreatedAt       int64             `json:"createdAt"       bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt       int64             `json:"updatedAt"       bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	ConfirmedBy  *tenant.User         `json:"confirmedBy,omitempty"  bun:"rel:belongs-to,join:confirmed_by_id=id"`
}

type Proposal struct {
	ExternalID   string
	ExternalName string
	Source       MappingSource
	Confidence   float64
	Reason       string
	Matchers     []MatcherSignal
	Candidates   []MappingCandidate
}

type Choice struct {
	ExternalID   string
	ExternalName string
	Source       MappingSource
	Reason       string
	ActorID      pulid.ID
	At           int64
}

func (m *AccountingMapping) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.ConnectionID, validation.Required.Error("Connection is required")),
		validation.Field(&m.TargetType,
			validation.Required.Error("Target type is required"),
			domainvalidation.ValidEnum[MappingTargetType]("Target type is not recognized"),
		),
		validation.Field(&m.TrenovaObjectID, validation.By(func(any) error {
			if m.TargetType.KeyedByObject() && m.TrenovaObjectID.IsNil() {
				return validation.NewError(
					"required",
					"This mapping needs the Trenova record it maps",
				)
			}
			if !m.TargetType.KeyedByObject() && !m.TrenovaObjectID.IsNil() {
				return validation.NewError(
					"invalid",
					"This mapping is keyed by value, not by record",
				)
			}
			return nil
		})),
		validation.Field(&m.TrenovaKey, validation.By(func(any) error {
			if m.TargetType.KeyedByObject() {
				if m.TrenovaKey != "" {
					return validation.NewError(
						"invalid",
						"This mapping is keyed by record, not by value",
					)
				}
				return nil
			}
			if !m.TargetType.AcceptsKey(m.TrenovaKey) {
				return validation.NewError("invalid", "This value cannot be mapped")
			}
			return nil
		})),
		validation.Field(&m.State,
			validation.Required.Error("State is required"),
			domainvalidation.ValidEnum[MappingState]("State is not recognized"),
		),
		validation.Field(&m.Source,
			domainvalidation.ValidEnum[MappingSource]("Source is not recognized"),
		),
		validation.Field(&m.ExternalID, validation.By(func(any) error {
			if (m.State == MappingStateUnmatched) != (m.ExternalID == "") {
				return validation.NewError(
					"invalid",
					"A mapping names a QuickBooks record exactly when it is proposed or confirmed",
				)
			}
			return nil
		})),
		validation.Field(&m.Confidence, validation.By(func(any) error {
			if m.Confidence != nil && (*m.Confidence < 0 || *m.Confidence > 1) {
				return validation.NewError("invalid", "Confidence must be between 0 and 1")
			}
			return nil
		})),
	))
}

func (m *AccountingMapping) GetTableName() string { return "accounting_mappings" }

func (m *AccountingMapping) GetID() pulid.ID { return m.ID }

func (m *AccountingMapping) GetOrganizationID() pulid.ID { return m.OrganizationID }

func (m *AccountingMapping) GetBusinessUnitID() pulid.ID { return m.BusinessUnitID }

func (m *AccountingMapping) GetCreatedAt() int64 { return m.CreatedAt }

func (m *AccountingMapping) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "acctm",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "target_label",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "external_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (m *AccountingMapping) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	m.SearchLabel = stringutils.NormalizeName(m.TargetLabel)
	if m.ProviderKind == "" {
		m.ProviderKind = m.TargetType.ProviderKind()
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("acctm_")
		}
		m.CreatedAt = now
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}

	return nil
}

func (m *AccountingMapping) IsRequired() bool {
	return IsRequiredTarget(m.TargetType, m.TrenovaKey)
}

func IsRequiredTarget(target MappingTargetType, key string) bool {
	switch target {
	case TargetAccountRole:
		return key == AccountRoleAR || key == AccountRoleRevenue || key == AccountRoleDeposit
	case TargetLineType:
		return key == string(invoice.InvoiceLineTypeFreight)
	case TargetAccessorialCharge,
		TargetItemRole,
		TargetCustomer,
		TargetCarrier,
		TargetPaymentTerm,
		TargetPaymentMethod:
		return false
	default:
		return false
	}
}

func (m *AccountingMapping) Rescorable() bool {
	if m.State == MappingStateConfirmed {
		return false
	}
	return !m.Source.Deliberate()
}

func (m *AccountingMapping) WasRejected(externalID string) bool {
	return slices.Contains(m.Signals.RejectedExternalIDs, externalID)
}

func (m *AccountingMapping) ApplyProposal(p *Proposal) bool {
	if !m.Rescorable() {
		return false
	}

	candidates := make([]MappingCandidate, 0, min(len(p.Candidates), MaxCandidates))
	for idx := range p.Candidates {
		if len(candidates) == MaxCandidates {
			break
		}
		if !m.WasRejected(p.Candidates[idx].ExternalID) {
			candidates = append(candidates, p.Candidates[idx])
		}
	}

	next := MappingStateUnmatched
	externalID, externalName, reason := "", "", stringutils.TruncateRunes(p.Reason, maxReasonLength)
	var confidence *float64
	var source MappingSource
	if p.ExternalID != "" && !m.WasRejected(p.ExternalID) {
		next = MappingStateProposed
		externalID, externalName, source = p.ExternalID, p.ExternalName, p.Source
		score := p.Confidence
		if source == MappingSourceModel {
			score = min(score, MaxModelConfidence)
		}
		confidence = &score
	}

	changed := m.State != next ||
		m.ExternalID != externalID ||
		m.ExternalName != externalName ||
		m.Source != source ||
		m.Reason != reason ||
		!sameConfidence(m.Confidence, confidence) ||
		!slices.Equal(m.Signals.Candidates, candidates) ||
		!slices.Equal(m.Signals.Matchers, p.Matchers)

	m.State = next
	m.ExternalID = externalID
	m.ExternalName = externalName
	m.Source = source
	m.Confidence = confidence
	m.Reason = reason
	m.Signals.Candidates = candidates
	m.Signals.Matchers = p.Matchers

	return changed
}

func (m *AccountingMapping) Confirm(choice *Choice) {
	m.State = MappingStateConfirmed
	m.ExternalID = choice.ExternalID
	m.ExternalName = choice.ExternalName
	m.Source = choice.Source
	if choice.Reason != "" {
		m.Reason = stringutils.TruncateRunes(choice.Reason, maxReasonLength)
	}
	m.ConfirmedByID = choice.ActorID
	at := choice.At
	m.ConfirmedAt = &at
	m.Signals.RejectedExternalIDs = slices.DeleteFunc(
		m.Signals.RejectedExternalIDs,
		func(id string) bool { return id == choice.ExternalID },
	)
}

func (m *AccountingMapping) Reject() error {
	if m.State != MappingStateProposed {
		return ErrMappingNotProposed
	}
	if !m.WasRejected(m.ExternalID) {
		m.Signals.RejectedExternalIDs = append(m.Signals.RejectedExternalIDs, m.ExternalID)
	}
	m.Signals.Candidates = slices.DeleteFunc(
		m.Signals.Candidates,
		func(c MappingCandidate) bool { return c.ExternalID == m.ExternalID },
	)
	m.State = MappingStateUnmatched
	m.ExternalID = ""
	m.ExternalName = ""
	m.Source = ""
	m.Confidence = nil
	m.Reason = ""
	return nil
}

func (m *AccountingMapping) Clear() error {
	if m.State == MappingStateUnmatched {
		return ErrMappingNotSet
	}
	m.State = MappingStateUnmatched
	m.ExternalID = ""
	m.ExternalName = ""
	m.Source = ""
	m.Confidence = nil
	m.Reason = ""
	m.ConfirmedByID = pulid.Nil
	m.ConfirmedAt = nil
	return nil
}

func (m *AccountingMapping) Prechecked() bool {
	return m.State == MappingStateProposed &&
		m.Confidence != nil &&
		*m.Confidence >= PrecheckConfidence
}

func sameConfidence(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
