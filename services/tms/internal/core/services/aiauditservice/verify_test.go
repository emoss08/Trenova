package aiauditservice

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeNotifier struct {
	mu       sync.Mutex
	permits  []notificationservice.NotifyPermittedRequest
	personal []*notification.Notification
}

func (f *fakeNotifier) NotifyPermitted(
	_ context.Context,
	req notificationservice.NotifyPermittedRequest,
) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.permits = append(f.permits, req)

	return 1, nil
}

func (f *fakeNotifier) Create(
	_ context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.personal = append(f.personal, entity)

	return entity, nil
}

type fakeAudit struct {
	serviceports.AuditService

	mu      sync.Mutex
	entries []*serviceports.LogActionParams
}

func (f *fakeAudit) LogAction(
	params *serviceports.LogActionParams,
	_ ...serviceports.LogOption,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, params)

	return nil
}

func testVerifier(ledger *fakeLedger, keyring *Keyring, notifier *fakeNotifier) *Verifier {
	params := VerifierParams{
		Ledger:  ledger,
		Keyring: keyring,
		Audit:   &fakeAudit{},
		Now:     func() time.Time { return time.Unix(testNow, 0) },
		Logger:  zap.NewNop(),
	}
	if notifier != nil {
		params.Notifier = notifier
	}

	return NewVerifier(&params)
}

func TestVerifier_AnUntouchedChainVerifies(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)
	verifier := testVerifier(ledger, testKeyring(true), &fakeNotifier{})

	result, err := verifier.VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)

	assert.Equal(t, aiaudit.VerificationVerified, result.Status)
	head, err := ledger.GetChainHead(t.Context(), s.tenant)
	require.NoError(t, err)
	assert.Equal(t, head.LastSeq, result.VerifiedSeq)
	assert.Equal(t, aiaudit.VerificationVerified, head.LastVerificationStatus)
}

func TestVerifier_AnEditedRowFailsAtItsSeq(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)
	rows := ledger.all(s.tenant)
	require.Greater(t, len(rows), 5)
	tampered := rows[4]
	tampered.Outcome = aiaudit.OutcomeFailed

	notifier := &fakeNotifier{}
	verifier := testVerifier(ledger, testKeyring(true), notifier)
	result, err := verifier.VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)

	assert.Equal(t, aiaudit.VerificationMismatch, result.Status)
	require.NotNil(t, result.FailedSeq)
	assert.Equal(t, tampered.Seq, *result.FailedSeq)
	assert.Equal(t, tampered.Seq-1, result.VerifiedSeq)

	require.Len(t, notifier.permits, 1)
	assert.Equal(t, permission.ResourceAIAuditTrail, notifier.permits[0].Resource)
	assert.Equal(t, notification.PriorityCritical, notifier.permits[0].Notification.Priority)
	audited := verifier.audit.(*fakeAudit).entries
	require.Len(t, audited, 1)
	assert.True(t, audited[0].Critical)
}

func TestVerifier_ASignedChainRewrittenAsUnsignedFails(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)
	rows := ledger.all(s.tenant)
	require.Greater(t, len(rows), 5)

	prev := rows[3].Hash
	for _, row := range rows[4:] {
		row.Outcome = aiaudit.OutcomeFailed
		require.NoError(t, aiaudit.Seal(row, prev, nil))
		prev = row.Hash
	}
	ledger.mu.Lock()
	for _, seal := range ledger.seals[s.tenant] {
		for _, row := range rows {
			if row.Seq == seal.ToSeq {
				seal.HeadHash = row.Hash
			}
		}
	}
	ledger.heads[s.tenant].LastHash = prev
	ledger.mu.Unlock()

	verifier := testVerifier(ledger, testKeyring(true), &fakeNotifier{})
	result, err := verifier.VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)

	assert.Equal(t, aiaudit.VerificationMismatch, result.Status)
	require.NotNil(t, result.FailedSeq)
	assert.Equal(t, rows[4].Seq, *result.FailedSeq)
}

