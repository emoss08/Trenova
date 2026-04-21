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

func TestWorkItemLifecycle(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	work := &bankreceiptworkitem.WorkItem{
		ID:             pulid.MustNew("brwi_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         bankreceiptworkitem.StatusOpen,
	}
	repo := mocks.NewMockBankReceiptWorkItemRepository(t)
	var current *bankreceiptworkitem.WorkItem
	setCurrent := func(item *bankreceiptworkitem.WorkItem) {
		copy := *item
		current = &copy
	}
	setCurrent(work)
	repo.EXPECT().
		ListActive(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			_ pagination.TenantInfo,
		) ([]*bankreceiptworkitem.WorkItem, error) {
			if current == nil {
				return nil, nil
			}
			return []*bankreceiptworkitem.WorkItem{current}, nil
		}).
		Once()
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			_ repositories.GetBankReceiptWorkItemByIDRequest,
		) (*bankreceiptworkitem.WorkItem, error) {
			return current, nil
		}).
		Times(4)
	repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *bankreceiptworkitem.WorkItem,
		) (*bankreceiptworkitem.WorkItem, error) {
			copy := *entity
			current = &copy
			return &copy, nil
		}).
		Times(4)
	svc := New(Params{Logger: zap.NewNop(), Repo: repo, AuditService: &mocks.NoopAuditService{}})

	items, err := svc.ListActive(t.Context(), pagination.TenantInfo{OrgID: orgID, BuID: buID})
	require.NoError(t, err)
	require.Len(t, items, 1)

	assigned, err := svc.Assign(
		t.Context(),
		&serviceports.AssignBankReceiptWorkItemRequest{
			WorkItemID:       work.ID,
			AssignedToUserID: userID,
			TenantInfo:       pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	assert.Equal(t, bankreceiptworkitem.StatusAssigned, assigned.Status)

	review, err := svc.StartReview(
		t.Context(),
		&serviceports.GetBankReceiptWorkItemRequest{
			WorkItemID: work.ID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	assert.Equal(t, bankreceiptworkitem.StatusInReview, review.Status)

	resolved, err := svc.Resolve(
		t.Context(),
		&serviceports.ResolveBankReceiptWorkItemRequest{
			WorkItemID:     work.ID,
			ResolutionType: bankreceiptworkitem.ResolutionMatchedToPayment,
			ResolutionNote: "done",
			TenantInfo:     pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	assert.Equal(t, bankreceiptworkitem.StatusResolved, resolved.Status)

	setCurrent(&bankreceiptworkitem.WorkItem{
		ID:             pulid.MustNew("brwi_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         bankreceiptworkitem.StatusOpen,
	})
	dismissed, err := svc.Dismiss(
		t.Context(),
		&serviceports.DismissBankReceiptWorkItemRequest{
			WorkItemID:     current.ID,
			ResolutionNote: "ignore",
			TenantInfo:     pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	assert.Equal(t, bankreceiptworkitem.StatusDismissed, dismissed.Status)
}
