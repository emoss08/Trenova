package accountingsync

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

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
	ModelReviewed       string             `json:"modelReviewed,omitempty"`
}

func (s *MappingSignals) CandidateFingerprint() string {
	ids := make([]string, 0, len(s.Candidates))
	for idx := range s.Candidates {
		ids = append(ids, s.Candidates[idx].ExternalID)
	}
	slices.Sort(ids)
	return strings.Join(ids, ",")
}

type AccountingMapping struct {
	keel.Table `keel:"accounting_mappings"`

	ID              pulid.ID          `json:"id" keel:"id,pk,autoid:acctm_"`
	BusinessUnitID  pulid.ID          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID          `json:"organizationId" keel:"organization_id,pk,tenant"`
	ConnectionID    pulid.ID          `json:"connectionId" keel:"connection_id"`
	TargetType      MappingTargetType `json:"targetType" keel:"target_type"`
	TrenovaObjectID pulid.ID          `json:"trenovaObjectId" keel:"trenova_object_id,nullzero"`
	TrenovaKey      string            `json:"trenovaKey" keel:"trenova_key,nullzero"`
	TargetLabel     string            `json:"targetLabel" keel:"target_label"`
	SearchLabel     string            `json:"-" keel:"search_label"`
	ProviderKind    ReferenceKind     `json:"providerKind" keel:"provider_kind"`
	ExternalID      string            `json:"externalId" keel:"external_id,nullzero"`
	ExternalName    string            `json:"externalName" keel:"external_name,nullzero"`
	State           MappingState      `json:"state" keel:"state"`
	Source          MappingSource     `json:"source" keel:"source,nullzero"`
	Confidence      *float64          `json:"confidence" keel:"confidence"`
	Reason          string            `json:"reason" keel:"reason,nullzero"`
	Signals         MappingSignals    `json:"signals" keel:"signals,jsonb"`
	ConfirmedByID   pulid.ID          `json:"confirmedById" keel:"confirmed_by_id,nullzero"`
	ConfirmedAt     *int64            `json:"confirmedAt" keel:"confirmed_at"`
	Version         int64             `json:"version" keel:"version,version"`
	CreatedAt       int64             `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt       int64             `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	ConfirmedBy  *tenant.User         `json:"confirmedBy,omitempty" keel:"belongs:confirmed_by_id"`
}

func (m *AccountingMapping) BeforeUpdate(_ context.Context) error {
	m.SearchLabel = stringutils.NormalizeName(m.TargetLabel)

	return nil
}

func (m *AccountingMapping) BeforeCreate(_ context.Context) error {
	m.SearchLabel = stringutils.NormalizeName(m.TargetLabel)

	return nil
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
					"A mapping names an accounting record exactly when it is proposed or confirmed",
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

func (m *AccountingMapping) BeforeSave(_ context.Context) error {
	m.SearchLabel = stringutils.NormalizeName(m.TargetLabel)
	return nil
}

func (m *AccountingMapping) IsRequired() bool {
	return IsRequiredTarget(m.TargetType, m.TrenovaKey)
}

func IsRequiredTarget(target MappingTargetType, key string) bool {
	for idx := range providerProfiles {
		if providerProfiles[idx].IsRequiredTarget(target, key) {
			return true
		}
	}
	return false
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

func (m *AccountingMapping) Clone() *AccountingMapping {
	clone := *m
	clone.Signals.Matchers = slices.Clone(m.Signals.Matchers)
	clone.Signals.Candidates = slices.Clone(m.Signals.Candidates)
	clone.Signals.RejectedExternalIDs = slices.Clone(m.Signals.RejectedExternalIDs)
	if m.Confidence != nil {
		confidence := *m.Confidence
		clone.Confidence = &confidence
	}
	if m.ConfirmedAt != nil {
		confirmedAt := *m.ConfirmedAt
		clone.ConfirmedAt = &confirmedAt
	}

	return &clone
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
	if m.ExternalID != "" && !m.WasRejected(m.ExternalID) {
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
	m.ConfirmedByID = pulid.Nil
	m.ConfirmedAt = nil
	return nil
}

func (m *AccountingMapping) NeedsModelReview() bool {
	return m.Rescorable() &&
		m.State == MappingStateUnmatched &&
		len(m.Signals.Candidates) > 0 &&
		m.Signals.ModelReviewed != m.Signals.CandidateFingerprint()
}

func (m *AccountingMapping) MarkModelReviewed() {
	m.Signals.ModelReviewed = m.Signals.CandidateFingerprint()
}

func (m *AccountingMapping) KeepsModelPick(p *Proposal) bool {
	if m.Source != MappingSourceModel || m.State != MappingStateProposed || p.ExternalID != "" {
		return false
	}
	return slices.ContainsFunc(p.Candidates, func(c MappingCandidate) bool {
		return c.ExternalID == m.ExternalID
	})
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
