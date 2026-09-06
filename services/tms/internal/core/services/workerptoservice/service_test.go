package workerptoservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakePTORepo struct {
	repositories.WorkerPTORepository
	create       func(ctx context.Context, entity *worker.WorkerPTO) (*worker.WorkerPTO, error)
	hasOverlap   func(ctx context.Context, req *repositories.PTOOverlapRequest) (bool, error)
	getByID      func(ctx context.Context, req *repositories.GetPTOByIDRequest) (*worker.WorkerPTO, error)
	getByIDs     func(ctx context.Context, req *repositories.GetPTOsByIDsRequest) ([]*worker.WorkerPTO, error)
	updateStatus func(ctx context.Context, req *repositories.UpdatePTOStatusRequest) (*worker.WorkerPTO, error)
	update       func(ctx context.Context, entity *worker.WorkerPTO) (*worker.WorkerPTO, error)
}

func (f *fakePTORepo) Update(
	ctx context.Context,
	entity *worker.WorkerPTO,
) (*worker.WorkerPTO, error) {
	return f.update(ctx, entity)
}

func (f *fakePTORepo) Create(
	ctx context.Context,
	entity *worker.WorkerPTO,
) (*worker.WorkerPTO, error) {
	return f.create(ctx, entity)
}

func (f *fakePTORepo) HasOverlap(
	ctx context.Context,
	req *repositories.PTOOverlapRequest,
) (bool, error) {
	if f.hasOverlap == nil {
		return false, nil
	}
	return f.hasOverlap(ctx, req)
}

func (f *fakePTORepo) GetByID(
	ctx context.Context,
	req *repositories.GetPTOByIDRequest,
) (*worker.WorkerPTO, error) {
	return f.getByID(ctx, req)
}

func (f *fakePTORepo) GetByIDs(
	ctx context.Context,
	req *repositories.GetPTOsByIDsRequest,
) ([]*worker.WorkerPTO, error) {
	return f.getByIDs(ctx, req)
}

func (f *fakePTORepo) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	return f.updateStatus(ctx, req)
}

type fakeAuditService struct {
	serviceports.AuditService
	logged []*serviceports.LogActionParams
}

func (f *fakeAuditService) LogAction(
	params *serviceports.LogActionParams,
	_ ...serviceports.LogOption,
) error {
	f.logged = append(f.logged, params)
	return nil
}

func validPTO() *worker.WorkerPTO {
	return &worker.WorkerPTO{
		WorkerID:       pulid.MustNew("wrk_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         worker.PTOStatusRequested,
		Type:           worker.PTOTypeVacation,
		StartDate:      1767225600,
		EndDate:        1767398400,
		Reason:         "Family vacation",
	}
}

func TestCreateRejectsInvalidPTO(t *testing.T) {
	t.Parallel()

	repoCalled := false
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			create: func(
				_ context.Context,
				entity *worker.WorkerPTO,
			) (*worker.WorkerPTO, error) {
				repoCalled = true
				return entity, nil
			},
		},
	}

	entity := validPTO()
	entity.WorkerID = pulid.Nil
	entity.Reason = ""
	entity.EndDate = entity.StartDate

	created, err := svc.Create(t.Context(), entity, pulid.MustNew("usr_"))
	require.Error(t, err)
	require.Nil(t, created)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.True(t, multiErr.HasErrors())
	assert.False(t, repoCalled, "invalid PTO must not reach the repository")
}

func TestCreatePersistsValidPTO(t *testing.T) {
	t.Parallel()

	audit := &fakeAuditService{}
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			create: func(
				_ context.Context,
				entity *worker.WorkerPTO,
			) (*worker.WorkerPTO, error) {
				entity.ID = pulid.MustNew("wrkpto_")
				return entity, nil
			},
		},
		auditService: audit,
	}

	created, err := svc.Create(t.Context(), validPTO(), pulid.MustNew("usr_"))
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.False(t, created.ID.IsNil())
	require.Len(t, audit.logged, 1)
}

func TestValidateChartRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		svc := &Service{}
		err := svc.validateChartRequest(&repositories.PTOChartRequest{
			Filter:        &pagination.QueryOptions{},
			StartDateFrom: 1735689600,
			StartDateTo:   1736294400,
			Type:          "all",
		})
		require.NoError(t, err)
	})

	t.Run("rejects invalid range and invalid type", func(t *testing.T) {
		svc := &Service{}
		err := svc.validateChartRequest(&repositories.PTOChartRequest{
			Filter:        &pagination.QueryOptions{},
			StartDateFrom: 1736294400,
			StartDateTo:   1735689600,
			Type:          "NotAType",
		})
		require.Error(t, err)

		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr)

		assert.True(t, multiErr.HasErrors())
		assert.GreaterOrEqual(t, len(multiErr.Errors), 2)
	})

	t.Run("rejects range over 366 days", func(t *testing.T) {
		svc := &Service{}
		err := svc.validateChartRequest(&repositories.PTOChartRequest{
			Filter:        &pagination.QueryOptions{},
			StartDateFrom: 1704067200,
			StartDateTo:   1735776002,
			Type:          "Vacation",
		})
		require.Error(t, err)
	})

	t.Run("rejects invalid worker ID", func(t *testing.T) {
		svc := &Service{}
		err := svc.validateChartRequest(&repositories.PTOChartRequest{
			Filter:        &pagination.QueryOptions{},
			StartDateFrom: 1735689600,
			StartDateTo:   1736294400,
			WorkerID:      "bad-id",
		})
		require.Error(t, err)
	})
}

func storedPTO(status worker.PTOStatus) *worker.WorkerPTO {
	pto := validPTO()
	pto.ID = pulid.MustNew("wrkpto_")
	pto.Status = status
	pto.Version = 3
	return pto
}

func tenantFor(pto *worker.WorkerPTO) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pto.OrganizationID,
		BuID:   pto.BusinessUnitID,
		UserID: pulid.MustNew("usr_"),
	}
}

func applyStatus(
	req *repositories.UpdatePTOStatusRequest,
	current *worker.WorkerPTO,
) *worker.WorkerPTO {
	updated := *current
	updated.Status = req.Status
	updated.Version = current.Version + 1
	switch req.Status { //nolint:exhaustive // test helper mirrors the repository
	case worker.PTOStatusApproved:
		updated.ApproverID = req.UserID
	case worker.PTOStatusRejected:
		updated.RejectorID = req.UserID
		updated.RejectionReason = req.Reason
	case worker.PTOStatusCancelled:
		updated.CancelledByID = req.UserID
		updated.CancellationReason = req.Reason
	}
	return &updated
}

func TestCreateRejectsOverlappingPTO(t *testing.T) {
	t.Parallel()

	repoCalled := false
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			hasOverlap: func(
				_ context.Context,
				req *repositories.PTOOverlapRequest,
			) (bool, error) {
				assert.Equal(t, int64(1767225600), req.StartDate)
				assert.Equal(t, int64(1767398400), req.EndDate)
				return true, nil
			},
			create: func(
				_ context.Context,
				entity *worker.WorkerPTO,
			) (*worker.WorkerPTO, error) {
				repoCalled = true
				return entity, nil
			},
		},
	}

	created, err := svc.Create(t.Context(), validPTO(), pulid.MustNew("usr_"))
	require.Error(t, err)
	require.Nil(t, created)
	assert.False(t, repoCalled, "overlapping PTO must not reach the repository")

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "startDate", validationErr.Field)
}

func TestRejectRequiresReason(t *testing.T) {
	t.Parallel()

	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(
				_ context.Context,
				_ *repositories.GetPTOByIDRequest,
			) (*worker.WorkerPTO, error) {
				t.Fatal("reject without a reason must not read the repository")
				return nil, nil
			},
		},
	}

	current := storedPTO(worker.PTOStatusRequested)
	_, err := svc.Reject(t.Context(), &repositories.UpdatePTOStatusRequest{
		ID:         current.ID,
		TenantInfo: tenantFor(current),
		UserID:     pulid.MustNew("usr_"),
		Reason:     "   ",
	})
	require.Error(t, err)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "reason", validationErr.Field)
}

func TestTransitionRefusesInvalidSourceStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		from   worker.PTOStatus
		action func(*Service, context.Context, *repositories.UpdatePTOStatusRequest) (*worker.WorkerPTO, error)
	}{
		{"approve rejected", worker.PTOStatusRejected, (*Service).Approve},
		{"approve cancelled", worker.PTOStatusCancelled, (*Service).Approve},
		{"approve approved", worker.PTOStatusApproved, (*Service).Approve},
		{"reject approved", worker.PTOStatusApproved, (*Service).Reject},
		{"cancel rejected", worker.PTOStatusRejected, (*Service).Cancel},
		{"cancel cancelled", worker.PTOStatusCancelled, (*Service).Cancel},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			current := storedPTO(tc.from)
			updateCalled := false
			svc := &Service{
				l: zap.NewNop(),
				repo: &fakePTORepo{
					getByID: func(
						_ context.Context,
						_ *repositories.GetPTOByIDRequest,
					) (*worker.WorkerPTO, error) {
						return current, nil
					},
					updateStatus: func(
						_ context.Context,
						_ *repositories.UpdatePTOStatusRequest,
					) (*worker.WorkerPTO, error) {
						updateCalled = true
						return nil, nil
					},
				},
			}

			_, err := tc.action(svc, t.Context(), &repositories.UpdatePTOStatusRequest{
				ID:         current.ID,
				TenantInfo: tenantFor(current),
				UserID:     pulid.MustNew("usr_"),
				Reason:     "not needed",
			})
			require.Error(t, err)
			assert.False(t, updateCalled, "invalid transition must not reach the repository")

			var validationErr *errortypes.Error
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, "status", validationErr.Field)
			assert.Equal(t, errortypes.ErrInvalidOperation, validationErr.Code)
		})
	}
}

func TestTransitionCarriesReasonVersionAndAudit(t *testing.T) {
	t.Parallel()

	current := storedPTO(worker.PTOStatusRequested)
	audit := &fakeAuditService{}
	var captured *repositories.UpdatePTOStatusRequest
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(
				_ context.Context,
				_ *repositories.GetPTOByIDRequest,
			) (*worker.WorkerPTO, error) {
				return current, nil
			},
			updateStatus: func(
				_ context.Context,
				req *repositories.UpdatePTOStatusRequest,
			) (*worker.WorkerPTO, error) {
				captured = req
				return applyStatus(req, current), nil
			},
		},
		auditService: audit,
	}

	actor := pulid.MustNew("usr_")
	updated, err := svc.Reject(t.Context(), &repositories.UpdatePTOStatusRequest{
		ID:         current.ID,
		TenantInfo: tenantFor(current),
		UserID:     actor,
		Reason:     "  Coverage gap on that route  ",
	})
	require.NoError(t, err)
	require.NotNil(t, captured)

	assert.Equal(t, worker.PTOStatusRejected, captured.Status)
	assert.Equal(t, "Coverage gap on that route", captured.Reason)
	assert.Equal(t, current.Version, captured.ExpectedVersion)
	assert.Equal(t, "Coverage gap on that route", updated.RejectionReason)
	assert.Equal(t, actor, updated.RejectorID)

	require.Len(t, audit.logged, 1)
	assert.Equal(t, permission.OpReject, audit.logged[0].Operation)
	assert.Equal(t, current.ID.String(), audit.logged[0].ResourceID)
	assert.NotNil(t, audit.logged[0].PreviousState)
}

func TestCancelApprovedPTO(t *testing.T) {
	t.Parallel()

	current := storedPTO(worker.PTOStatusApproved)
	audit := &fakeAuditService{}
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(
				_ context.Context,
				_ *repositories.GetPTOByIDRequest,
			) (*worker.WorkerPTO, error) {
				return current, nil
			},
			updateStatus: func(
				_ context.Context,
				req *repositories.UpdatePTOStatusRequest,
			) (*worker.WorkerPTO, error) {
				return applyStatus(req, current), nil
			},
		},
		auditService: audit,
	}

	updated, err := svc.Cancel(t.Context(), &repositories.UpdatePTOStatusRequest{
		ID:         current.ID,
		TenantInfo: tenantFor(current),
		UserID:     pulid.MustNew("usr_"),
	})
	require.NoError(t, err)
	assert.Equal(t, worker.PTOStatusCancelled, updated.Status)
	assert.Empty(t, updated.CancellationReason)
	require.Len(t, audit.logged, 1)
	assert.Equal(t, permission.OpCancel, audit.logged[0].Operation)
}

