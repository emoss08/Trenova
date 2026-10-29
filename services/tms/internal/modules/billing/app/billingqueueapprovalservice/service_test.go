package billingqueueapprovalservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueapprovalservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type reviewRepo struct {
	billingqueueports.BillingQueueReviewRepository
}

func (reviewRepo) GetApprovalRunByKey(
	context.Context,
	pagination.TenantInfo,
	string,
) (*billingapprovalrun.ApprovalRun, error) {
	return nil, errortypes.NewNotFoundError("Approval run not found")
}

func (reviewRepo) CreateApprovalRun(
	_ context.Context,
	req *billingqueueports.CreateApprovalRunRequest,
) (*billingapprovalrun.ApprovalRun, error) {
	req.Run.ID = pulid.MustNew("bqar_")
	return req.Run, nil
}

func TestStartRequestsTheWorkflowByItsRegisteredName(t *testing.T) {
	t.Parallel()

	workflows := kernelportsmock.NewMockWorkflowStarter(t)
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

	_, err := svc.Start(t.Context(), &billingqueueports.StartBillingQueueApprovalRequest{
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
