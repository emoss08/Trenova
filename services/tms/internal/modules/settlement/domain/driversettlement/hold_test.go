package driversettlement

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func holdFixture(status Status) *Settlement {
	return &Settlement{
		ID:               pulid.MustNew("dstl_"),
		SettlementNumber: "DS-204",
		Status:           status,
	}
}

func holdInput(reason HoldReason, note string) *HoldInput {
	return &HoldInput{Reason: reason, Note: note, UserID: pulid.MustNew("usr_"), At: 1_700_000_000}
}

func requireCode(t *testing.T, err error, code errortypes.ErrorCode, field string) {
	t.Helper()

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, code, validation.Code)
	assert.Equal(t, field, validation.Field())
}

func TestASettlementCanBeHeldUntilItIsPosted(t *testing.T) {
	t.Parallel()

	for _, status := range []Status{StatusDraft, StatusPendingApproval, StatusApproved} {
		entity := holdFixture(status)
		input := holdInput(HoldReasonRateDispute, "  Waiting on the broker's corrected rate  ")
		require.NoError(t, entity.PlaceHold(input), status)
		assert.True(t, entity.IsHeld())
		assert.Equal(t, status, entity.Status, "the hold keeps the status")
		assert.Equal(t, HoldReasonRateDispute, entity.HoldReason)
		assert.Equal(t, "Waiting on the broker's corrected rate", entity.HoldNote)
		assert.Equal(t, input.UserID, entity.HeldByID)
		require.NotNil(t, entity.HeldAt)
		assert.Equal(t, input.At, *entity.HeldAt)
	}
}

func TestAPostedPaidOrVoidedSettlementCannotBeHeld(t *testing.T) {
	t.Parallel()

	for _, status := range []Status{StatusPosted, StatusPaid, StatusVoided} {
		entity := holdFixture(status)
		err := entity.PlaceHold(holdInput(HoldReasonMissingPaperwork, ""))
		requireCode(t, err, errortypes.ErrInvalidOperation, "status")
		assert.False(t, entity.IsHeld(), status)
	}
	err := holdFixture(StatusPosted).PlaceHold(holdInput(HoldReasonMissingPaperwork, ""))
	assert.Contains(t, err.Error(), "void it instead")
}

func TestAHoldNeedsAReasonFromTheSetAndANoteForOther(t *testing.T) {
	t.Parallel()

	err := holdFixture(StatusDraft).PlaceHold(holdInput(HoldReason("Vacation"), ""))
	requireCode(t, err, errortypes.ErrInvalid, "holdReason")

	err = holdFixture(StatusDraft).PlaceHold(holdInput(HoldReasonOther, "   "))
	requireCode(t, err, errortypes.ErrRequired, "holdNote")

	err = holdFixture(StatusDraft).PlaceHold(
		holdInput(HoldReasonDeductionReview, strings.Repeat("é", MaxHoldNoteLength+1)),
	)
	requireCode(t, err, errortypes.ErrInvalid, "holdNote")

	require.NoError(t, holdFixture(StatusDraft).PlaceHold(
		holdInput(HoldReasonDeductionReview, strings.Repeat("é", MaxHoldNoteLength)),
	))
	require.NoError(t, holdFixture(StatusDraft).PlaceHold(holdInput(HoldReasonOther, "Garnishment")))
}

func TestAHeldSettlementIsNotHeldAgainAndAFreeOneIsNotReleased(t *testing.T) {
	t.Parallel()

	entity := holdFixture(StatusApproved)
	require.NoError(t, entity.PlaceHold(holdInput(HoldReasonNegativeNet, "")))
	err := entity.PlaceHold(holdInput(HoldReasonRateDispute, ""))
	requireCode(t, err, errortypes.ErrInvalidOperation, "holdReason")
	assert.Equal(t, HoldReasonNegativeNet, entity.HoldReason, "the first hold stands")

	err = holdFixture(StatusDraft).ReleaseHold(pulid.MustNew("usr_"), 1)
	requireCode(t, err, errortypes.ErrInvalidOperation, "holdReason")
	assert.Contains(t, err.Error(), "DS-204 is not on hold")
}

