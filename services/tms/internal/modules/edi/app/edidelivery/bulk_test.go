package edidelivery

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBulkRetryMessageDeliveryAggregatesFailures(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	missingID := pulid.MustNew("edimsg_")
	fixture.messageRepo.EXPECT().
		GetMessageByID(mock.Anything, mock.MatchedBy(func(req edimessageports.GetEDIMessageByIDRequest) bool {
			return req.ID == missingID
		})).
		Return(nil, errortypes.NewNotFoundError("EDIMessage not found")).
		Once()

	result, err := fixture.service.BulkRetryMessageDelivery(
		t.Context(),
		&BulkRetryMessageDeliveryRequest{
			TenantInfo: fixture.payload().TenantInfo,
			MessageIDs: []pulid.ID{missingID},
		},
	)
	require.NoError(t, err)
	require.Empty(t, result.Succeeded)
	require.Len(t, result.Failed, 1)
	require.Equal(t, missingID, result.Failed[0].ID)
	require.NotEmpty(t, result.Failed[0].Error)
}