func TestBulkActionValidation(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), repo: &fakePTORepo{}}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	_, err := svc.BulkAction(t.Context(), &serviceports.PTOBulkActionRequest{
		TenantInfo: tenant,
		Action:     serviceports.PTOBulkActionApprove,
	})
	require.Error(t, err, "empty selection")

	_, err = svc.BulkAction(t.Context(), &serviceports.PTOBulkActionRequest{
		TenantInfo: tenant,
		PTOIDs:     []pulid.ID{pulid.MustNew("wrkpto_")},
		Action:     serviceports.PTOBulkActionType("Archive"),
	})
	require.Error(t, err, "unknown action")

	_, err = svc.BulkAction(t.Context(), &serviceports.PTOBulkActionRequest{
		TenantInfo: tenant,
		PTOIDs:     []pulid.ID{pulid.MustNew("wrkpto_")},
		Action:     serviceports.PTOBulkActionReject,
	})
	require.Error(t, err, "reject without reason")

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "reason", validationErr.Field)
}

func TestBulkActionReportsPerItemOutcomes(t *testing.T) {
	t.Parallel()

	requested := storedPTO(worker.PTOStatusRequested)
	alreadyRejected := storedPTO(worker.PTOStatusRejected)
	missing := pulid.MustNew("wrkpto_")
	tenant := tenantFor(requested)
	alreadyRejected.OrganizationID = requested.OrganizationID
	alreadyRejected.BusinessUnitID = requested.BusinessUnitID

	audit := &fakeAuditService{}
	var updates []*repositories.UpdatePTOStatusRequest
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			getByIDs: func(
				_ context.Context,
				req *repositories.GetPTOsByIDsRequest,
			) ([]*worker.WorkerPTO, error) {
				assert.Len(t, req.IDs, 3)
				return []*worker.WorkerPTO{requested, alreadyRejected}, nil
			},
			getByID: func(
				_ context.Context,
				_ *repositories.GetPTOByIDRequest,
			) (*worker.WorkerPTO, error) {
				t.Fatal("bulk actions must reuse the batch lookup")
				return nil, nil
			},
			updateStatus: func(
				_ context.Context,
				req *repositories.UpdatePTOStatusRequest,
			) (*worker.WorkerPTO, error) {
				updates = append(updates, req)
				return applyStatus(req, requested), nil
			},
		},
		auditService: audit,
	}

	payload, err := svc.BulkAction(t.Context(), &serviceports.PTOBulkActionRequest{
		TenantInfo: tenant,
		PTOIDs:     []pulid.ID{requested.ID, alreadyRejected.ID, missing},
		Action:     serviceports.PTOBulkActionApprove,
		UserID:     tenant.UserID,
	})
	require.NoError(t, err)
	require.Len(t, payload.Results, 3)
	assert.Equal(t, 1, payload.SuccessCount)
	assert.Equal(t, 2, payload.FailureCount)

	assert.True(t, payload.Results[0].Success)
	assert.Equal(t, requested.ID, payload.Results[0].PTOID)

	assert.False(t, payload.Results[1].Success)
	assert.Contains(t, payload.Results[1].Error, "rejected")

	assert.False(t, payload.Results[2].Success)
	assert.Contains(t, payload.Results[2].Error, "not found")

	require.Len(t, updates, 1)
	assert.Equal(t, worker.PTOStatusApproved, updates[0].Status)
	assert.Equal(t, requested.Version, updates[0].ExpectedVersion)
	require.Len(t, audit.logged, 1)
	assert.Equal(t, permission.OpApprove, audit.logged[0].Operation)
}

