package worker

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidClearinghouseQueryType = errors.New("invalid clearinghouse query type")
	ErrInvalidClearinghouseResult    = errors.New("invalid clearinghouse result")
)

// ClearinghouseQueryIntervalMonths is how often a limited query must be run for
// every driver the employer uses (49 CFR 382.701(b)).
const ClearinghouseQueryIntervalMonths = 12

type ClearinghouseQueryType string

const (
	ClearinghouseQueryPreEmploymentFull = ClearinghouseQueryType("PreEmploymentFull")
	ClearinghouseQueryAnnualLimited     = ClearinghouseQueryType("AnnualLimited")
	ClearinghouseQueryFull              = ClearinghouseQueryType("Full")
	ClearinghouseQueryLimited           = ClearinghouseQueryType("Limited")
)

func (t ClearinghouseQueryType) String() string { return string(t) }

func (t ClearinghouseQueryType) IsValid() bool {
	switch t {
	case ClearinghouseQueryPreEmploymentFull, ClearinghouseQueryAnnualLimited,
		ClearinghouseQueryFull, ClearinghouseQueryLimited:
		return true
	default:
		return false
	}
}

func (t ClearinghouseQueryType) Label() string {
	switch t {
	case ClearinghouseQueryPreEmploymentFull:
		return "Pre-employment full query"
	case ClearinghouseQueryAnnualLimited:
		return "Annual limited query"
	case ClearinghouseQueryFull:
		return "Full query"
	case ClearinghouseQueryLimited:
		return "Limited query"
	default:
		return string(t)
	}
}

// IsFull reports whether the query returns the detail of any violation rather
// than only whether one exists. A full query needs the driver's specific
// electronic consent in the Clearinghouse itself.
func (t ClearinghouseQueryType) IsFull() bool {
	return t == ClearinghouseQueryPreEmploymentFull || t == ClearinghouseQueryFull
}

type ClearinghouseResult string

const (
	ClearinghouseResultPending         = ClearinghouseResult("Pending")
	ClearinghouseResultNoViolations    = ClearinghouseResult("NoViolations")
	ClearinghouseResultViolationsFound = ClearinghouseResult("ViolationsFound")
	ClearinghouseResultConsentDenied   = ClearinghouseResult("ConsentDenied")
)

func (r ClearinghouseResult) String() string { return string(r) }

func (r ClearinghouseResult) IsValid() bool {
	switch r {
	case ClearinghouseResultPending, ClearinghouseResultNoViolations,
		ClearinghouseResultViolationsFound, ClearinghouseResultConsentDenied:
		return true
	default:
		return false
	}
}

// Prohibits reports whether the answer bars the driver from safety-sensitive
// duty. Refusing consent is a prohibition in its own right: without consent the
// employer may not use the driver (49 CFR 382.701(a)(3)).
func (r ClearinghouseResult) Prohibits() bool {
	return r == ClearinghouseResultViolationsFound || r == ClearinghouseResultConsentDenied
}

func (r ClearinghouseResult) IsResolved() bool { return r != ClearinghouseResultPending }

var (
	_ bun.BeforeAppendModelHook          = (*WorkerClearinghouseQuery)(nil)
	_ validationframework.TenantedEntity = (*WorkerClearinghouseQuery)(nil)
)

