package driversettlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func heldFixture(t *testing.T, status driversettlement.Status) *driversettlement.Settlement {
	t.Helper()

	entity := planFixture(status)
	require.NoError(t, PlanHold(entity, &HoldRequest{
		Reason: driversettlement.HoldReasonRateDispute,
		Note:   "Waiting on the broker's corrected rate",
	}, pulid.MustNew("usr_"), 1_790_000_000))

	return entity
}

func requireHeldRefusal(t *testing.T, err error) {
	t.Helper()

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "holdReason", validation.Field())
	assert.Equal(t, errortypes.ErrInvalidOperation, validation.Code)
	assert.Equal(t,
		"Settlement DS-100 is on hold for a rate dispute; release the hold first",
		err.Error())
}

func TestEveryPathThatAdvancesASettlementRefusesAHeldOne(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	const now = int64(1_790_000_100)

	approving := heldFixture(t, driversettlement.StatusPendingApproval)
	requireHeldRefusal(t, PlanApprove(approving, userID, now))
	assert.Equal(t, driversettlement.StatusPendingApproval, approving.Status)
	assert.Nil(t, approving.ApprovedAt)

	posting := heldFixture(t, driversettlement.StatusApproved)
	requireHeldRefusal(t, PlanPost(posting, userID, now))
	assert.Equal(t, driversettlement.StatusApproved, posting.Status)
	assert.Nil(t, posting.PostedAt)

	paying := heldFixture(t, driversettlement.StatusApproved)
	paying.Status = driversettlement.StatusPosted
	requireHeldRefusal(t, PlanMarkPaid(paying, &MarkPaidInput{
		PaymentMethod: "ACH", PaidAt: now, UserID: userID,
	}))
	assert.Nil(t, paying.PaidAt)
}

func TestAHeldSettlementMayStillBeSubmittedSentBackAndVoided(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	const now = int64(1_790_000_200)

	submitting := heldFixture(t, driversettlement.StatusDraft)
	require.NoError(t, PlanSubmit(submitting, userID, now))
	assert.True(t, submitting.IsHeld(), "submitting asks for review; it does not release")

	rejecting := heldFixture(t, driversettlement.StatusPendingApproval)
	require.NoError(t, PlanReject(rejecting, "Wrong rate"))
	assert.True(t, rejecting.IsHeld())

	voiding := heldFixture(t, driversettlement.StatusApproved)
	require.NoError(t, PlanVoid(voiding, "Duplicate", userID, now))
	assert.Equal(t, driversettlement.StatusVoided, voiding.Status)
	assert.False(t, voiding.IsHeld(), "a voided settlement is out of the pay run for good")
	assert.Nil(t, voiding.HoldReleasedAt, "voiding is not a release")
}

func TestPlanHoldPlacesAndReleases(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	entity := planFixture(driversettlement.StatusApproved)
	require.NoError(t, PlanHold(entity, &HoldRequest{
		Reason: driversettlement.HoldReasonMissingPaperwork,
	}, userID, 10))
	assert.Equal(t, driversettlement.HoldReasonMissingPaperwork, entity.HoldReason)
	assert.Equal(t, userID, entity.HeldByID)

	require.NoError(t, PlanHold(entity, &HoldRequest{Release: true}, userID, 20))
	assert.False(t, entity.IsHeld())
	assert.Equal(t, userID, entity.HoldReleasedByID)

	err := PlanHold(entity, &HoldRequest{Release: true}, userID, 30)
	require.Error(t, err)

	posted := planFixture(driversettlement.StatusPosted)
	err = PlanHold(posted, &HoldRequest{Reason: driversettlement.HoldReasonRateDispute}, userID, 40)
	require.Error(t, err)
	assert.False(t, posted.IsHeld())
}

func TestCloneSettlementCarriesTheHoldWithoutSharingIt(t *testing.T) {
	t.Parallel()

	held := heldFixture(t, driversettlement.StatusApproved)
	clone := CloneSettlement(held)
	require.NoError(t, PlanHold(clone, &HoldRequest{Release: true}, pulid.MustNew("usr_"), 50))
	assert.True(t, held.IsHeld(), "planning a release leaves the read settlement held")
	assert.False(t, clone.IsHeld())
}

func TestTheAuditCommentSaysWhatWasHeldAndReleased(t *testing.T) {
	t.Parallel()

	held := heldFixture(t, driversettlement.StatusApproved)
	assert.Equal(t, "Settlement held: missing paperwork: BOL missing",
		holdComment(&HoldRequest{
			Reason: driversettlement.HoldReasonMissingPaperwork,
			Note:   " BOL missing ",
		}, held))
	assert.Equal(t, "Settlement held: negative net",
		holdComment(&HoldRequest{Reason: driversettlement.HoldReasonNegativeNet}, held))
	assert.Equal(t, "Settlement hold released (rate dispute)",
		holdComment(&HoldRequest{Release: true}, held))
}
