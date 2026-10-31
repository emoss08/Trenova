package billingqueuerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAHeadlineLeadsWithWhatTheBillerCanAct(t *testing.T) {
	t.Parallel()

	asked := int64(1)
	headline := headlineOf([]*billingqueue.Issue{
		{
			Code:     billingqueue.IssueDetentionAwaitingApproval,
			CheckKey: billingqueue.CheckCharges,
			Summary:  "Detention waits on an approver",
		},
		{
			Code:        billingqueue.IssuePODUnsigned,
			CheckKey:    billingqueue.CheckPOD,
			Summary:     "The POD is unsigned",
			RequestedAt: &asked,
		},
		{
			Code:     billingqueue.IssuePossibleDuplicate,
			CheckKey: billingqueue.CheckDuplicate,
			Summary:  "INV-23988 already bills this shipment",
		},
	})

	require.NotNil(t, headline)
	assert.Equal(t, billingqueue.Headline{
		Code:     billingqueue.IssuePossibleDuplicate,
		CheckKey: billingqueue.CheckDuplicate,
		Summary:  "INV-23988 already bills this shipment",
		Open:     3,
		Waiting:  false,
	}, *headline)
}

func TestAHeadlineSaysWhenEveryIssueWaitsOnSomeoneElse(t *testing.T) {
	t.Parallel()

	asked := int64(1)
	headline := headlineOf([]*billingqueue.Issue{
		{
			Code:        billingqueue.IssuePODMissing,
			CheckKey:    billingqueue.CheckPOD,
			Summary:     "No POD yet",
			RequestedAt: &asked,
		},
		{
			Code:     billingqueue.IssueBillToCreditHold,
			CheckKey: billingqueue.CheckTerms,
			Summary:  "Acme is on credit hold",
		},
	})

	require.NotNil(t, headline)
	assert.Equal(t, "No POD yet", headline.Summary)
	assert.True(t, headline.Waiting)
	assert.Equal(t, 2, headline.Open)
}

func TestAnItemWithNoOpenIssueHasNoHeadline(t *testing.T) {
	t.Parallel()

	assert.Nil(t, headlineOf(nil))
}
