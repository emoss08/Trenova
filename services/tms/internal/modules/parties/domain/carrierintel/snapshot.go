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
	keel.Table    `                                                          keel:"carrier_intel_snapshots"`

	ID             pulid.ID         `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"                                                      keel:"id,pk,autoid:cisnap_"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"                                        keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID         `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"                                         keel:"organization_id,pk,tenant"`
	SubjectType    SubjectType      `json:"subjectType"    bun:"subject_type,type:VARCHAR(20),notnull"                                                keel:"subject_type"`
	SubjectID      string           `json:"subjectId"      bun:"subject_id,type:VARCHAR(100),notnull"                                                 keel:"subject_id"`
	CarrierID      pulid.ID         `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),nullzero"                                                keel:"carrier_id,nullzero"`
	DOTNumber      string           `json:"dotNumber"      bun:"dot_number,type:VARCHAR(12),notnull"                                                  keel:"dot_number"`
	DocketNumber   string           `json:"docketNumber"   bun:"docket_number,type:VARCHAR(12),nullzero"                                              keel:"docket_number,nullzero"`
	Provider       integration.Type `json:"provider"       bun:"provider,type:integration_type,notnull"                                               keel:"provider"`
	ProviderRef    string           `json:"providerRef"    bun:"provider_ref,type:VARCHAR(64),nullzero"                                               keel:"provider_ref,nullzero"`
	Depth          LookupDepth      `json:"depth"          bun:"depth,type:VARCHAR(10),notnull"                                                       keel:"depth"`
	Source         SnapshotSource   `json:"source"         bun:"source,type:VARCHAR(20),notnull"                                                      keel:"source"`
	IsCurrent      bool             `json:"isCurrent"      bun:"is_current,type:BOOLEAN,notnull"                                                      keel:"is_current"`
	NotFound       bool             `json:"notFound"       bun:"not_found,type:BOOLEAN,notnull"                                                       keel:"not_found"`
	Profile        *Profile         `json:"profile"        bun:"profile,type:JSONB,notnull"                                                           keel:"profile,jsonb"`
	Findings       []Finding        `json:"findings"       bun:"findings,type:JSONB,notnull"                                                          keel:"findings,jsonb"`
	BlockingCodes  []string         `json:"blockingCodes"  bun:"blocking_codes,type:TEXT[],array"                                                     keel:"blocking_codes"`
	AdvisoryCodes  []string         `json:"advisoryCodes"  bun:"advisory_codes,type:TEXT[],array"                                                     keel:"advisory_codes"`
	RiskLevel      RiskLevel        `json:"riskLevel"      bun:"risk_level,type:VARCHAR(20),notnull"                                                  keel:"risk_level"`
	ReviewState    ReviewState      `json:"reviewState"    bun:"review_state,type:VARCHAR(20),notnull"                                                keel:"review_state"`
	ReviewedByID   pulid.ID         `json:"reviewedById"   bun:"reviewed_by_id,type:VARCHAR(100),nullzero"                                            keel:"reviewed_by_id,nullzero"`
	ReviewedAt     *int64           `json:"reviewedAt"     bun:"reviewed_at,type:BIGINT,nullzero"                                                     keel:"reviewed_at"`
	ReviewNote     string           `json:"reviewNote"     bun:"review_note,type:TEXT,nullzero"                                                       keel:"review_note,nullzero"`
	PolicyVersion  int64            `json:"policyVersion"  bun:"policy_version,type:BIGINT,notnull"                                                   keel:"policy_version"`
	RawPayloadID   pulid.ID         `json:"rawPayloadId"   bun:"raw_payload_id,type:VARCHAR(100),nullzero"                                            keel:"raw_payload_id,nullzero"`
	ContentHash    string           `json:"contentHash"    bun:"content_hash,type:VARCHAR(64),notnull"                                                keel:"content_hash"`
	FetchedAt      int64            `json:"fetchedAt"      bun:"fetched_at,type:BIGINT,notnull"                                                       keel:"fetched_at"`
	FetchedDepth   LookupDepth      `json:"fetchedDepth"   bun:"fetched_depth,type:VARCHAR(10),notnull"                                               keel:"fetched_depth"`
	DepthFetchedAt int64            `json:"depthFetchedAt" bun:"depth_fetched_at,type:BIGINT,notnull"                                                 keel:"depth_fetched_at"`
	SourceAsOf     *int64           `json:"sourceAsOf"     bun:"source_as_of,type:BIGINT,nullzero"                                                    keel:"source_as_of"`
	ConfirmedAt    *int64           `json:"confirmedAt"    bun:"confirmed_at,type:BIGINT,nullzero"                                                    keel:"confirmed_at"`
	CreatedAt      int64            `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64            `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (s *CarrierIntelSnapshot) BeforeCreate(context.Context) error {
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

type CarrierIntelRawPayload struct {
	bun.BaseModel `bun:"table:carrier_intel_raw_payloads,alias:ciraw" json:"-"`
	keel.Table    `                                                            keel:"carrier_intel_raw_payloads"`

	ID             pulid.ID          `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"                                                      keel:"id,pk,autoid:ciraw_"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"                                        keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID          `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"                                         keel:"organization_id,pk,tenant"`
	Provider       integration.Type  `json:"provider"       bun:"provider,type:integration_type,notnull"                                               keel:"provider"`
	Endpoint       string            `json:"endpoint"       bun:"endpoint,type:VARCHAR(64),notnull"                                                    keel:"endpoint"`
	DOTNumber      string            `json:"dotNumber"      bun:"dot_number,type:VARCHAR(12),nullzero"                                                 keel:"dot_number,nullzero"`
	Payload        jsonutils.RawJSON `json:"-"              bun:"payload,type:JSONB,notnull"                                                           keel:"payload,jsonb"`
	ByteSize       int               `json:"byteSize"       bun:"byte_size,type:INTEGER,notnull"                                                       keel:"byte_size"`
	FetchedAt      int64             `json:"fetchedAt"      bun:"fetched_at,type:BIGINT,notnull"                                                       keel:"fetched_at"`
	ExpiresAt      int64             `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"                                                       keel:"expires_at"`
	CreatedAt      int64             `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
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

func (r *CarrierIntelRawPayload) BeforeCreate(context.Context) error {
	r.ByteSize = len(r.Payload)

	return nil
}

func SubjectKey(subjectType SubjectType, id string) string {
	return subjectType.String() + ":" + id
}

func ProspectSubjectID(dotNumber string) string {
	return "DOT" + strings.TrimSpace(dotNumber)
}
