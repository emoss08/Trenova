package workerinjuryservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// OSHALog is the 300 log for one year and the 300A totals over it. The totals
// are counted from the cases on every read rather than stored: 29 CFR 1904.33
// requires the log be corrected for five years, and a stored total would go
// stale the first time somebody did.
type OSHALog struct {
	Year    int16
	Cases   []*worker.WorkerInjury
	Totals  worker.OSHASummaryTotals
	Summary *worker.OSHAAnnualSummary
	// PostFrom and PostThrough are the February 1 – April 30 window the
	// summary has to be posted in.
	PostFrom    int64
	PostThrough int64
	// TotalRecordableIncidentRate and DaysAwayRestrictedRate are nil until the
	// summary records hours worked, because a rate over zero hours is not a
	// number.
	TotalRecordableIncidentRate *float64
	DaysAwayRestrictedRate      *float64
}

// Log reads one year of the log. Non-recordable cases are included so the
// office can see the decisions that were made, and excluded from the totals,
// which is what the 300A counts.
func (s *Service) Log(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	year int16,
) (*OSHALog, error) {
	if year <= 0 {
		year = int16(timeutils.YearOfUnix(timeutils.NowUnix()))
	}

	cases, err := s.repo.ListInjuries(ctx, &repositories.ListWorkerInjuriesRequest{
		TenantInfo:    tenantInfo,
		CaseYear:      year,
		IncludeWorker: true,
	})
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.GetSummary(ctx, &repositories.GetOSHASummaryRequest{
		TenantInfo: tenantInfo,
		Year:       year,
	})
	if err != nil {
		return nil, err
	}

	from, through := worker.PostingWindow(year)
	log := &OSHALog{
		Year:        year,
		Cases:       cases,
		Totals:      worker.BuildOSHASummaryTotals(cases),
		Summary:     summary,
		PostFrom:    from,
		PostThrough: through,
	}
	if summary != nil {
		log.TotalRecordableIncidentRate = log.Totals.TotalRecordableIncidentRate(
			summary.TotalHoursWorked,
		)
		log.DaysAwayRestrictedRate = log.Totals.DaysAwayRestrictedRate(summary.TotalHoursWorked)
	}

	return log, nil
}

func (s *Service) ListSummaries(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.OSHAAnnualSummary, error) {
	return s.repo.ListSummaries(ctx, &repositories.ListOSHASummariesRequest{
		TenantInfo: tenantInfo,
	})
}

// SaveSummaryRequest carries the establishment figures the 300A needs and the
// log cannot supply: how many people worked there and for how many hours.
type SaveSummaryRequest struct {
	TenantInfo          pagination.TenantInfo
	Year                int16
	NAICSCode           *string
	AverageEmployees    *int32
	TotalHoursWorked    *int64
	ExecutiveName       *string
	ExecutiveTitle      *string
	ExecutivePhone      *string
	PostedFrom          *int64
	PostedThrough       *int64
	SubmittedAt         *int64
	SubmissionReference *string
	Notes               *string
	UserID              pulid.ID
}

// SaveSummary creates or updates the year's summary. A certified summary is
// deliberately not editable here: correcting one means uncertifying it first,
// so nobody quietly changes a figure an executive has signed for.
func (s *Service) SaveSummary(
	ctx context.Context,
	req *SaveSummaryRequest,
) (*worker.OSHAAnnualSummary, error) {
	existing, err := s.repo.GetSummary(ctx, &repositories.GetOSHASummaryRequest{
		TenantInfo: req.TenantInfo,
		Year:       req.Year,
	})
	if err != nil {
		return nil, err
	}

	if existing != nil && existing.IsCertified() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"This summary is certified; uncertify it before changing a figure",
		)
	}

	entity := existing
	creating := entity == nil
	if creating {
		from, through := worker.PostingWindow(req.Year)
		entity = &worker.OSHAAnnualSummary{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			Year:           req.Year,
			Status:         worker.SummaryDraft,
			PostedFrom:     &from,
			PostedThrough:  &through,
		}
	}

	previous := *entity
	applySummaryUpdate(entity, req)

	if err = s.prepareSummary(entity); err != nil {
		return nil, err
	}

	var saved *worker.OSHAAnnualSummary
	if creating {
		saved, err = s.repo.CreateSummary(ctx, entity)
	} else {
		saved, err = s.repo.UpdateSummary(ctx, entity)
	}
	if err != nil {
		return nil, err
	}

	params := &auditParams{
		resourceID: saved.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    saved,
		previous:   &previous,
		comment:    "Updated the " + strconv.Itoa(int(saved.Year)) + " OSHA 300A summary",
	}
	if creating {
		params.operation = permission.OpCreate
		params.previous = nil
		params.comment = "Started the " + strconv.Itoa(int(saved.Year)) + " OSHA 300A summary"
	}
	s.audit(params)
	s.publish(ctx, req.TenantInfo, realtimeSummary, params.operation, saved.ID, req.UserID)

	return saved, nil
}

