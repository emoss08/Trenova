package carrierintel

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CarrierIntelSnapshot struct {
	keel.Table `                                                          keel:"carrier_intel_snapshots"`

	ID             pulid.ID         `json:"id" keel:"id,pk,autoid:cisnap_"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	SubjectType    SubjectType      `json:"subjectType" keel:"subject_type"`
	SubjectID      string           `json:"subjectId" keel:"subject_id"`
	CarrierID      pulid.ID         `json:"carrierId" keel:"carrier_id,nullzero"`
	DOTNumber      string           `json:"dotNumber" keel:"dot_number"`
	DocketNumber   string           `json:"docketNumber" keel:"docket_number,nullzero"`
	Provider       integration.Type `json:"provider" keel:"provider"`
	ProviderRef    string           `json:"providerRef" keel:"provider_ref,nullzero"`
	Depth          LookupDepth      `json:"depth" keel:"depth"`
	Source         SnapshotSource   `json:"source" keel:"source"`
	IsCurrent      bool             `json:"isCurrent" keel:"is_current"`
	NotFound       bool             `json:"notFound" keel:"not_found"`
	Profile        *Profile         `json:"profile" keel:"profile,jsonb"`
	Findings       []Finding        `json:"findings" keel:"findings,jsonb"`
	BlockingCodes  []string         `json:"blockingCodes" keel:"blocking_codes"`
	AdvisoryCodes  []string         `json:"advisoryCodes" keel:"advisory_codes"`
	RiskLevel      RiskLevel        `json:"riskLevel" keel:"risk_level"`
	ReviewState    ReviewState      `json:"reviewState" keel:"review_state"`
	ReviewedByID   pulid.ID         `json:"reviewedById" keel:"reviewed_by_id,nullzero"`
	ReviewedAt     *int64           `json:"reviewedAt" keel:"reviewed_at"`
	ReviewNote     string           `json:"reviewNote" keel:"review_note,nullzero"`
	PolicyVersion  int64            `json:"policyVersion" keel:"policy_version"`
	RawPayloadID   pulid.ID         `json:"rawPayloadId" keel:"raw_payload_id,nullzero"`
	ContentHash    string           `json:"contentHash" keel:"content_hash"`
	FetchedAt      int64            `json:"fetchedAt" keel:"fetched_at"`
	FetchedDepth   LookupDepth      `json:"fetchedDepth" keel:"fetched_depth"`
	DepthFetchedAt int64            `json:"depthFetchedAt" keel:"depth_fetched_at"`
	SourceAsOf     *int64           `json:"sourceAsOf" keel:"source_as_of"`
	ConfirmedAt    *int64           `json:"confirmedAt" keel:"confirmed_at"`
	CreatedAt      int64            `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64            `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (s *CarrierIntelSnapshot) BeforeCreate(_ context.Context) error {
	if s.Findings == nil {
		s.Findings = []Finding{}
	}
	if s.Profile == nil {
		s.Profile = &Profile{Coverage: []Section{}}
	}
	if s.RiskLevel == "" {
		s.RiskLevel = RiskLevelUnknown
	}
	if s.ReviewState == "" {
		s.ReviewState = ReviewStateNone
	}
	if !s.FetchedDepth.IsValid() {
		s.FetchedDepth = s.Depth
	}
	if s.DepthFetchedAt <= 0 {
		s.DepthFetchedAt = s.FetchedAt
	}

	return nil
}

func (s *CarrierIntelSnapshot) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.SubjectType,
			validation.Required.Error("Subject type is required"),
			domainvalidation.ValidEnum[SubjectType]("Subject type is invalid"),
		),
		validation.Field(&s.SubjectID, validation.Required.Error("Subject is required")),
		validation.Field(&s.DOTNumber,
			validation.Required.Error("DOT number is required"),
			validation.Length(1, 12).Error("DOT number must be at most 12 characters"),
		),
		validation.Field(&s.Depth,
			validation.Required.Error("Depth is required"),
			domainvalidation.ValidEnum[LookupDepth]("Depth is invalid"),
		),
		validation.Field(&s.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[SnapshotSource]("Source is invalid"),
		),
	))
	if !s.Provider.SupportsCarrierIntelligence() {
		multiErr.Add(
			"provider",
			errortypes.ErrInvalid,
			"Provider is not a carrier intelligence provider",
		)
	}
}

func (s *CarrierIntelSnapshot) EffectiveAsOf() int64 {
	if s.ConfirmedAt != nil && *s.ConfirmedAt > s.FetchedAt {
		return *s.ConfirmedAt
	}
	return s.FetchedAt
}

func (s *CarrierIntelSnapshot) IsFresh(now, maxAgeSeconds int64) bool {
	return now-s.EffectiveAsOf() <= maxAgeSeconds
}

func (s *CarrierIntelSnapshot) DepthAsOf() int64 {
	if s.DepthFetchedAt <= 0 {
		return s.EffectiveAsOf()
	}
	if s.FetchedDepth.Satisfies(s.Depth) {
		return max(s.DepthFetchedAt, s.EffectiveAsOf())
	}
	return s.DepthFetchedAt
}

func (s *CarrierIntelSnapshot) DepthIsFresh(now, maxAgeSeconds int64) bool {
	return now-s.DepthAsOf() <= maxAgeSeconds
}

func (s *CarrierIntelSnapshot) AsOfForDepth(wanted LookupDepth) int64 {
	if !s.FetchedDepth.IsValid() || s.FetchedDepth.Satisfies(wanted) {
		return s.EffectiveAsOf()
	}
	return s.DepthAsOf()
}

func (s *CarrierIntelSnapshot) IsFreshForDepth(now, maxAgeSeconds int64, wanted LookupDepth) bool {
	return now-s.AsOfForDepth(wanted) <= maxAgeSeconds
}

func (s *CarrierIntelSnapshot) BlockingFindings() []Finding {
	out := make([]Finding, 0, len(s.BlockingCodes))
	for idx := range s.Findings {
		if s.Findings[idx].IsBlocking() {
			out = append(out, s.Findings[idx])
		}
	}
	return out
}

func (s *CarrierIntelSnapshot) ApplyFindings(findings []Finding, policyVersion int64) {
	s.Findings = findings
	s.BlockingCodes = BlockingCodes(findings)
	s.AdvisoryCodes = AdvisoryCodes(findings)
	s.RiskLevel = s.Profile.DeriveRiskLevel(findings)
	if s.NotFound {
		s.RiskLevel = RiskLevelUnknown
	}
	s.PolicyVersion = policyVersion
}

func (s *CarrierIntelSnapshot) GetID() pulid.ID { return s.ID }

func (s *CarrierIntelSnapshot) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *CarrierIntelSnapshot) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *CarrierIntelSnapshot) GetTableName() string { return "carrier_intel_snapshots" }

type CarrierIntelRawPayload struct {
	keel.Table `                                                            keel:"carrier_intel_raw_payloads"`

	ID             pulid.ID          `json:"id" keel:"id,pk,autoid:ciraw_"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID          `json:"organizationId" keel:"organization_id,pk,tenant"`
	Provider       integration.Type  `json:"provider" keel:"provider"`
	Endpoint       string            `json:"endpoint" keel:"endpoint"`
	DOTNumber      string            `json:"dotNumber" keel:"dot_number,nullzero"`
	Payload        jsonutils.RawJSON `json:"-" keel:"payload,jsonb"`
	ByteSize       int               `json:"byteSize" keel:"byte_size"`
	FetchedAt      int64             `json:"fetchedAt" keel:"fetched_at"`
	ExpiresAt      int64             `json:"expiresAt" keel:"expires_at"`
	CreatedAt      int64             `json:"createdAt" keel:"created_at,autocreate"`
}

func (r *CarrierIntelRawPayload) BeforeCreate(_ context.Context) error {
	r.ByteSize = len(r.Payload)

	return nil
}

func SubjectKey(subjectType SubjectType, id string) string {
	return subjectType.String() + ":" + id
}

func ProspectSubjectID(dotNumber string) string {
	return "DOT" + strings.TrimSpace(dotNumber)
}
