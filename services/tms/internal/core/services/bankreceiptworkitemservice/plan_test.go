package bankreceiptworkitemservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPlansShowTheMoveWithoutSavingIt(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	assignee := pulid.MustNew("usr_")
	work := &bankreceiptworkitem.WorkItem{
		ID:             pulid.MustNew("brwi_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         bankreceiptworkitem.StatusOpen,
	}
	repo := mocks.NewMockBankReceiptWorkItemRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).RunAndReturn(
		func(
			_ context.Context,
			_ repositories.GetBankReceiptWorkItemByIDRequest,
		) (*bankreceiptworkitem.WorkItem, error) {
			copied := *work
			return &copied, nil
		},
	).Times(2)
	svc := New(Params{Logger: zap.NewNop(), Repo: repo, AuditService: &mocks.NoopAuditService{}})
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	actor := testutil.NewSessionActor(userID, orgID, buID)

	assigned, err := svc.PlanAssign(t.Context(), &serviceports.AssignBankReceiptWorkItemRequest{
		WorkItemID:       work.ID,
		AssignedToUserID: assignee,
		TenantInfo:       tenant,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, bankreceiptworkitem.StatusOpen, assigned.Before.Status)
	assert.Equal(t, bankreceiptworkitem.StatusAssigned, assigned.After.Status)
	assert.Equal(t, assignee, assigned.After.AssignedToUserID)

	review, err := svc.PlanStartReview(t.Context(), &serviceports.GetBankReceiptWorkItemRequest{
		WorkItemID: work.ID,
		TenantInfo: tenant,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, bankreceiptworkitem.StatusInReview, review.After.Status)
	assert.Equal(t, bankreceiptworkitem.StatusOpen, work.Status)
}