// CertifySummary records a company executive's statement that the summary is
// true (29 CFR 1904.32(b)(3)). It refuses while cases are still accruing days:
// certifying a log that has not finished moving is certifying a number that is
// about to change.
func (s *Service) CertifySummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	year int16,
	userID pulid.ID,
) (*worker.OSHAAnnualSummary, error) {
	entity, err := s.repo.GetSummary(ctx, &repositories.GetOSHASummaryRequest{
		TenantInfo: tenantInfo,
		Year:       year,
	})
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, errortypes.NewValidationError(
			"year",
			errortypes.ErrRequired,
			"Start the summary and record its employment figures before certifying it",
		)
	}
	if entity.IsCertified() {
		return entity, nil
	}

	cases, err := s.repo.ListInjuries(ctx, &repositories.ListWorkerInjuriesRequest{
		TenantInfo: tenantInfo,
		CaseYear:   year,
	})
	if err != nil {
		return nil, err
	}
	if open := worker.BuildOSHASummaryTotals(cases).OpenCases; open > 0 {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Close the {0} case(s) still accruing days before certifying the summary",
			strconv.Itoa(open),
		)
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.Status = worker.SummaryCertified
	entity.CertifiedAt = &now
	entity.CertifiedByID = userID

	if err = s.prepareSummary(entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateSummary(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpManage,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment: "Certified the " + strconv.Itoa(int(updated.Year)) +
			" OSHA 300A summary",
	})
	s.publish(ctx, tenantInfo, realtimeSummary, permission.OpManage, updated.ID, userID)

	return updated, nil
}

// UncertifySummary reopens a certified summary so a correction can be made.
// The certification is cleared rather than kept, because a signature on figures
// that have since changed is worse than no signature at all.
func (s *Service) UncertifySummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	year int16,
	userID pulid.ID,
) (*worker.OSHAAnnualSummary, error) {
	entity, err := s.repo.GetSummary(ctx, &repositories.GetOSHASummaryRequest{
		TenantInfo: tenantInfo,
		Year:       year,
	})
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, errortypes.NewNotFoundError("No summary exists for that year")
	}
	if !entity.IsCertified() {
		return entity, nil
	}

	previous := *entity
	entity.Status = worker.SummaryDraft
	entity.CertifiedAt = nil
	entity.CertifiedByID = pulid.Nil

	updated, err := s.repo.UpdateSummary(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpManage,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment: "Reopened the " + strconv.Itoa(int(updated.Year)) +
			" OSHA 300A summary for correction",
	})
	s.publish(ctx, tenantInfo, realtimeSummary, permission.OpManage, updated.ID, userID)

	return updated, nil
}

func (s *Service) prepareSummary(entity *worker.OSHAAnnualSummary) error {
	entity.NAICSCode = strings.TrimSpace(entity.NAICSCode)
	entity.ExecutiveName = strings.TrimSpace(entity.ExecutiveName)
	entity.ExecutiveTitle = strings.TrimSpace(entity.ExecutiveTitle)
	entity.ExecutivePhone = strings.TrimSpace(entity.ExecutivePhone)
	entity.SubmissionReference = strings.TrimSpace(entity.SubmissionReference)
	entity.Notes = strings.TrimSpace(entity.Notes)
	if entity.Status == "" {
		entity.Status = worker.SummaryDraft
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func applySummaryUpdate(entity *worker.OSHAAnnualSummary, req *SaveSummaryRequest) {
	if req.NAICSCode != nil {
		entity.NAICSCode = *req.NAICSCode
	}
	if req.AverageEmployees != nil {
		entity.AverageEmployees = *req.AverageEmployees
	}
	if req.TotalHoursWorked != nil {
		entity.TotalHoursWorked = *req.TotalHoursWorked
	}
	if req.ExecutiveName != nil {
		entity.ExecutiveName = *req.ExecutiveName
	}
	if req.ExecutiveTitle != nil {
		entity.ExecutiveTitle = *req.ExecutiveTitle
	}
	if req.ExecutivePhone != nil {
		entity.ExecutivePhone = *req.ExecutivePhone
	}
	if req.PostedFrom != nil {
		entity.PostedFrom = req.PostedFrom
	}
	if req.PostedThrough != nil {
		entity.PostedThrough = req.PostedThrough
	}
	if req.SubmittedAt != nil {
		entity.SubmittedAt = req.SubmittedAt
	}
	if req.SubmissionReference != nil {
		entity.SubmissionReference = *req.SubmissionReference
	}
	if req.Notes != nil {
		entity.Notes = *req.Notes
	}
}
