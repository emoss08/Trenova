package factoring_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoaLifecycle(t *testing.T) {
	t.Parallel()

	noa := &factoring.Noa{Status: factoring.NoaStatusDraft}
	require.Error(t, noa.MarkSent(now, nil), "a notice needs a recipient")

	require.NoError(t, noa.MarkSent(now, []string{"ap@customer.example.com", "cfo@customer.example.com"}))
	assert.Equal(t, factoring.NoaStatusSent, noa.Status)
	assert.Equal(t, "ap@customer.example.com, cfo@customer.example.com", noa.SentTo)
	require.NotNil(t, noa.SentAt)

	require.NoError(t, noa.MarkSent(now+10, []string{"ap@customer.example.com"}),
		"a sent notice can be sent again")

	err := noa.Acknowledge(now, now+3600, pulid.MustNew("doc_"), nil)
	require.Error(t, err, "a signature cannot be dated in the future")
	assert.True(t, errortypes.IsError(err))
	require.Error(t, noa.Acknowledge(now, now-60, pulid.Nil, nil), "the signed copy is required")

	expires := now + 90*86400
	require.NoError(t, noa.Acknowledge(now, now-60, pulid.MustNew("doc_"), &expires))
	assert.Equal(t, factoring.NoaStatusAcknowledged, noa.Status)
	assert.True(t, noa.IsEffective(now))
	assert.False(t, noa.IsEffective(expires))
	assert.True(t, noa.ExpiresWithin(expires-10*86400, factoring.NoaExpiryWarningDays))
	assert.False(t, noa.CanSend(), "an acknowledged notice is not sent again")

	require.Error(t, noa.Release(now, "  "), "a release needs a reason")
	require.NoError(t, noa.Release(now, "Factor's release letter of 1 March"))
	assert.Equal(t, factoring.NoaStatusReleased, noa.Status)
	assert.False(t, noa.IsOpen())
	require.Error(t, noa.Release(now, "again"), "a released notice stays released")
}

func TestNoaSentToFitsTheColumn(t *testing.T) {
	t.Parallel()

	recipients := make([]string, 0, 40)
	for range 40 {
		recipients = append(recipients, "accounts.payable.department@customer.example.com")
	}
	noa := &factoring.Noa{Status: factoring.NoaStatusDraft}
	require.NoError(t, noa.MarkSent(now, recipients))
	assert.Equal(t, recipients[0], noa.SentTo)
}

func TestAgreementInForce(t *testing.T) {
	t.Parallel()

	ends := now + 100
	a := &factoring.Agreement{
		Status:        factoring.AgreementStatusActive,
		EffectiveFrom: now - 100,
		EffectiveTo:   &ends,
		Recourse:      factoring.RecourseNone,
	}
	assert.True(t, a.InForce(now))
	assert.False(t, a.InForce(ends), "the end is exclusive")
	assert.False(t, a.InForce(now-101))
	assert.True(t, a.IsNonRecourse())

	var none *factoring.Agreement
	assert.False(t, none.InForce(now))
	assert.False(t, none.IsNonRecourse())
}

func TestNoaSentToSeveralRecipientsStaysValid(t *testing.T) {
	t.Parallel()

	noa := &factoring.Noa{
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		CustomerID:         pulid.MustNew("cus_"),
		FactoringCompanyID: pulid.MustNew("fc_"),
		Status:             factoring.NoaStatusDraft,
	}
	require.NoError(t, noa.MarkSent(now, []string{"ap@customer.example.com", "cfo@customer.example.com"}))

	multiErr := errortypes.NewMultiError()
	noa.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), "a notice sent to two people must save: %v", multiErr)

	noa.SentTo = "ap@customer.example.com, not-an-address"
	multiErr = errortypes.NewMultiError()
	noa.Validate(multiErr)
	assert.True(t, multiErr.HasErrors(), "one bad recipient in the list is refused")
}
