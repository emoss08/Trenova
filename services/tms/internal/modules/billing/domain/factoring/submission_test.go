package factoring_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sendingSubmission() *factoring.Submission {
	sub := &factoring.Submission{
		OrganizationID:     "org_1",
		BusinessUnitID:     "bu_1",
		Number:             "FS2610000001",
		FactoringCompanyID: factorID,
		AgreementID:        "facagr_1",
		Status:             factoring.SubmissionStatusSending,
		TransportMethod:    factoring.TransportMethodEmail,
		TransportTarget:    "schedules@factor.test",
		IdempotencyKey:     "key",
		SendAttemptGroup:   1,
		Items: []*factoring.SubmissionItem{
			{InvoiceID: "inv_1", FaceMinor: 1000, ExpectedAdvanceMinor: 900, ExpectedFeeMinor: 30, ExpectedReserveMinor: 100, Status: factoring.SubmissionItemStatusPending},
			{InvoiceID: "inv_2", FaceMinor: 2000, ExpectedAdvanceMinor: 1800, ExpectedFeeMinor: 60, ExpectedReserveMinor: 200, Status: factoring.SubmissionItemStatusPending},
		},
	}
	sub.Totalize()
	return sub
}

func TestSubmissionTotalizeAndValidate(t *testing.T) {
	t.Parallel()

	sub := sendingSubmission()
	assert.Equal(t, 2, sub.InvoiceCount)
	assert.Equal(t, int64(3000), sub.FaceMinor)
	assert.Equal(t, int64(2700), sub.ExpectedAdvanceMinor)
	assert.Equal(t, int64(90), sub.ExpectedFeeMinor)
	assert.Equal(t, int64(300), sub.ExpectedReserveMinor)

	multiErr := errortypes.NewMultiError()
	sub.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())

	sub.Items = nil
	sub.Totalize()
	multiErr = errortypes.NewMultiError()
	sub.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestSubmissionLifecycle(t *testing.T) {
	t.Parallel()

	sub := sendingSubmission()
	require.NoError(t, sub.MarkFailed(now, strings.Repeat("x", 3000), 6))
	assert.Equal(t, factoring.SubmissionStatusFailed, sub.Status)
	assert.Len(t, sub.LastError, factoring.MaxSubmissionErrorLength)
	assert.True(t, sub.IsLive())

	require.NoError(t, sub.Retry())
	assert.Equal(t, factoring.SubmissionStatusSending, sub.Status)
	assert.Equal(t, 2, sub.SendAttemptGroup)
	assert.Nil(t, sub.FailedAt)

	require.NoError(t, sub.MarkSent(now+60, "<msg@factor>", 1))
	assert.Equal(t, factoring.SubmissionStatusSent, sub.Status)
	assert.Empty(t, sub.LastError)
	require.Error(t, sub.Cancel(now, "no"))
	require.Error(t, sub.Retry())
	require.Error(t, sub.MarkFailed(now, "late", 1))
}

func TestSubmissionCancelReleasesItems(t *testing.T) {
	t.Parallel()

	sub := sendingSubmission()
	require.Error(t, sub.Cancel(now, "too early"))
	require.NoError(t, sub.MarkFailed(now, "", 6))
	assert.Equal(t, "The factor could not be reached", sub.LastError)
	require.Error(t, sub.Cancel(now, "  "))
	require.NoError(t, sub.Cancel(now, "Factor closed the account"))
	assert.Equal(t, factoring.SubmissionStatusCancelled, sub.Status)
	assert.False(t, sub.IsLive())
	for _, item := range sub.Items {
		assert.Equal(t, factoring.SubmissionItemStatusReleased, item.Status)
		assert.NotNil(t, item.ReleasedAt)
	}
	assert.Equal(t, []string{"inv_1", "inv_2"}, idsOf(sub))
}

func idsOf(sub *factoring.Submission) []string {
	out := make([]string, 0, len(sub.Items))
	for _, id := range sub.InvoiceIDs() {
		out = append(out, id.String())
	}
	return out
}

func TestTransportNormalizeAndValidate(t *testing.T) {
	t.Parallel()

	email := &factoring.Transport{
		OrganizationID:     "org_1",
		BusinessUnitID:     "bu_1",
		FactoringCompanyID: factorID,
		Method:             factoring.TransportMethodEmail,
		EmailRecipients:    []string{" a@factor.test ", "A@factor.test", ""},
		SftpHost:           "left.over",
	}
	email.Normalize()
	assert.Equal(t, []string{"a@factor.test"}, email.EmailRecipients)
	assert.Empty(t, email.SftpHost)
	multiErr := errortypes.NewMultiError()
	email.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
	assert.Equal(t, "a@factor.test", email.Target())

	sftp := &factoring.Transport{
		OrganizationID:     "org_1",
		BusinessUnitID:     "bu_1",
		FactoringCompanyID: factorID,
		Method:             factoring.TransportMethodSftp,
		EmailRecipients:    []string{"a@factor.test"},
		SftpHost:           "sftp.factor.test",
		SftpUsername:       "carrier",
		SftpAuthMethod:     factoring.SftpAuthPassword,
		SftpSecret:         "sealed",
		SftpKnownHostKey:   "ssh-ed25519 AAAA",
		SftpDirectory:      "/in/schedules/",
	}
	sftp.Normalize()
	assert.Nil(t, sftp.EmailRecipients)
	assert.Equal(t, 22, sftp.SftpPort)
	assert.Equal(t, "/in/schedules", sftp.SftpDirectory)
	multiErr = errortypes.NewMultiError()
	sftp.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
	assert.Equal(t, "carrier@sftp.factor.test:/in/schedules", sftp.Target())

	sftp.SftpDirectory = "relative/../dir"
	sftp.SftpHost = "sftp://bad"
	sftp.SftpSecret = ""
	multiErr = errortypes.NewMultiError()
	sftp.Validate(multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	assert.Contains(t, fields, "sftpHost")
	assert.Contains(t, fields, "sftpSecret")
	assert.Contains(t, fields, "sftpDirectory")
}