type WorkerClearinghouseQuery struct {
	bun.BaseModel `bun:"table:worker_clearinghouse_queries,alias:wchq" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`

	QueryType ClearinghouseQueryType `json:"queryType" bun:"query_type,type:dot_clearinghouse_query_type_enum,notnull"`
	Result    ClearinghouseResult    `json:"result"    bun:"result,type:dot_clearinghouse_result_enum,notnull,default:'Pending'"`

	ConsentObtainedAt *int64 `json:"consentObtainedAt" bun:"consent_obtained_at,type:BIGINT,nullzero"`
	ConsentExpiresAt  *int64 `json:"consentExpiresAt"  bun:"consent_expires_at,type:BIGINT,nullzero"`
	RequestedAt       int64  `json:"requestedAt"       bun:"requested_at,type:BIGINT,notnull"`
	CompletedAt       *int64 `json:"completedAt"       bun:"completed_at,type:BIGINT,nullzero"`

	ViolationCount int32    `json:"violationCount" bun:"violation_count,type:INTEGER,notnull"`
	Reference      string   `json:"reference"      bun:"reference,type:VARCHAR(100),nullzero"`
	DocumentID     pulid.ID `json:"documentId"     bun:"document_id,type:VARCHAR(100),nullzero"`
	Notes          string   `json:"notes"          bun:"notes,type:TEXT,nullzero"`
	PerformedByID  pulid.ID `json:"performedById"  bun:"performed_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker      *Worker            `json:"worker,omitempty"      bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document    *document.Document `json:"document,omitempty"    bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	PerformedBy *tenant.User       `json:"performedBy,omitempty" bun:"rel:belongs-to,join:performed_by_id=id"`
}

func (q *WorkerClearinghouseQuery) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(q,
		validation.Field(&q.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&q.QueryType,
			validation.Required.Error("Query type is required"),
			domainvalidation.ValidEnum[ClearinghouseQueryType]("Query type is not valid"),
		),
		validation.Field(&q.Result,
			validation.Required.Error("Result is required"),
			domainvalidation.ValidEnum[ClearinghouseResult]("Result is not valid"),
		),
		validation.Field(&q.RequestedAt, validation.Required.Error("Request date is required")),
		validation.Field(&q.Reference,
			validation.Length(0, 100).Error("Reference cannot exceed 100 characters"),
		),
	))

	// A full query cannot be run without the driver's consent in the
	// Clearinghouse, so a record of one without a consent date is a record of
	// something that could not have happened.
	if q.QueryType.IsFull() && (q.ConsentObtainedAt == nil || *q.ConsentObtainedAt <= 0) {
		multiErr.Add(
			"consentObtainedAt",
			errortypes.ErrRequired,
			"A full query requires the driver's electronic consent (49 CFR 382.701)",
		)
	}

	if q.ConsentObtainedAt != nil && q.ConsentExpiresAt != nil &&
		*q.ConsentExpiresAt <= *q.ConsentObtainedAt {
		multiErr.Add(
			"consentExpiresAt",
			errortypes.ErrInvalid,
			"Consent expiry must be after the consent date",
		)
	}

	if q.CompletedAt != nil && *q.CompletedAt < q.RequestedAt {
		multiErr.Add(
			"completedAt",
			errortypes.ErrInvalid,
			"The answer cannot pre-date the query",
		)
	}

	if q.Result.IsResolved() && (q.CompletedAt == nil || *q.CompletedAt <= 0) {
		multiErr.Add(
			"completedAt",
			errortypes.ErrRequired,
			"An answered query must record when the answer arrived",
		)
	}

	if q.Result == ClearinghouseResultViolationsFound && q.ViolationCount <= 0 {
		multiErr.Add(
			"violationCount",
			errortypes.ErrRequired,
			"Record how many violations the query returned",
		)
	}
}

func (q *WorkerClearinghouseQuery) IsPending() bool {
	return q.Result == ClearinghouseResultPending
}

// NextDueAt is when the next limited query falls due after this one was
// answered. A query that has not come back yet sets no clock.
func (q *WorkerClearinghouseQuery) NextDueAt() *int64 {
	if q.CompletedAt == nil || *q.CompletedAt <= 0 || !q.Result.IsResolved() {
		return nil
	}
	due := timeutils.AddMonthsUTC(*q.CompletedAt, ClearinghouseQueryIntervalMonths)
	return &due
}

func (q *WorkerClearinghouseQuery) GetID() pulid.ID { return q.ID }

func (q *WorkerClearinghouseQuery) GetCreatedAt() int64 { return q.CreatedAt }

func (q *WorkerClearinghouseQuery) GetOrganizationID() pulid.ID { return q.OrganizationID }

func (q *WorkerClearinghouseQuery) GetBusinessUnitID() pulid.ID { return q.BusinessUnitID }

func (q *WorkerClearinghouseQuery) GetTableName() string { return "worker_clearinghouse_queries" }

func (q *WorkerClearinghouseQuery) GetResourceType() string { return "worker_clearinghouse_query" }

func (q *WorkerClearinghouseQuery) GetResourceID() string { return q.ID.String() }

func (q *WorkerClearinghouseQuery) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if q.ID.IsNil() {
			q.ID = pulid.MustNew("wchq_")
		}
		if q.Result == "" {
			q.Result = ClearinghouseResultPending
		}
		if q.RequestedAt == 0 {
			q.RequestedAt = now
		}
		q.CreatedAt = now
		q.UpdatedAt = now
	case *bun.UpdateQuery:
		q.UpdatedAt = now
	}

	return nil
}
