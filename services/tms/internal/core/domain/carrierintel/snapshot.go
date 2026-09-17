package carrierintel

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*CarrierIntelSnapshot)(nil)
	_ bun.BeforeAppendModelHook = (*CarrierIntelRawPayload)(nil)
)

type CarrierIntelSnapshot struct {
	bun.BaseModel `bun:"table:carrier_intel_snapshots,alias:cisnap" json:"-"`

	ID             pulid.ID         `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID         `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	SubjectType    SubjectType      `json:"subjectType"    bun:"subject_type,type:VARCHAR(20),notnull"`
	SubjectID      string           `json:"subjectId"      bun:"subject_id,type:VARCHAR(100),notnull"`
	CarrierID      pulid.ID         `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),nullzero"`
	DOTNumber      string           `json:"dotNumber"      bun:"dot_number,type:VARCHAR(12),notnull"`
	DocketNumber   string           `json:"docketNumber"   bun:"docket_number,type:VARCHAR(12),nullzero"`
	Provider       integration.Type `json:"provider"       bun:"provider,type:integration_type,notnull"`
	ProviderRef    string           `json:"providerRef"    bun:"provider_ref,type:VARCHAR(64),nullzero"`
	Depth          LookupDepth      `json:"depth"          bun:"depth,type:VARCHAR(10),notnull"`
	Source         SnapshotSource   `json:"source"         bun:"source,type:VARCHAR(20),notnull"`
	IsCurrent      bool             `json:"isCurrent"      bun:"is_current,type:BOOLEAN,notnull"`
	NotFound       bool             `json:"notFound"       bun:"not_found,type:BOOLEAN,notnull"`
	Profile        *Profile         `json:"profile"        bun:"profile,type:JSONB,notnull"`
	Findings       []Finding        `json:"findings"       bun:"findings,type:JSONB,notnull"`
	BlockingCodes  []string         `json:"blockingCodes"  bun:"blocking_codes,type:TEXT[],array"`
	AdvisoryCodes  []string         `json:"advisoryCodes"  bun:"advisory_codes,type:TEXT[],array"`
	RiskLevel      RiskLevel        `json:"riskLevel"      bun:"risk_level,type:VARCHAR(20),notnull"`
	ReviewState    ReviewState      `json:"reviewState"    bun:"review_state,type:VARCHAR(20),notnull"`
	ReviewedByID   pulid.ID         `json:"reviewedById"   bun:"reviewed_by_id,type:VARCHAR(100),nullzero"`
	ReviewedAt     *int64           `json:"reviewedAt"     bun:"reviewed_at,type:BIGINT,nullzero"`
	ReviewNote     string           `json:"reviewNote"     bun:"review_note,type:TEXT,nullzero"`
	PolicyVersion  int64            `json:"policyVersion"  bun:"policy_version,type:BIGINT,notnull"`
	RawPayloadID   pulid.ID         `json:"rawPayloadId"   bun:"raw_payload_id,type:VARCHAR(100),nullzero"`
	ContentHash    string           `json:"contentHash"    bun:"content_hash,type:VARCHAR(64),notnull"`
	FetchedAt      int64            `json:"fetchedAt"      bun:"fetched_at,type:BIGINT,notnull"`
	FetchedDepth   LookupDepth      `json:"fetchedDepth"   bun:"fetched_depth,type:VARCHAR(10),notnull"`
	DepthFetchedAt int64            `json:"depthFetchedAt" bun:"depth_fetched_at,type:BIGINT,notnull"`
	SourceAsOf     *int64           `json:"sourceAsOf"     bun:"source_as_of,type:BIGINT,nullzero"`
	ConfirmedAt    *int64           `json:"confirmedAt"    bun:"confirmed_at,type:BIGINT,nullzero"`
	CreatedAt      int64            `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64            `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

func (s *CarrierIntelSnapshot) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("cisnap_")
		}
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
		s.CreatedAt = now
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}

type CarrierIntelRawPayload struct {
	bun.BaseModel `bun:"table:carrier_intel_raw_payloads,alias:ciraw" json:"-"`

	ID             pulid.ID          `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID          `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Provider       integration.Type  `json:"provider"       bun:"provider,type:integration_type,notnull"`
	Endpoint       string            `json:"endpoint"       bun:"endpoint,type:VARCHAR(64),notnull"`
	DOTNumber      string            `json:"dotNumber"      bun:"dot_number,type:VARCHAR(12),nullzero"`
	Payload        jsonutils.RawJSON `json:"-"              bun:"payload,type:JSONB,notnull"`
	ByteSize       int               `json:"byteSize"       bun:"byte_size,type:INTEGER,notnull"`
	FetchedAt      int64             `json:"fetchedAt"      bun:"fetched_at,type:BIGINT,notnull"`
	ExpiresAt      int64             `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"`
	CreatedAt      int64             `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (r *CarrierIntelRawPayload) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("ciraw_")
		}
		r.ByteSize = len(r.Payload)
		r.CreatedAt = timeutils.NowUnix()
	}
	return nil
}

func SubjectKey(subjectType SubjectType, id string) string {
	return subjectType.String() + ":" + id
}

func ProspectSubjectID(dotNumber string) string {
	return "DOT" + strings.TrimSpace(dotNumber)
}
