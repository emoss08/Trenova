package workerdrugalcoholservice

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

func (s *Service) ListQueries(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) ([]*worker.WorkerClearinghouseQuery, error) {
	return s.repo.ListQueries(ctx, &repositories.ListClearinghouseQueriesRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
}

// RecordQuery files a Clearinghouse query. It can be filed pending — the office
// often logs the request before the answer comes back — or with the answer
// already in hand.
func (s *Service) RecordQuery(
	ctx context.Context,
	entity *worker.WorkerClearinghouseQuery,
	userID pulid.ID,
) (*worker.WorkerClearinghouseQuery, error) {
	tenantInfo := queryTenant(entity)

	if _, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         entity.WorkerID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return nil, err
	}

	entity.PerformedByID = userID
	entity.Reference = strings.TrimSpace(entity.Reference)
	entity.Notes = strings.TrimSpace(entity.Notes)
	if entity.RequestedAt <= 0 {
		entity.RequestedAt = timeutils.NowUnix()
	}
	// The column defaults to Pending, but validation runs before the insert
	// does, so the default has to be applied here too or a caller who leaves
	// the field alone is told the result is missing.
	if entity.Result == "" {
		entity.Result = worker.ClearinghouseResultPending
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateQuery(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerDOTTest,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    created,
		comment:    "Ran a " + created.QueryType.Label(),
	})
	s.publish(ctx, tenantInfo, realtimeQuery, permission.OpCreate, created.ID, userID)
	s.refreshRollupQuietly(ctx, tenantInfo, created.WorkerID)

	return created, nil
}

// CompleteQueryRequest carries the answer the Clearinghouse returned.
type CompleteQueryRequest struct {
	TenantInfo     pagination.TenantInfo
	QueryID        pulid.ID
	Result         worker.ClearinghouseResult
	CompletedAt    int64
	ViolationCount int32
	Reference      string
	DocumentID     pulid.ID
	Notes          string
	UserID         pulid.ID
}

func (s *Service) CompleteQuery(
	ctx context.Context,
	req *CompleteQueryRequest,
) (*worker.WorkerClearinghouseQuery, error) {
	entity, err := s.repo.GetQueryByID(ctx, &repositories.GetClearinghouseQueryByIDRequest{
		ID:         req.QueryID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !entity.IsPending() {
		return nil, errortypes.NewValidationError(
			"result",
			errortypes.ErrInvalid,
			"This query has already been answered; run a new query to re-check",
		)
	}
	if req.Result == worker.ClearinghouseResultPending {
		return nil, errortypes.NewValidationError(
			"result",
			errortypes.ErrRequired,
			"An answer is required",
		)
	}

	previous := *entity

	completedAt := req.CompletedAt
	if completedAt <= 0 {
		completedAt = timeutils.NowUnix()
	}
	entity.Result = req.Result
	entity.CompletedAt = &completedAt
	entity.ViolationCount = req.ViolationCount
	if req.Reference != "" {
		entity.Reference = strings.TrimSpace(req.Reference)
	}
	if req.Notes != "" {
		entity.Notes = strings.TrimSpace(req.Notes)
	}
	if !req.DocumentID.IsNil() {
		entity.DocumentID = req.DocumentID
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.repo.UpdateQuery(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerDOTTest,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Clearinghouse answered: " + string(updated.Result),
	})
	s.publish(ctx, req.TenantInfo, realtimeQuery, permission.OpUpdate, updated.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, updated.WorkerID)

	return updated, nil
}
