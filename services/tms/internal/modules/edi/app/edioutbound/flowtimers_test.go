package edioutbound

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/testutil/mocks/ladingportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func dueInstance() ladingports.DueFlowInstance {
	return ladingports.DueFlowInstance{
		ID:             pulid.MustNew("efi_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

// An instance that cannot be woken is tried again next minute; within the
// run it is left out of later pages, so a page full of failures cannot hold
// up the instances behind it.
func TestDispatchDueFlowTimersLeavesFailuresOutOfLaterPages(t *testing.T) {
	t.Parallel()

	failing := make([]ladingports.DueFlowInstance, dueFlowPage)
	for i := range failing {
		failing[i] = dueInstance()
	}
	behind := dueInstance()
	notYet := time.Now().Add(time.Hour).Unix()

	repo := ladingportsmock.NewMockFlowRepository(t)
	repo.EXPECT().
		ListDueInstances(mock.Anything, mock.MatchedBy(func(req *ladingports.ListDueFlowInstancesRequest) bool {
			return len(req.Exclude) == 0
		})).
		Return(failing, nil).Once()
	repo.EXPECT().
		ListDueInstances(mock.Anything, mock.MatchedBy(func(req *ladingports.ListDueFlowInstancesRequest) bool {
			return len(req.Exclude) == dueFlowPage
		})).
		Return([]ladingports.DueFlowInstance{behind}, nil).Once()
	repo.EXPECT().
		GetInstance(mock.Anything, mock.MatchedBy(func(req ladingports.GetFlowInstanceRequest) bool {
			return req.ID != behind.ID
		})).
		Return(nil, errors.New("database unavailable")).Times(dueFlowPage)
	repo.EXPECT().
		GetInstance(mock.Anything, mock.MatchedBy(func(req ladingports.GetFlowInstanceRequest) bool {
			return req.ID == behind.ID
		})).
		Return(&ladingproject.FlowInstance{ID: behind.ID, WakeAt: &notYet}, nil).Once()

	svc := &Service{flowRepo: repo, l: zap.NewNop()}

	dispatched, err := svc.DispatchDueFlowTimers(t.Context(), time.Now().Unix())

	require.NoError(t, err)
	assert.Equal(t, 1, dispatched, "the instance behind the failures is reached")
}
