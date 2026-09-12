package workerleaveservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

func (s *Service) ListCases(
	ctx context.Context,
	req *repositories.ListLeaveCasesRequest,
) ([]*worker.WorkerLeaveCase, error) {
	return s.repo.ListCases(ctx, req)
}

// CountCases answers how many cases match a filter. The home tile needs the
// number of outstanding certifications, not the cases themselves.
func (s *Service) CountCases(
	ctx context.Context,
	req *repositories.ListLeaveCasesRequest,
) (int, error) {
	return s.repo.CountCases(ctx, req)
}

func (s *Service) GetCase(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerLeaveCase, error) {
	return s.repo.GetCaseByID(ctx, &repositories.GetLeaveCaseByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeDocument: true,
		IncludeEntries:  true,
	})
}

func (s *Service) prepareCase(entity *worker.WorkerLeaveCase) error {
	entity.Reason = strings.TrimSpace(entity.Reason)
	entity.Notes = strings.TrimSpace(entity.Notes)

	if entity.Status == "" {
		entity.Status = worker.LeaveCasePending
	}
	if entity.LeaveType == "" {
		entity.LeaveType = worker.LeaveTypeFMLA
	}
	if entity.Frequency == "" {
		entity.Frequency = worker.LeaveContinuous
	}
	if entity.CertificationStatus == "" {
		entity.CertificationStatus = worker.CertificationNotRequired
	}
	if entity.RequestedAt <= 0 {
		entity.RequestedAt = timeutils.NowUnix()
	}
	// A case that was never decided has no decision date, whatever a form sent
	// before somebody changed their mind.
	if entity.Status == worker.LeaveCasePending {
		entity.DecidedAt = nil
		entity.DecidedByID = pulid.Nil
	}
	if entity.Status != worker.LeaveCaseClosed {
		entity.ClosedAt = nil
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// OpenCase files a request for leave.
func (s *Service) OpenCase(
	ctx context.Context,
	entity *worker.WorkerLeaveCase,
	userID pulid.ID,
) (*worker.WorkerLeaveCase, error) {
	tenantInfo := caseTenant(entity)

	if _, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         entity.WorkerID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return nil, err
	}

	entity.RecordedByID = userID
	if err := s.prepareCase(entity); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateCase(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    created,
		comment:    "Opened a " + string(created.LeaveType) + " leave case",
	})
	s.publish(ctx, tenantInfo, realtimeCase, permission.OpCreate, created.ID, userID)

	return created, nil
}

// UpdateCaseRequest carries a correction or a step forward. Every field is
// optional: the office fills them in as the case progresses.
type UpdateCaseRequest struct {
	TenantInfo             pagination.TenantInfo
	CaseID                 pulid.ID
	LeaveType              *worker.LeaveType
	Frequency              *worker.LeaveFrequency
	Reason                 *string
	MilitaryCaregiver      *bool
	StartsAt               *int64
	EndsAt                 *int64
	EligibilityHoursWorked *int32
	DocumentID             pulid.ID
	Notes                  *string
	UserID                 pulid.ID
}

