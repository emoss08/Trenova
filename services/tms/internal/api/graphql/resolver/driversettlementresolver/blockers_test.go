package driversettlementresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheBlockerListReadsTheNewestBatchTwentyFiveAtATimeUnlessAsked(t *testing.T) {
	t.Parallel()

	req, err := blockersRequest(nil, nil)
	require.NoError(t, err)
	assert.True(t, req.BatchID.IsNil())
	assert.Equal(t, settlementreleaseports.DefaultBlocked, req.Limit)

	batch := pulid.MustNew("dstlb_").String()
	limit := 100
	req, err = blockersRequest(&batch, &limit)
	require.NoError(t, err)
	assert.Equal(t, batch, req.BatchID.String())
	assert.Equal(t, 100, req.Limit)
}

func TestTheBlockerListRefusesALimitOutOfRangeAndABatchThatIsNotAnID(t *testing.T) {
	t.Parallel()

	for _, limit := range []int{0, 101} {
		_, err := blockersRequest(nil, &limit)
		require.Error(t, err, limit)
	}
	bad := "this week"
	_, err := blockersRequest(&bad, nil)
	require.Error(t, err)
}

func TestABlockerSaysWhatKeepsItInTheProductsWords(t *testing.T) {
	t.Parallel()

	batch := pulid.MustNew("dstlb_")
	rows := blockersToModel(context.Background(), []*settlementreleaseports.Settlement{
		{
			ID: pulid.MustNew("dstl_"), WorkerID: pulid.MustNew("wrk_"), BatchID: &batch,
			Status: driversettlement.StatusDraft, CarryForwardOutMinor: -333_879,
			Block: driversettlement.ReleaseBlockCriticalException,
			Exceptions: []driversettlement.Exception{{
				Code:     driversettlement.ExceptionCodeNegativeNet,
				Severity: driversettlement.ExceptionSeverityCritical,
				Message:  "Deductions exceed earnings; 3338.79 USD carries forward",
			}},
		},
		{
			ID: pulid.MustNew("dstl_"), WorkerID: pulid.MustNew("wrk_"),
			Status: driversettlement.StatusApproved, NetPayMinor: 98_500,
			Block:           driversettlement.ReleaseBlockDisputeOpen,
			DisputeCategory: driversettlement.DisputeCategoryIncorrectRate,
		},
	})

	require.Len(t, rows, 2)
	assert.Equal(t, "Has a critical exception", rows[0].Reason)
	assert.Equal(t, "Deductions exceed earnings; 3338.79 USD carries forward", rows[0].Message)
	assert.Equal(t, -333_879, *rows[0].CarryForwardOutMinor)
	require.NotNil(t, rows[0].BatchID)
	assert.Equal(t, "Dispute open", rows[1].Reason)
	assert.Equal(t, "about the rate", rows[1].Message)
	assert.Nil(t, rows[1].BatchID)
	assert.Equal(t, 98_500, *rows[1].NetPayMinor)
}
