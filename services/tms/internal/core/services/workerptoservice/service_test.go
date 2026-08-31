package workerptoservice

import (
	"context"
	"testing"

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
	create func(ctx context.Context, entity *worker.WorkerPTO) (*worker.WorkerPTO, error)
}

func (f *fakePTORepo) Create(
	ctx context.Context,
	entity *worker.WorkerPTO,
) (*worker.WorkerPTO, error) {
	return f.create(ctx, entity)
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
			create: func(_ context.Context, entity *worker.WorkerPTO) (*worker.WorkerPTO, error) {
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
			create: func(_ context.Context, entity *worker.WorkerPTO) (*worker.WorkerPTO, error) {
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
