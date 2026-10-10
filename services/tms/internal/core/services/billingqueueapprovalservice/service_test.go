package billingqueueapprovalservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingqueueapprovalservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type reviewRepo struct {
	repositories.BillingQueueReviewRepository
}

func (reviewRepo) GetApprovalRunByKey(
	context.Context,
	pagination.TenantInfo,
	string,
) (*billingqueue.ApprovalRun, error) {
	return nil, errortypes.NewNotFoundError("Approval run not found")
}

func (reviewRepo) CreateApprovalRun(
	_ context.Context,
	req *repositories.CreateApprovalRunRequest,
) (*billingqueue.ApprovalRun, error) {
	req.Run.ID = pulid.MustNew("bqar_")
	return req.Run, nil
}

func TestStartRequestsTheWorkflowByItsRegisteredName(t *testing.T) {
	t.Parallel()

	workflows := mocks.NewMockWorkflowStarter(t)
	workflows.EXPECT().Enabled().Return(true)
	workflows.EXPECT().
		StartWorkflow(mock.Anything, mock.Anything, "BillingQueueBulkApprovalWorkflow", mock.Anything).
		Return(nil, nil).
		Once()

	svc := billingqueueapprovalservice.New(billingqueueapprovalservice.Params{
		Repo:      reviewRepo{},
		Workflows: workflows,
		Logger:    zap.NewNop(),
	})

	_, err := svc.Start(t.Context(), &services.StartBillingQueueApprovalRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		IdempotencyKey: "approve-1",
		ItemIDs:        []pulid.ID{pulid.MustNew("bqi_")},
	})

	require.NoError(t, err)
}
