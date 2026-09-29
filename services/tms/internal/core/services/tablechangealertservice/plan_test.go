package tablechangealertservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func ownedSubscription(owner pulid.ID) *tablechangealert.TCASubscription {
	return &tablechangealert.TCASubscription{
		ID:             pulid.MustNew("tcas_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         owner,
		Name:           "Delayed loads",
		TableName:      "shipments",
		EventTypes:     []string{"UPDATE"},
		ConditionMatch: "all",
		Priority:       "medium",
		Status:         tablechangealert.SubscriptionStatusActive,
		CreatedAt:      1_700_000_000,
		Version:        3,
	}
}

func TestPlanUpdateSubscription_ReadsOnlyTheCallersOwnAndChecksTheAllowlist(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	stored := ownedSubscription(owner)
	subs := mocks.NewMockTCASubscriptionRepository(t)
	subs.EXPECT().GetByID(mock.Anything, mock.MatchedBy(
		func(req repositories.GetTCASubscriptionByIDRequest) bool {
			return req.SubscriptionID == stored.ID && req.TenantInfo.UserID == owner
		},
	)).Return(stored, nil)
	allowlist := mocks.NewMockTCAAllowlistRepository(t)
	allowlist.EXPECT().IsTableAllowed(mock.Anything, "workers", mock.Anything).Return(false, nil)
	svc := &Service{l: zap.NewNop(), subRepo: subs, allowlistRepo: allowlist}

	edited := *stored
	edited.TableName = "workers"
	edited.CreatedAt = 0
	_, err := svc.PlanUpdateSubscription(t.Context(), &edited)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "tableName", multiErr.Errors[0].Field)
	subs.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestPlanSetSubscriptionStatus_PausesACopy(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	stored := ownedSubscription(owner)
	subs := mocks.NewMockTCASubscriptionRepository(t)
	subs.EXPECT().GetByID(mock.Anything, mock.Anything).Return(stored, nil)
	svc := &Service{l: zap.NewNop(), subRepo: subs}

	change, err := svc.PlanSetSubscriptionStatus(t.Context(), stored.ID, pagination.TenantInfo{
		OrgID: stored.OrganizationID, BuID: stored.BusinessUnitID, UserID: owner,
	}, tablechangealert.SubscriptionStatusPaused)
	require.NoError(t, err)
	assert.Equal(t, tablechangealert.SubscriptionStatusActive, change.Before.Status)
	assert.Equal(t, tablechangealert.SubscriptionStatusPaused, change.After.Status)
}