func (s *Service) UpdateCase(
	ctx context.Context,
	req *UpdateCaseRequest,
) (*worker.WorkerLeaveCase, error) {
	entity, err := s.repo.GetCaseByID(ctx, &repositories.GetLeaveCaseByIDRequest{
		ID:         req.CaseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := *entity
	applyCaseUpdate(entity, req)

	if err = s.prepareCase(entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateCase(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Updated the leave case",
	})
	s.publish(ctx, req.TenantInfo, realtimeCase, permission.OpUpdate, updated.ID, req.UserID)

	return updated, nil
}

func applyCaseUpdate(entity *worker.WorkerLeaveCase, req *UpdateCaseRequest) {
	if req.LeaveType != nil {
		entity.LeaveType = *req.LeaveType
	}
	if req.Frequency != nil {
		entity.Frequency = *req.Frequency
	}
	if req.Reason != nil {
		entity.Reason = *req.Reason
	}
	if req.MilitaryCaregiver != nil {
		entity.MilitaryCaregiver = *req.MilitaryCaregiver
	}
	if req.StartsAt != nil {
		entity.StartsAt = *req.StartsAt
	}
	if req.EndsAt != nil {
		entity.EndsAt = req.EndsAt
	}
	if req.EligibilityHoursWorked != nil {
		entity.EligibilityHoursWorked = req.EligibilityHoursWorked
	}
	if req.Notes != nil {
		entity.Notes = *req.Notes
	}
	if !req.DocumentID.IsNil() {
		entity.DocumentID = req.DocumentID
	}
}

// DecideCaseRequest is the employer's answer, and the designation decision that
// goes with it. Designating is separate from approving: leave can be granted
// without counting against the entitlement, and 29 CFR 825.301 makes the
// designation the employer's own call.
type DecideCaseRequest struct {
	TenantInfo pagination.TenantInfo
	CaseID     pulid.ID
	Approve    bool
	Designate  bool
	Notes      *string
	UserID     pulid.ID
}

func (s *Service) DecideCase(
	ctx context.Context,
	req *DecideCaseRequest,
) (*worker.WorkerLeaveCase, error) {
	entity, err := s.repo.GetCaseByID(ctx, &repositories.GetLeaveCaseByIDRequest{
		ID:         req.CaseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if entity.Status == worker.LeaveCaseClosed {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A closed case cannot be decided again",
		)
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.DecidedAt = &now
	entity.DecidedByID = req.UserID
	if req.Approve {
		entity.Status = worker.LeaveCaseApproved
		entity.FMLADesignated = req.Designate
	} else {
		entity.Status = worker.LeaveCaseDenied
		// Denied leave cannot draw down an entitlement, whatever the caller
		// asked for.
		entity.FMLADesignated = false
	}
	if req.Notes != nil {
		entity.Notes = strings.TrimSpace(*req.Notes)
	}

	if err = s.prepareCase(entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateCase(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpApprove,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Leave case " + strings.ToLower(updated.Status.Label()),
	})
	s.publish(ctx, req.TenantInfo, realtimeCase, permission.OpApprove, updated.ID, req.UserID)

	return updated, nil
}

// CloseCase ends a case. The days recorded against it stay: they are what the
// entitlement was drawn down by, and closing the case does not give them back.
func (s *Service) CloseCase(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) (*worker.WorkerLeaveCase, error) {
	entity, err := s.repo.GetCaseByID(ctx, &repositories.GetLeaveCaseByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if entity.Status == worker.LeaveCaseClosed {
		return entity, nil
	}
	if entity.Status == worker.LeaveCasePending {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Decide the case before closing it",
		)
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.Status = worker.LeaveCaseClosed
	entity.ClosedAt = &now
	if entity.EndsAt == nil {
		entity.EndsAt = &now
	}

	if err = s.prepareCase(entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateCase(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpApprove,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Closed the leave case",
	})
	s.publish(ctx, tenantInfo, realtimeCase, permission.OpApprove, updated.ID, userID)

	return updated, nil
}

// RequestCertificationRequest asks the employee for medical certification and
// starts the clock. The deadline comes from the organisation's setting, which
// defaults to the fifteen calendar days 29 CFR 825.305(b) allows.
type RequestCertificationRequest struct {
	TenantInfo pagination.TenantInfo
	CaseID     pulid.ID
	DueAt      *int64
	UserID     pulid.ID
}

func (s *Service) RequestCertification(
	ctx context.Context,
	req *RequestCertificationRequest,
) (*worker.WorkerLeaveCase, error) {
	entity, err := s.repo.GetCaseByID(ctx, &repositories.GetLeaveCaseByIDRequest{
		ID:         req.CaseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	control, err := s.repo.GetControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	previous := *entity
	now := timeutils.NowUnix()
	due := now + int64(control.CertificationDueDays)*secondsPerDay
	if req.DueAt != nil && *req.DueAt > 0 {
		due = *req.DueAt
	}

	entity.CertificationStatus = worker.CertificationRequested
	entity.CertificationRequestedAt = &now
	entity.CertificationDueAt = &due
	entity.CertificationReceivedAt = nil

	if err = s.prepareCase(entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateCase(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Requested medical certification",
	})
	s.publish(ctx, req.TenantInfo, realtimeCase, permission.OpUpdate, updated.ID, req.UserID)

	return updated, nil
}

// RecordCertificationRequest files what came back.
type RecordCertificationRequest struct {
	TenantInfo           pagination.TenantInfo
	CaseID               pulid.ID
	Status               worker.LeaveCertificationStatus
	ReceivedAt           *int64
	RecertificationDueAt *int64
	DocumentID           pulid.ID
	UserID               pulid.ID
}

func (s *Service) RecordCertification(
	ctx context.Context,
	req *RecordCertificationRequest,
) (*worker.WorkerLeaveCase, error) {
	entity, err := s.repo.GetCaseByID(ctx, &repositories.GetLeaveCaseByIDRequest{
		ID:         req.CaseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := *entity
	entity.CertificationStatus = req.Status
	if req.Status == worker.CertificationReceived ||
		req.Status == worker.CertificationInsufficient {
		received := timeutils.NowUnix()
		if req.ReceivedAt != nil && *req.ReceivedAt > 0 {
			received = *req.ReceivedAt
		}
		entity.CertificationReceivedAt = &received
	}
	if req.RecertificationDueAt != nil {
		entity.RecertificationDueAt = req.RecertificationDueAt
	}
	if !req.DocumentID.IsNil() {
		entity.DocumentID = req.DocumentID
	}

	if err = s.prepareCase(entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateCase(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Certification is now " + strings.ToLower(updated.CertificationStatus.Label()),
	})
	s.publish(ctx, req.TenantInfo, realtimeCase, permission.OpUpdate, updated.ID, req.UserID)

	return updated, nil
}

func (s *Service) ListEntries(
	ctx context.Context,
	req *repositories.ListLeaveEntriesRequest,
) ([]*worker.WorkerLeaveEntry, error) {
	return s.repo.ListEntries(ctx, req)
}

func (s *Service) ListEntriesByCaseIDs(
	ctx context.Context,
	req *repositories.ListLeaveEntriesByCaseIDsRequest,
) (map[pulid.ID][]*worker.WorkerLeaveEntry, error) {
	return s.repo.ListEntriesByCaseIDs(ctx, req)
}

// RecordDayRequest records one day of leave taken against a case.
type RecordDayRequest struct {
	TenantInfo pagination.TenantInfo
	CaseID     pulid.ID
	UsedOn     int64
	Hours      decimal.Decimal
	PTOID      pulid.ID
	Notes      string
	UserID     pulid.ID
}

// RecordDay adds a day to a case. Whether it counts against the entitlement is
// copied from the case as it stands now, rather than read through the case
// later: undesignating a case months afterwards must not silently rewrite what
// was already counted against a period that has since closed.
func (s *Service) RecordDay(
	ctx context.Context,
	req *RecordDayRequest,
) (*worker.WorkerLeaveEntry, error) {
	leaveCase, err := s.repo.GetCaseByID(ctx, &repositories.GetLeaveCaseByIDRequest{
		ID:         req.CaseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !leaveCase.IsOpen() {
		return nil, errortypes.NewValidationError(
			"leaveCaseId",
			errortypes.ErrInvalid,
			"Leave cannot be recorded against a {0} case",
			strings.ToLower(leaveCase.Status.Label()),
		)
	}

	entity := &worker.WorkerLeaveEntry{
		OrganizationID:           leaveCase.OrganizationID,
		BusinessUnitID:           leaveCase.BusinessUnitID,
		WorkerID:                 leaveCase.WorkerID,
		LeaveCaseID:              leaveCase.ID,
		UsedOn:                   req.UsedOn,
		Hours:                    req.Hours,
		CountsAgainstEntitlement: leaveCase.CountsAgainstEntitlement(),
		PTOID:                    req.PTOID,
		Notes:                    strings.TrimSpace(req.Notes),
		RecordedByID:             req.UserID,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateEntry(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    created,
		comment:    "Recorded " + created.Hours.String() + " hours of leave",
	})
	s.publish(ctx, req.TenantInfo, realtimeEntry, permission.OpCreate, created.ID, req.UserID)

	return created, nil
}

// UpdateDayRequest corrects a day already recorded.
type UpdateDayRequest struct {
	TenantInfo pagination.TenantInfo
	EntryID    pulid.ID
	Hours      *decimal.Decimal
	// Counts lets the office change its mind about whether one particular day
	// draws the entitlement down, without touching the rest of the case.
	Counts *bool
	PTOID  pulid.ID
	Notes  *string
	UserID pulid.ID
}

func (s *Service) UpdateDay(
	ctx context.Context,
	req *UpdateDayRequest,
) (*worker.WorkerLeaveEntry, error) {
	entity, err := s.repo.GetEntryByID(ctx, &repositories.GetLeaveEntryByIDRequest{
		ID:         req.EntryID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := *entity
	if req.Hours != nil {
		entity.Hours = *req.Hours
	}
	if req.Counts != nil {
		entity.CountsAgainstEntitlement = *req.Counts
	}
	if req.Notes != nil {
		entity.Notes = strings.TrimSpace(*req.Notes)
	}
	if !req.PTOID.IsNil() {
		entity.PTOID = req.PTOID
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.repo.UpdateEntry(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Corrected a day of leave",
	})
	s.publish(ctx, req.TenantInfo, realtimeEntry, permission.OpUpdate, updated.ID, req.UserID)

	return updated, nil
}

// DeleteDay removes a day recorded in error, which gives the hours back to the
// entitlement.
func (s *Service) DeleteDay(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) error {
	entity, err := s.repo.GetEntryByID(ctx, &repositories.GetLeaveEntryByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	if err = s.repo.DeleteEntry(ctx, &repositories.GetLeaveEntryByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		return err
	}

	s.audit(&auditParams{
		resourceID: id.String(),
		operation:  permission.OpDelete,
		userID:     userID,
		tenant:     tenantInfo,
		current:    entity,
		comment:    "Removed a day of leave recorded in error",
	})
	s.publish(ctx, tenantInfo, realtimeEntry, permission.OpDelete, id, userID)

	return nil
}
