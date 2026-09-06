package workerdqfservice

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
)

func (s *Service) ListVerifications(
	ctx context.Context,
	req *repositories.ListEmploymentVerificationsRequest,
) ([]*worker.WorkerEmploymentVerification, error) {
	return s.repo.ListVerifications(ctx, req)
}

func (s *Service) ListVerificationsByWorkerIDs(
	ctx context.Context,
	req *repositories.ListEmploymentVerificationsByWorkerIDsRequest,
) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error) {
	return s.repo.ListVerificationsByWorkerIDs(ctx, req)
}

func (s *Service) GetVerification(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	return s.repo.GetVerificationByID(ctx, &repositories.GetEmploymentVerificationByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeDocument: true,
	})
}

func (s *Service) prepare(entity *worker.WorkerEmploymentVerification) error {
	entity.EmployerName = strings.TrimSpace(entity.EmployerName)
	entity.EmployerDOTNumber = strings.TrimSpace(entity.EmployerDOTNumber)
	entity.EmployerMCNumber = strings.TrimSpace(entity.EmployerMCNumber)
	entity.ContactName = strings.TrimSpace(entity.ContactName)
	entity.ContactPhone = strings.TrimSpace(entity.ContactPhone)
	entity.ContactEmail = strings.TrimSpace(entity.ContactEmail)
	entity.Findings = strings.TrimSpace(entity.Findings)
	entity.Notes = strings.TrimSpace(entity.Notes)

	// The column defaults cover the insert, but validation runs first, so the
	// same defaults have to be applied here.
	if entity.Status == "" {
		entity.Status = worker.VerificationPending
	}
	if entity.Method == "" {
		entity.Method = worker.VerificationByEmail
	}
	// A non-regulated employer has no testing record to ask about, so anything
	// recorded against the drug and alcohol half is meaningless there.
	if !entity.WasDOTRegulated {
		entity.DrugAlcoholResponseReceivedAt = nil
		entity.HadDrugAlcoholViolations = false
	}
	if !entity.HadAccidents {
		entity.AccidentCount = 0
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// RecordVerification files a previous employer to investigate.
func (s *Service) RecordVerification(
	ctx context.Context,
	entity *worker.WorkerEmploymentVerification,
	userID pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	tenantInfo := verificationTenant(entity)

	if _, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         entity.WorkerID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return nil, err
	}

	entity.RequestedByID = userID
	if err := s.prepare(entity); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateVerification(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    created,
		comment:    "Recorded previous employer " + created.EmployerName,
	})
	s.publish(ctx, tenantInfo, permission.OpCreate, created.ID, userID)

	return created, nil
}

// UpdateVerificationRequest carries a correction or a step forward. Every field
// is optional: the office fills them in as the investigation progresses.
type UpdateVerificationRequest struct {
	TenantInfo                    pagination.TenantInfo
	VerificationID                pulid.ID
	EmployerName                  *string
	EmployerDOTNumber             *string
	EmployerMCNumber              *string
	ContactName                   *string
	ContactPhone                  *string
	ContactEmail                  *string
	EmployedFrom                  *int64
	EmployedTo                    *int64
	WasDOTRegulated               *bool
	Status                        *worker.EmploymentVerificationStatus
	Method                        *worker.EmploymentVerificationMethod
	RequestedAt                   *int64
	ResponseReceivedAt            *int64
	DrugAlcoholResponseReceivedAt *int64
	HadAccidents                  *bool
	AccidentCount                 *int32
	HadDrugAlcoholViolations      *bool
	Findings                      *string
	Notes                         *string
	DocumentID                    pulid.ID
	UserID                        pulid.ID
}

