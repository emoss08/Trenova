package rateconfirmation

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViaIsValid(t *testing.T) {
	assert.True(t, ViaDispatcher.IsValid())
	assert.True(t, ViaTenderAcceptance.IsValid())
	assert.True(t, ViaPublicSignature.IsValid())
	assert.False(t, Via("").IsValid())
	assert.False(t, Via("Robot").IsValid())
}

func TestCanSendIncludesConfirmed(t *testing.T) {
	for _, status := range []Status{StatusGenerated, StatusSent, StatusConfirmed} {
		assert.True(t, (&RateConfirmation{Status: status}).CanSend(), string(status))
	}
	assert.False(t, (&RateConfirmation{Status: StatusVoided}).CanSend())
}

func TestCanConfirmExcludesConfirmedAndVoided(t *testing.T) {
	assert.True(t, (&RateConfirmation{Status: StatusGenerated}).CanConfirm())
	assert.True(t, (&RateConfirmation{Status: StatusSent}).CanConfirm())
	assert.False(t, (&RateConfirmation{Status: StatusConfirmed}).CanConfirm())
	assert.False(t, (&RateConfirmation{Status: StatusVoided}).CanConfirm())
}

func validRateCon() *RateConfirmation {
	return &RateConfirmation{
		CarrierAssignmentID: pulid.MustNew("cass_"),
		CarrierID:           pulid.MustNew("car_"),
		ShipmentID:          pulid.MustNew("shp_"),
		ShipmentMoveID:      pulid.MustNew("smv_"),
	}
}

func TestValidateDefaultsGeneratedVia(t *testing.T) {
	entity := validRateCon()
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	require.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.Equal(t, ViaDispatcher, entity.GeneratedVia)
}

func TestValidateRejectsUnknownVia(t *testing.T) {
	entity := validRateCon()
	entity.GeneratedVia = Via("Robot")
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	entity = validRateCon()
	entity.ConfirmedVia = Via("Robot")
	multiErr = errortypes.NewMultiError()
	entity.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestTokenIsUsable(t *testing.T) {
	now := timeutils.NowUnix()
	base := func() *RateConfirmationToken {
		return &RateConfirmationToken{ExpiresAt: now + 3600}
	}

	assert.True(t, base().IsUsable(now))

	expired := base()
	expired.ExpiresAt = now - 1
	assert.False(t, expired.IsUsable(now))

	used := base()
	used.UsedAt = &now
	assert.False(t, used.IsUsable(now))

	revoked := base()
	revoked.RevokedAt = &now
	assert.False(t, revoked.IsUsable(now))

	var nilToken *RateConfirmationToken
	assert.False(t, nilToken.IsUsable(now))
}
