package document

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reviewableDocument() *Document {
	return &Document{IsCurrentVersion: true, Status: StatusActive}
}

func TestDocumentReject(t *testing.T) {
	t.Parallel()

	reviewer := pulid.MustNew("usr_")
	approvedAt := int64(100)
	doc := reviewableDocument()
	doc.ApprovedByID = pulid.MustNew("usr_")
	doc.ApprovedAt = &approvedAt

	require.NoError(t, doc.Reject(reviewer, 200, "  Signature is illegible  "))

	assert.Equal(t, StatusRejected, doc.Status)
	assert.Equal(t, reviewer, doc.RejectedByID)
	require.NotNil(t, doc.RejectedAt)
	assert.Equal(t, int64(200), *doc.RejectedAt)
	assert.Equal(t, "Signature is illegible", doc.RejectionReason)
	assert.True(t, doc.ApprovedByID.IsNil())
	assert.Nil(t, doc.ApprovedAt)
	assert.Equal(t, StandingRejected, doc.StandingAt(300))
}

func TestDocumentReject_RequiresReason(t *testing.T) {
	t.Parallel()

	for _, reason := range []string{"", "   ", strings.Repeat("x", MaxRejectionReasonLength+1)} {
		doc := reviewableDocument()
		err := doc.Reject(pulid.MustNew("usr_"), 200, reason)

		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr)
		assert.Equal(t, StatusActive, doc.Status)
	}
}

func TestDocumentReview_RefusesWhatCannotBeReviewed(t *testing.T) {
	t.Parallel()

	expired := int64(50)
	tests := []struct {
		name    string
		doc     *Document
		approve bool
	}{
		{name: "superseded version", doc: &Document{Status: StatusActive}},
		{name: "archived", doc: &Document{IsCurrentVersion: true, Status: StatusArchived}},
		{
			name: "already rejected",
			doc: &Document{
				IsCurrentVersion: true,
				Status:           StatusRejected,
			},
		},
		{
			name: "approve past expiration",
			doc: &Document{
				IsCurrentVersion: true,
				Status:           StatusActive,
				ExpirationDate:   &expired,
			},
			approve: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var err error
			if tt.approve {
				err = tt.doc.Approve(pulid.MustNew("usr_"), 100)
			} else {
				err = tt.doc.Reject(pulid.MustNew("usr_"), 100, "wrong load")
			}

			var business *errortypes.BusinessError
			require.ErrorAs(t, err, &business)
		})
	}
}

func TestDocumentApprove_ClearsRejection(t *testing.T) {
	t.Parallel()

	rejectedAt := int64(100)
	doc := &Document{
		IsCurrentVersion: true,
		Status:           StatusRejected,
		RejectedByID:     pulid.MustNew("usr_"),
		RejectedAt:       &rejectedAt,
		RejectionReason:  "blurred",
	}
	reviewer := pulid.MustNew("usr_")

	require.NoError(t, doc.Approve(reviewer, 200))

	assert.Equal(t, StatusActive, doc.Status)
	assert.Equal(t, reviewer, doc.ApprovedByID)
	require.NotNil(t, doc.ApprovedAt)
	assert.True(t, doc.RejectedByID.IsNil())
	assert.Nil(t, doc.RejectedAt)
	assert.Empty(t, doc.RejectionReason)
	assert.Equal(t, StandingAccepted, doc.StandingAt(300))

	var business *errortypes.BusinessError
	require.ErrorAs(t, doc.Approve(reviewer, 300), &business)
}

func TestDocumentStandingAt(t *testing.T) {
	t.Parallel()

	past := int64(100)
	future := int64(10_000)
	tests := []struct {
		doc  *Document
		want Standing
	}{
		{doc: &Document{Status: StatusActive}, want: StandingAccepted},
		{doc: &Document{Status: StatusActive, ExpirationDate: &future}, want: StandingAccepted},
		{doc: &Document{Status: StatusActive, ExpirationDate: &past}, want: StandingExpired},
		{doc: &Document{Status: StatusExpired}, want: StandingExpired},
		{doc: &Document{Status: StatusRejected}, want: StandingRejected},
		{doc: &Document{Status: StatusPendingApproval}, want: StandingPendingReview},
		{doc: &Document{Status: StatusPending}, want: StandingPendingReview},
		{doc: &Document{Status: StatusDraft}, want: StandingPendingReview},
		{doc: &Document{Status: StatusArchived}, want: StandingInactive},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.doc.StandingAt(1_000), string(tt.doc.Status))
	}
}
