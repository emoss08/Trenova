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
	repo := &fakeBRWorkItemRepo{item: work}
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

	repo.item = &bankreceiptworkitem.WorkItem{
		ID:             pulid.MustNew("brwi_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         bankreceiptworkitem.StatusOpen,
	}
	dismissed, err := svc.Dismiss(
		t.Context(),
		&serviceports.DismissBankReceiptWorkItemRequest{
			WorkItemID:     repo.item.ID,
			ResolutionNote: "ignore",
			TenantInfo:     pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	assert.Equal(t, bankreceiptworkitem.StatusDismissed, dismissed.Status)
}

type fakeBRWorkItemRepo struct{ item *bankreceiptworkitem.WorkItem }

func (f *fakeBRWorkItemRepo) GetByID(
	context.Context,
	repositories.GetBankReceiptWorkItemByIDRequest,
) (*bankreceiptworkitem.WorkItem, error) {
	return f.item, nil
}

func (f *fakeBRWorkItemRepo) GetActiveByReceiptID(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*bankreceiptworkitem.WorkItem, error) {
	return f.item, nil
}

func (f *fakeBRWorkItemRepo) ListActive(
	context.Context,
	pagination.TenantInfo,
) ([]*bankreceiptworkitem.WorkItem, error) {
	if f.item == nil {
		return nil, nil
	}
	return []*bankreceiptworkitem.WorkItem{f.item}, nil
}

func (f *fakeBRWorkItemRepo) Create(
	_ context.Context,
	entity *bankreceiptworkitem.WorkItem,
) (*bankreceiptworkitem.WorkItem, error) {
	copy := *entity
	f.item = &copy
	return &copy, nil
}

func (f *fakeBRWorkItemRepo) Update(
	_ context.Context,
	entity *bankreceiptworkitem.WorkItem,
) (*bankreceiptworkitem.WorkItem, error) {
	copy := *entity
	f.item = &copy
	return &copy, nil
}
