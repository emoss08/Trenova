package workerinjury

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var ErrInvalidSummaryStatus = errors.New("invalid osha summary status")

// The posting window for the 300A summary: February 1 to April 30 of the year
// after the one it covers (29 CFR 1904.32(b)(6)).
const (
	SummaryPostFromMonth    = time.February
	SummaryPostThroughMonth = time.April
	SummaryPostThroughDay   = 30
)

type OSHASummaryStatus string

const (
	SummaryDraft     = OSHASummaryStatus("Draft")
	SummaryCertified = OSHASummaryStatus("Certified")
)

func (s OSHASummaryStatus) String() string { return string(s) }

func (s OSHASummaryStatus) IsValid() bool {
	return s == SummaryDraft || s == SummaryCertified
}

var (
	_ validationframework.TenantedEntity = (*OSHAAnnualSummary)(nil)
)

// OSHAAnnualSummary is the 300A for one establishment and year. The
// organisation is the establishment: it carries one address, which is what OSHA
// means by one.
//
// The case totals are deliberately not columns here. They are derived from the
// injuries whenever the summary is read, so a case corrected in March — which
// the rule requires for five years — cannot leave a stale summary behind it.
type OSHAAnnualSummary struct {
	keel.Table `keel:"osha_annual_summaries"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:osum_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Year   int16             `json:"year"   keel:"year"`
	Status OSHASummaryStatus `json:"status" keel:"status"`

	NAICSCode        string `json:"naicsCode"        keel:"naics_code,nullzero"`
	AverageEmployees int32  `json:"averageEmployees" keel:"average_employees"`
	TotalHoursWorked int64  `json:"totalHoursWorked" keel:"total_hours_worked"`

	ExecutiveName  string `json:"executiveName"  keel:"executive_name,nullzero"`
	ExecutiveTitle string `json:"executiveTitle" keel:"executive_title,nullzero"`
	ExecutivePhone string `json:"executivePhone" keel:"executive_phone,nullzero"`

	CertifiedAt   *int64   `json:"certifiedAt"   keel:"certified_at"`
	CertifiedByID pulid.ID `json:"certifiedById" keel:"certified_by_id,nullzero"`
	PostedFrom    *int64   `json:"postedFrom"    keel:"posted_from"`
	PostedThrough *int64   `json:"postedThrough" keel:"posted_through"`

	SubmittedAt         *int64 `json:"submittedAt"         keel:"submitted_at"`
	SubmissionReference string `json:"submissionReference" keel:"submission_reference,nullzero"`
	Notes               string `json:"notes"               keel:"notes,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	CertifiedBy *tenant.User `json:"certifiedBy,omitempty" keel:"belongs:certified_by_id"`
}

func (s *OSHAAnnualSummary) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.Year, validation.Required.Error("Year is required")),
		validation.Field(&s.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[OSHASummaryStatus]("Status is not valid"),
		),
		validation.Field(&s.NAICSCode,
			validation.Length(0, 10).Error("NAICS code cannot exceed 10 characters"),
		),
		validation.Field(&s.ExecutiveName,
			validation.Length(0, 100).Error("Executive name cannot exceed 100 characters"),
		),
	))

	if s.AverageEmployees < 0 || s.TotalHoursWorked < 0 {
		multiErr.Add(
			"averageEmployees",
			errortypes.ErrInvalid,
			"Employment figures cannot be negative",
		)
	}

	// A certification is a company executive's signed statement that the
	// summary is true. Recording one without saying who signed it is a
	// signature with nobody behind it.
	if s.Status == SummaryCertified {
		if s.ExecutiveName == "" {
			multiErr.Add(
				"executiveName",
				errortypes.ErrRequired,
				"Name the company executive who certified the summary (29 CFR 1904.32(b)(3))",
			)
		}
		if s.AverageEmployees <= 0 {
			multiErr.Add(
				"averageEmployees",
				errortypes.ErrRequired,
				"The annual average number of employees is required on a certified summary",
			)
		}
		if s.TotalHoursWorked <= 0 {
			multiErr.Add(
				"totalHoursWorked",
				errortypes.ErrRequired,
				"Total hours worked is required on a certified summary",
			)
		}
	}
}

func (s *OSHAAnnualSummary) IsCertified() bool { return s.Status == SummaryCertified }

