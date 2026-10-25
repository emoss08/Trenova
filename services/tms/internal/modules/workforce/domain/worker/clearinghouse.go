package worker

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	_ validationframework.TenantedEntity = (*WorkerClearinghouseQuery)(nil)
)

type WorkerClearinghouseQuery struct {
	keel.Table `keel:"worker_clearinghouse_queries"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wchq_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID `json:"workerId"       keel:"worker_id"`

	QueryType ClearinghouseQueryType `json:"queryType" keel:"query_type"`
	Result    ClearinghouseResult    `json:"result"    keel:"result"`

	ConsentObtainedAt *int64 `json:"consentObtainedAt" keel:"consent_obtained_at"`
	ConsentExpiresAt  *int64 `json:"consentExpiresAt"  keel:"consent_expires_at"`
	RequestedAt       int64  `json:"requestedAt"       keel:"requested_at"`
	CompletedAt       *int64 `json:"completedAt"       keel:"completed_at"`

	ViolationCount int32    `json:"violationCount" keel:"violation_count"`
	Reference      string   `json:"reference"      keel:"reference,nullzero"`
	DocumentID     pulid.ID `json:"documentId"     keel:"document_id,nullzero"`
	Notes          string   `json:"notes"          keel:"notes,nullzero"`
	PerformedByID  pulid.ID `json:"performedById"  keel:"performed_by_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Worker      *Worker            `json:"worker,omitempty"      keel:"belongs:worker_id"`
	Document    *document.Document `json:"document,omitempty"    keel:"belongs:document_id"`
	PerformedBy *tenant.User       `json:"performedBy,omitempty" keel:"belongs:performed_by_id"`
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

func (q *WorkerClearinghouseQuery) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if q.Result == "" {
		q.Result = ClearinghouseResultPending
	}
	if q.RequestedAt == 0 {
		q.RequestedAt = now
	}

	return nil
}