func TestVerifier_AnUnsignedHistoryFollowedBySignedRowsVerifies(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, false)
	rows := ledger.all(s.tenant)
	require.Greater(t, len(rows), 5)

	keyring := testKeyring(true)
	key, ok := keyring.Key(keyring.ActiveKeyID())
	require.True(t, ok)
	prev := rows[3].Hash
	for _, row := range rows[4:] {
		require.NoError(t, aiaudit.Seal(row, prev, key))
		prev = row.Hash
	}
	ledger.mu.Lock()
	for _, seal := range ledger.seals[s.tenant] {
		for _, row := range rows {
			if row.Seq == seal.ToSeq {
				seal.HeadHash = row.Hash
			}
		}
	}
	ledger.heads[s.tenant].LastHash = prev
	ledger.mu.Unlock()

	verifier := testVerifier(ledger, keyring, &fakeNotifier{})
	result, err := verifier.VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)

	assert.Equal(t, aiaudit.VerificationVerified, result.Status)
}

func TestVerifier_ARemovedRowFailsTheChain(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)
	rows := ledger.events[s.tenant]
	ledger.events[s.tenant] = append(rows[:3:3], rows[4:]...)

	result, err := testVerifier(
		ledger,
		testKeyring(true),
		nil,
	).VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)

	assert.Equal(t, aiaudit.VerificationMismatch, result.Status)
	assert.Equal(t, int64(4), *result.FailedSeq)
}

func TestVerifier_ARowSignedWithAnUnknownKeyIsReported(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)

	result, err := testVerifier(
		ledger,
		testKeyring(false),
		nil,
	).VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)

	assert.Equal(t, aiaudit.VerificationKeyMissing, result.Status)
	assert.Equal(t, int64(1), *result.FailedSeq)
}

func TestVerifier_APrunedChainVerifiesFromItsSeal(t *testing.T) {
	t.Parallel()

	s := newScenario()
	ledger := newFakeLedger()
	projector := testProjector(ledger, s.source, testKeyring(true))
	for range 10 {
		_, err := projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
	}
	late := *s.source.usage[0]
	late.ID = pulid.MustNew("aiu_")
	late.CreatedAt = testNow - 50
	s.source.usage = append(s.source.usage, &late)
	_, err := projector.RunOnce(t.Context(), nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(ledger.seals[s.tenant]), 2)

	cut := ledger.seals[s.tenant][0]
	deleted, err := ledger.Prune(t.Context(), pruneThrough(s, cut.ToSeq))
	require.NoError(t, err)
	require.Positive(t, deleted)

	result, err := testVerifier(
		ledger,
		testKeyring(true),
		nil,
	).VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)
	assert.Equal(t, aiaudit.VerificationVerified, result.Status)
}

func TestRetention_PrunesOnlyAtSealsPastTheRetentionPeriod(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)
	before := len(ledger.all(s.tenant))

	retention := NewRetention(ledger, emptyRetention{}, nil, zap.NewNop())
	retention.now = func() time.Time { return time.Unix(testNow, 0) }
	result, err := retention.PruneAll(t.Context(), nil)
	require.NoError(t, err)
	assert.Zero(t, result.Deleted, "rows inside seven years are kept")

	retention.now = func() time.Time { return time.Unix(testNow, 0).AddDate(8, 0, 0) }
	result, err = retention.PruneAll(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, before, result.Deleted)
	assert.NotEmpty(t, ledger.seals[s.tenant], "seals outlive the rows they cover")

	verified, err := testVerifier(
		ledger,
		testKeyring(true),
		nil,
	).VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)
	assert.Equal(t, aiaudit.VerificationVerified, verified.Status)
}

func TestCorrelate_MatchesByRecordPrincipalAndTime(t *testing.T) {
	t.Parallel()

	s, ledger, events := projected(t, true)
	executed := events["proposal:"+s.proposal.String()+":executed"]
	require.NotNil(t, executed)

	ledger.entries = []*audit.Entry{
		{
			ID: "ae_match", ResourceID: executed.EntityID, PrincipalID: s.approver,
			Timestamp: executed.WindowEnd,
		},
		{
			ID: "ae_other_person", ResourceID: executed.EntityID, PrincipalID: s.user,
			Timestamp: executed.WindowEnd,
		},
		{
			ID: "ae_too_late", ResourceID: executed.EntityID, PrincipalID: s.approver,
			Timestamp: executed.WindowEnd + 10,
		},
	}

	linked, err := Correlate(t.Context(), ledger, s.tenant, []*aiaudit.AIAuditEvent{executed})
	require.NoError(t, err)
	require.Len(t, linked[executed.ID], 1)
	assert.Equal(t, "ae_match", linked[executed.ID][0].ID.String())
}