// PostingWindow is when the summary must be posted: February 1 to April 30 of
// the year after the one it covers.
func PostingWindow(year int16) (from int64, through int64) {
	start := time.Date(int(year)+1, SummaryPostFromMonth, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(
		int(year)+1,
		SummaryPostThroughMonth,
		SummaryPostThroughDay,
		23, 59, 59, 0,
		time.UTC,
	)
	return start.Unix(), end.Unix()
}

// YearBounds is the calendar year a case is counted in.
func YearBounds(year int16) (from int64, through int64) {
	start := time.Date(int(year), time.January, 1, 0, 0, 0, 0, time.UTC)
	return start.Unix(), start.AddDate(1, 0, 0).Unix()
}

func (s *OSHAAnnualSummary) GetID() pulid.ID { return s.ID }

func (s *OSHAAnnualSummary) GetCreatedAt() int64 { return s.CreatedAt }

func (s *OSHAAnnualSummary) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *OSHAAnnualSummary) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *OSHAAnnualSummary) GetTableName() string { return "osha_annual_summaries" }

func (s *OSHAAnnualSummary) GetResourceType() string { return "osha_annual_summary" }

func (s *OSHAAnnualSummary) GetResourceID() string { return s.ID.String() }

func (s *OSHAAnnualSummary) BeforeCreate(_ context.Context) error {
	if s.Status == "" {
		s.Status = SummaryDraft
	}

	return nil
}

func (OSHAAnnualSummary) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_osha_annual_summaries_year",
			Message: "A summary already exists for that year",
		},
	}
}

// OSHASummaryTotals is the boxed part of the 300A: the case counts and day
// totals, derived from the log rather than typed in.
type OSHASummaryTotals struct {
	Deaths               int
	DaysAwayCases        int
	JobTransferCases     int
	OtherRecordableCases int
	TotalRecordableCases int
	TotalDaysAway        int
	TotalDaysRestricted  int
	InjuryCount          int
	SkinDisorderCount    int
	RespiratoryCount     int
	PoisoningCount       int
	HearingLossCount     int
	OtherIllnessCount    int
	// OpenCases is not part of the 300A, but a summary with cases still
	// accruing days is not final, and whoever certifies it should know.
	OpenCases int
}

// TotalRecordableIncidentRate is the industry's headline number: recordable
// cases per 100 full-time workers a year, which is what 200,000 hours stands
// for. It is nil when no hours have been recorded, because a rate over zero
// hours is not a small number — it is not a number at all.
func (t OSHASummaryTotals) TotalRecordableIncidentRate(totalHours int64) *float64 {
	if totalHours <= 0 {
		return nil
	}
	rate := float64(t.TotalRecordableCases) * 200_000 / float64(totalHours)
	return &rate
}

// DaysAwayRestrictedRate is the DART rate: the cases serious enough to keep
// somebody off their job, per 100 full-time workers.
func (t OSHASummaryTotals) DaysAwayRestrictedRate(totalHours int64) *float64 {
	if totalHours <= 0 {
		return nil
	}
	rate := float64(t.DaysAwayCases+t.JobTransferCases) * 200_000 / float64(totalHours)
	return &rate
}

// BuildOSHASummaryTotals counts the log. Only recordable cases are counted:
// first aid and non-recordable cases are kept on file but are not the summary.
func BuildOSHASummaryTotals(injuries []*WorkerInjury) OSHASummaryTotals {
	totals := OSHASummaryTotals{}
	for _, injury := range injuries {
		if injury == nil || !injury.IsRecordable() {
			continue
		}
		totals.TotalRecordableCases++
		totals.TotalDaysAway += int(injury.DaysAway)
		totals.TotalDaysRestricted += int(injury.DaysRestricted)
		if injury.IsOpen() {
			totals.OpenCases++
		}

		switch injury.Classification {
		case CaseDeath:
			totals.Deaths++
		case CaseDaysAway:
			totals.DaysAwayCases++
		case CaseJobTransferOrRestriction:
			totals.JobTransferCases++
		case CaseOtherRecordable:
			totals.OtherRecordableCases++
		case CaseNotRecordable, CaseFirstAidOnly:
		}

		switch injury.IllnessType {
		case IllnessInjury:
			totals.InjuryCount++
		case IllnessSkinDisorder:
			totals.SkinDisorderCount++
		case IllnessRespiratoryCondition:
			totals.RespiratoryCount++
		case IllnessPoisoning:
			totals.PoisoningCount++
		case IllnessHearingLoss:
			totals.HearingLossCount++
		case IllnessOther:
			totals.OtherIllnessCount++
		}
	}
	return totals
}