func TestTransitionSMSMessage(t *testing.T) {
	t.Parallel()

	pto := storedPTO(worker.PTOStatusRequested)

	approved, ok := transitionSMSMessage("Dana", pto, worker.PTOStatusApproved, "")
	require.True(t, ok)
	assert.Contains(t, approved, "Dana has approved")

	rejected, ok := transitionSMSMessage("Dana", pto, worker.PTOStatusRejected, "Coverage")
	require.True(t, ok)
	assert.Contains(t, rejected, "Reason: Coverage")

	cancelled, ok := transitionSMSMessage("Dana", pto, worker.PTOStatusCancelled, "")
	require.True(t, ok)
	assert.NotContains(t, cancelled, "Reason:")

	_, ok = transitionSMSMessage("Dana", pto, worker.PTOStatusRequested, "")
	assert.False(t, ok)
}

func TestUpdateRefusesNonRequestedPTO(t *testing.T) {
	t.Parallel()

	current := storedPTO(worker.PTOStatusApproved)
	updateCalled := false
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(
				_ context.Context,
				_ *repositories.GetPTOByIDRequest,
			) (*worker.WorkerPTO, error) {
				return current, nil
			},
			update: func(
				_ context.Context,
				_ *worker.WorkerPTO,
			) (*worker.WorkerPTO, error) {
				updateCalled = true
				return nil, nil
			},
		},
	}

	edited := *current
	edited.Reason = "Changed my mind"
	_, err := svc.Update(t.Context(), &edited, pulid.MustNew("usr_"))
	require.Error(t, err)
	assert.False(t, updateCalled)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "status", validationErr.Field)
}

func TestUpdateExcludesItselfFromOverlapAndAudits(t *testing.T) {
	t.Parallel()

	current := storedPTO(worker.PTOStatusRequested)
	audit := &fakeAuditService{}
	var overlapReq *repositories.PTOOverlapRequest
	var persisted *worker.WorkerPTO
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(
				_ context.Context,
				_ *repositories.GetPTOByIDRequest,
			) (*worker.WorkerPTO, error) {
				return current, nil
			},
			hasOverlap: func(
				_ context.Context,
				req *repositories.PTOOverlapRequest,
			) (bool, error) {
				overlapReq = req
				return false, nil
			},
			update: func(
				_ context.Context,
				entity *worker.WorkerPTO,
			) (*worker.WorkerPTO, error) {
				persisted = entity
				entity.Version++
				return entity, nil
			},
		},
		auditService: audit,
	}

	edited := &worker.WorkerPTO{
		ID:             current.ID,
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
		Type:           worker.PTOTypeSick,
		StartDate:      current.StartDate + 86400,
		EndDate:        current.EndDate + 86400,
		Reason:         "Moved by a day",
		Version:        current.Version,
	}
	updated, err := svc.Update(t.Context(), edited, pulid.MustNew("usr_"))
	require.NoError(t, err)

	require.NotNil(t, overlapReq)
	assert.Equal(t, current.ID, overlapReq.ExcludeID)
	assert.Equal(t, current.WorkerID, overlapReq.WorkerID)

	require.NotNil(t, persisted)
	assert.Equal(t, current.WorkerID, persisted.WorkerID, "worker cannot be reassigned on edit")
	assert.Equal(t, worker.PTOStatusRequested, persisted.Status)
	assert.Equal(t, worker.PTOTypeSick, updated.Type)
	assert.Equal(t, current.Version+1, updated.Version)

	require.Len(t, audit.logged, 1)
	assert.Equal(t, permission.OpUpdate, audit.logged[0].Operation)
}

func TestUpdateRejectsOverlap(t *testing.T) {
	t.Parallel()

	current := storedPTO(worker.PTOStatusRequested)
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(
				_ context.Context,
				_ *repositories.GetPTOByIDRequest,
			) (*worker.WorkerPTO, error) {
				return current, nil
			},
			hasOverlap: func(
				_ context.Context,
				_ *repositories.PTOOverlapRequest,
			) (bool, error) {
				return true, nil
			},
			update: func(
				_ context.Context,
				_ *worker.WorkerPTO,
			) (*worker.WorkerPTO, error) {
				t.Fatal("overlapping edit must not be persisted")
				return nil, nil
			},
		},
	}

	edited := *current
	_, err := svc.Update(t.Context(), &edited, pulid.MustNew("usr_"))
	require.Error(t, err)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "startDate", validationErr.Field)
}