func TestReleasingAHoldClearsItAndSaysWhoReleasedIt(t *testing.T) {
	t.Parallel()

	entity := holdFixture(StatusPendingApproval)
	require.NoError(t, entity.PlaceHold(holdInput(HoldReasonRateDispute, "Rate con")))
	releaser := pulid.MustNew("usr_")
	require.NoError(t, entity.ReleaseHold(releaser, 1_700_000_500))

	assert.False(t, entity.IsHeld())
	assert.Empty(t, entity.HoldNote)
	assert.Nil(t, entity.HeldAt)
	assert.True(t, entity.HeldByID.IsNil())
	assert.Equal(t, releaser, entity.HoldReleasedByID)
	require.NotNil(t, entity.HoldReleasedAt)
	assert.Equal(t, int64(1_700_000_500), *entity.HoldReleasedAt)
	assert.Equal(t, StatusPendingApproval, entity.Status)

	require.NoError(t, entity.PlaceHold(holdInput(HoldReasonOther, "Garnishment order")))
	assert.Nil(t, entity.HoldReleasedAt, "a new hold clears the last release")
}

func TestAHeldSettlementIsRefusedWithItsReason(t *testing.T) {
	t.Parallel()

	assert.NoError(t, holdFixture(StatusApproved).RefuseIfHeld())

	cases := map[HoldReason]string{
		HoldReasonRateDispute:      "Settlement DS-204 is on hold for a rate dispute; release the hold first",
		HoldReasonMissingPaperwork: "Settlement DS-204 is on hold for missing paperwork; release the hold first",
		HoldReasonNegativeNet:      "Settlement DS-204 is on hold for a negative net; release the hold first",
		HoldReasonDeductionReview:  "Settlement DS-204 is on hold while a deduction is reviewed; release the hold first",
		HoldReasonOther:            "Settlement DS-204 is on hold; release the hold first",
	}
	for reason, message := range cases {
		entity := holdFixture(StatusApproved)
		require.NoError(t, entity.PlaceHold(holdInput(reason, "Paperwork")))
		err := entity.RefuseIfHeld()
		requireCode(t, err, errortypes.ErrInvalidOperation, "holdReason")
		assert.Equal(t, message, err.Error())
	}
}

func TestValidationRejectsAHoldThatIsNotWhole(t *testing.T) {
	t.Parallel()

	at := int64(1)
	cases := map[string]*Settlement{
		"reason without a moment": {Status: StatusDraft, HoldReason: HoldReasonRateDispute},
		"moment without a reason": {Status: StatusDraft, HeldAt: &at},
		"held after posting": {
			Status: StatusPosted, HoldReason: HoldReasonRateDispute, HeldAt: &at,
		},
		"a reason outside the set": {Status: StatusDraft, HoldReason: "Vacation", HeldAt: &at},
	}
	for name, entity := range cases {
		multiErr := errortypes.NewMultiError()
		entity.validateHold(multiErr)
		assert.True(t, multiErr.HasErrors(), name)
	}

	multiErr := errortypes.NewMultiError()
	(&Settlement{Status: StatusApproved, HoldReason: HoldReasonRateDispute, HeldAt: &at}).
		validateHold(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestTheHoldComesFirstAmongWhatKeepsASettlementFromRelease(t *testing.T) {
	t.Parallel()

	blocks := ReleaseBlocks()
	require.NotEmpty(t, blocks)
	assert.Equal(t, ReleaseBlockHeld, blocks[0])
	assert.True(t, ReleaseBlockHeld.IsValid())
	assert.Equal(t, UnreleasedStatuses(), HoldableStatuses())
}
