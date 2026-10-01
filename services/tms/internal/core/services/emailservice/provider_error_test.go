package emailservice

import (
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/email"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/require"
)

func TestResendSenderRejection(t *testing.T) {
	t.Parallel()

	body := []byte(
		`{"statusCode":403,"message":"The trenova.example.com domain is not verified. Please, add and verify your domain on https://resend.com/domains","name":"validation_error"}`,
	)
	rejected := resendSenderRejection(http.StatusForbidden, body)
	require.NotNil(t, rejected)
	require.Equal(t, email.ProviderResend, rejected.Provider)
	require.Equal(t, SenderDomainUnverified, rejected.Rejection)

	require.Nil(t, resendSenderRejection(http.StatusForbidden, []byte(`{"message":"API key is invalid"}`)))
	require.Nil(t, resendSenderRejection(http.StatusBadRequest, body))
	require.Nil(t, resendSenderRejection(http.StatusForbidden, []byte(`not json`)))
}

func TestPostmarkSenderRejection(t *testing.T) {
	t.Parallel()

	missing := postmarkSenderRejection(
		http.StatusUnprocessableEntity,
		[]byte(`{"ErrorCode":400,"Message":"The 'From' address you supplied is not a Sender Signature on your account."}`),
	)
	require.NotNil(t, missing)
	require.Equal(t, SenderSignatureMissing, missing.Rejection)

	unconfirmed := postmarkSenderRejection(
		http.StatusUnprocessableEntity,
		[]byte(`{"ErrorCode":401,"Message":"Sender signature not confirmed."}`),
	)
	require.NotNil(t, unconfirmed)
	require.Equal(t, SenderSignatureUnconfirmed, unconfirmed.Rejection)

	require.Nil(t, postmarkSenderRejection(
		http.StatusUnprocessableEntity,
		[]byte(`{"ErrorCode":300,"Message":"Invalid email request"}`),
	))
}

func TestSenderRejectedErrorMessage(t *testing.T) {
	t.Parallel()

	err := &SenderRejectedError{
		Provider:  email.ProviderResend,
		Rejection: SenderDomainUnverified,
		Address:   "billing@trenova.example.com",
		Origin:    "the From address on the Email profile tab of customer ACME Manufacturing",
	}
	require.Equal(
		t,
		"Resend refused to send from billing@trenova.example.com because the domain trenova.example.com is not verified in your Resend account. This address comes from the From address on the Email profile tab of customer ACME Manufacturing. Change it to an address on a domain verified in Resend, or verify trenova.example.com in Resend.",
		err.Error(),
	)
	require.True(t, errors.Is(err, serviceports.ErrNonRetryableEmailSend))

	err.Origin = ""
	require.NotContains(t, err.Error(), "comes from")
}
