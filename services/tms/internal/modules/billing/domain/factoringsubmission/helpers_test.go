package factoringsubmission_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

const now = int64(1_800_000_000)

var factorID = pulid.MustNew("fac_")

func fundedItem(t *testing.T) *factoringsubmission.SubmissionItem {
	t.Helper()
	item := &factoringsubmission.SubmissionItem{
		InvoiceNumber:        "INV-1",
		FaceMinor:            1000,
		ExpectedAdvanceMinor: 900,
		ExpectedFeeMinor:     30,
		Status:               factoringsubmission.SubmissionItemStatusPending,
	}
	require.NoError(t, item.Fund(now, item.ExpectedFunding()))
	return item
}
