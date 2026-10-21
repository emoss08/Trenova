package contentscanservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeScanner struct {
	result *contentscanports.ScanResult
	err    error
	calls  int
}

func (f *fakeScanner) Scan(context.Context, []byte) (*contentscanports.ScanResult, error) {
	f.calls++
	return f.result, f.err
}

func (f *fakeScanner) Ping(context.Context) error { return f.err }

func scanRequest() *contentscanports.ScanRequest {
	return &contentscanports.ScanRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Source:     contentscanports.SourceInboundAttachment,
		FileName:   "invoice.pdf",
		Content:    []byte("%PDF-1.7"),
	}
}

func TestCheck_RefusesAndRecordsAnInfectedFile(t *testing.T) {
	t.Parallel()

	auditService := auditportsmock.NewMockAuditService(t)
	var recorded *auditports.LogActionParams
	var comment string
	auditService.EXPECT().
		LogAction(mock.Anything, mock.Anything).
		RunAndReturn(func(params *auditports.LogActionParams, opts ...auditports.LogOption) error {
			recorded = params
			entry := &audit.Entry{}
			for _, opt := range opts {
				require.NoError(t, opt(entry))
			}
			comment = entry.Comment
			return nil
		}).
		Once()

	scanner := &fakeScanner{result: &contentscanports.ScanResult{
		Verdict:   contentscanports.VerdictInfected,
		Signature: "Win.Test.EICAR_HDB-1",
	}}
	gate := NewWithScanner(scanner, false, auditService, zap.NewNop())

	_, err := gate.Check(t.Context(), scanRequest())
	require.ErrorIs(t, err, contentscanports.ErrInfected)
	var infected *contentscanports.InfectedError
	require.ErrorAs(t, err, &infected)
	assert.Equal(t, "Win.Test.EICAR_HDB-1", infected.Signature)

	require.NotNil(t, recorded)
	assert.True(t, recorded.Critical)
	assert.Equal(t, permission.ResourceDocument, recorded.Resource)
	assert.Equal(t, permission.OpReject, recorded.Operation)
	assert.Equal(t, "Win.Test.EICAR_HDB-1", recorded.CurrentState["signature"])
	assert.Contains(t, comment, "invoice.pdf")

	var refusal *errortypes.Error
	require.ErrorAs(t, contentscanports.RefusalError("file", err), &refusal)
	assert.Equal(t, "file", refusal.Field)
	assert.Equal(t, []any{"Win.Test.EICAR_HDB-1"}, refusal.Args)
}

func TestCheck_HoldsAFileWhenTheScannerCannotCheckIt(t *testing.T) {
	t.Parallel()

	unavailable := &fakeScanner{
		err: errors.Join(contentscanports.ErrScannerUnavailable, errors.New("dial")),
	}

	_, err := NewWithScanner(
		unavailable,
		false,
		nil,
		zap.NewNop(),
	).Check(t.Context(), scanRequest())
	require.ErrorIs(t, err, contentscanports.ErrScannerUnavailable)
	assert.True(t, errortypes.IsBusinessError(contentscanports.RefusalError("file", err)))

	result, err := NewWithScanner(
		unavailable,
		true,
		nil,
		zap.NewNop(),
	).Check(t.Context(), scanRequest())
	require.NoError(t, err)
	assert.Equal(t, contentscanports.VerdictUnscanned, result.Verdict)
}

func TestCheck_PassesACleanFileAndLetsEverythingThroughWithoutAScanner(t *testing.T) {
	t.Parallel()

	clean := &fakeScanner{
		result: &contentscanports.ScanResult{Verdict: contentscanports.VerdictClean},
	}
	result, err := NewWithScanner(clean, false, nil, zap.NewNop()).Check(t.Context(), scanRequest())
	require.NoError(t, err)
	assert.Equal(t, contentscanports.VerdictClean, result.Verdict)
	assert.Equal(t, 1, clean.calls)

	result, err = NewWithScanner(nil, false, nil, zap.NewNop()).Check(t.Context(), scanRequest())
	require.NoError(t, err)
	assert.Equal(t, contentscanports.VerdictUnscanned, result.Verdict)

	result, err = contentscanports.Check(t.Context(), nil, scanRequest())
	require.NoError(t, err)
	assert.Equal(t, contentscanports.VerdictUnscanned, result.Verdict)
}

func TestCheck_RefusesAFileTooLargeToScanEvenWhenFailingOpen(t *testing.T) {
	t.Parallel()

	tooLarge := &fakeScanner{err: &contentscanports.TooLargeError{Size: 200 << 20, Limit: 100 << 20}}

	for _, failOpen := range []bool{false, true} {
		_, err := NewWithScanner(tooLarge, failOpen, nil, zap.NewNop()).
			Check(t.Context(), scanRequest())
		require.ErrorIs(t, err, contentscanports.ErrTooLarge, "failOpen %v", failOpen)
	}

	refusal := contentscanports.RefusalError("file", &contentscanports.TooLargeError{
		Size:  200 << 20,
		Limit: 100 << 20,
	})
	var validation *errortypes.Error
	require.ErrorAs(t, refusal, &validation)
	assert.Equal(t, "file", validation.Field)
	assert.Contains(t, refusal.Error(), "too large to be checked for malware")
	assert.Contains(t, refusal.Error(), "100")
	assert.NotContains(t, refusal.Error(), "Try again")

	unknownLimit := contentscanports.RefusalError("file", &contentscanports.TooLargeError{Size: 1})
	require.ErrorAs(t, unknownLimit, &validation)
	assert.Contains(t, unknownLimit.Error(), "too large to be checked for malware")
}