func (s *Service) UpdateVerification(
	ctx context.Context,
	req *UpdateVerificationRequest,
) (*worker.WorkerEmploymentVerification, error) {
	entity, err := s.repo.GetVerificationByID(
		ctx,
		&repositories.GetEmploymentVerificationByIDRequest{
			ID:         req.VerificationID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	previous := *entity
	applyVerificationUpdate(entity, req)

	if err = s.prepare(entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateVerification(ctx, entity)
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
		comment:    updated.EmployerName + " is now " + updated.Status.Label(),
	})
	s.publish(ctx, req.TenantInfo, permission.OpUpdate, updated.ID, req.UserID)

	return updated, nil
}

func applyVerificationUpdate(
	entity *worker.WorkerEmploymentVerification,
	req *UpdateVerificationRequest,
) {
	if req.EmployerName != nil {
		entity.EmployerName = *req.EmployerName
	}
	if req.EmployerDOTNumber != nil {
		entity.EmployerDOTNumber = *req.EmployerDOTNumber
	}
	if req.EmployerMCNumber != nil {
		entity.EmployerMCNumber = *req.EmployerMCNumber
	}
	if req.ContactName != nil {
		entity.ContactName = *req.ContactName
	}
	if req.ContactPhone != nil {
		entity.ContactPhone = *req.ContactPhone
	}
	if req.ContactEmail != nil {
		entity.ContactEmail = *req.ContactEmail
	}
	if req.EmployedFrom != nil {
		entity.EmployedFrom = req.EmployedFrom
	}
	if req.EmployedTo != nil {
		entity.EmployedTo = req.EmployedTo
	}
	if req.WasDOTRegulated != nil {
		entity.WasDOTRegulated = *req.WasDOTRegulated
	}
	if req.Status != nil {
		entity.Status = *req.Status
	}
	if req.Method != nil {
		entity.Method = *req.Method
	}
	if req.RequestedAt != nil {
		entity.RequestedAt = req.RequestedAt
	}
	if req.ResponseReceivedAt != nil {
		entity.ResponseReceivedAt = req.ResponseReceivedAt
	}
	if req.DrugAlcoholResponseReceivedAt != nil {
		entity.DrugAlcoholResponseReceivedAt = req.DrugAlcoholResponseReceivedAt
	}
	if req.HadAccidents != nil {
		entity.HadAccidents = *req.HadAccidents
	}
	if req.AccidentCount != nil {
		entity.AccidentCount = *req.AccidentCount
	}
	if req.HadDrugAlcoholViolations != nil {
		entity.HadDrugAlcoholViolations = *req.HadDrugAlcoholViolations
	}
	if req.Findings != nil {
		entity.Findings = *req.Findings
	}
	if req.Notes != nil {
		entity.Notes = *req.Notes
	}
	if !req.DocumentID.IsNil() {
		entity.DocumentID = req.DocumentID
	}
}

// MarkRequested records that the request went out. It is its own operation
// because "we sent it today" is the commonest thing the office does, and making
// them fill a form to say so is how a file ends up with no request date.
func (s *Service) MarkRequested(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	now := timeutils.NowUnix()
	status := worker.VerificationRequested
	return s.UpdateVerification(ctx, &UpdateVerificationRequest{
		TenantInfo:     tenantInfo,
		VerificationID: id,
		Status:         &status,
		RequestedAt:    &now,
		UserID:         userID,
	})
}

// RecordFollowUp is the record of good-faith effort 49 CFR 391.23 asks for when
// a previous employer does not answer. Each chase is counted and dated, because
// the count is the evidence.
func (s *Service) RecordFollowUp(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	entity, err := s.repo.GetVerificationByID(
		ctx,
		&repositories.GetEmploymentVerificationByIDRequest{
			ID:         id,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	if entity.Status != worker.VerificationRequested {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only a request that is awaiting a response can be chased",
		)
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.FollowUpCount++
	entity.LastFollowUpAt = &now

	if err = s.prepare(entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateVerification(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Followed up with " + updated.EmployerName,
	})
	s.publish(ctx, tenantInfo, permission.OpUpdate, updated.ID, userID)

	return updated, nil
}

// DeleteVerification removes an employer recorded in error. It is a delete
// rather than an archive because a row that should never have existed is not
// history worth keeping — and a settled investigation is normally corrected
// rather than removed.
func (s *Service) DeleteVerification(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) error {
	entity, err := s.repo.GetVerificationByID(
		ctx,
		&repositories.GetEmploymentVerificationByIDRequest{
			ID:         id,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return err
	}

	if err = s.repo.DeleteVerification(
		ctx,
		&repositories.GetEmploymentVerificationByIDRequest{ID: id, TenantInfo: tenantInfo},
	); err != nil {
		return err
	}

	s.audit(&auditParams{
		resourceID: id.String(),
		operation:  permission.OpDelete,
		userID:     userID,
		tenant:     tenantInfo,
		current:    entity,
		comment:    "Removed previous employer " + entity.EmployerName,
	})
	s.publish(ctx, tenantInfo, permission.OpDelete, id, userID)

	return nil
}
