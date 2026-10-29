package aiauditprojector

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditcorrelate"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditkeyring"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditretention"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiaudittest"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditverify"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testVerifier(
	ledger *aiaudittest.FakeLedger,
	keyring *aiauditkeyring.Keyring,
	notifier *aiaudittest.FakeNotifier,
) *aiauditverify.Verifier {
	verifier, _ := auditedVerifier(ledger, keyring, notifier)

	return verifier
}

func auditedVerifier(
	ledger *aiaudittest.FakeLedger,
	keyring *aiauditkeyring.Keyring,
	notifier *aiaudittest.FakeNotifier,
) (*aiauditverify.Verifier, *aiaudittest.FakeAudit) {
	audited := &aiaudittest.FakeAudit{}
	params := aiauditverify.Params{
		Ledger:  ledger,
		Keyring: keyring,
		Audit:   audited,
		Now:     func() time.Time { return time.Unix(aiaudittest.FixedNow, 0) },
		Logger:  zap.NewNop(),
	}
	if notifier != nil {
		params.Notifier = notifier
	}

	return aiauditverify.New(&params), audited
}

func TestVerifier_AnUntouchedChainVerifies(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)
	verifier := testVerifier(ledger, aiaudittest.TestKeyring(true), &aiaudittest.FakeNotifier{})

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
	rows := ledger.All(s.tenant)
	require.Greater(t, len(rows), 5)
	tampered := rows[4]
	tampered.Outcome = aiaudit.OutcomeFailed

	notifier := &aiaudittest.FakeNotifier{}
	verifier, auditor := auditedVerifier(ledger, aiaudittest.TestKeyring(true), notifier)
	result, err := verifier.VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)

	assert.Equal(t, aiaudit.VerificationMismatch, result.Status)
	require.NotNil(t, result.FailedSeq)
	assert.Equal(t, tampered.Seq, *result.FailedSeq)
	assert.Equal(t, tampered.Seq-1, result.VerifiedSeq)

	require.Len(t, notifier.Permits, 1)
	assert.Equal(t, permission.ResourceAIAuditTrail, notifier.Permits[0].Resource)
	assert.Equal(t, notification.PriorityCritical, notifier.Permits[0].Notification.Priority)
	audited := auditor.Entries
	require.Len(t, audited, 1)
	assert.True(t, audited[0].Critical)
}

func TestVerifier_ASignedChainRewrittenAsUnsignedFails(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)
	rows := ledger.All(s.tenant)
	require.Greater(t, len(rows), 5)

	prev := rows[3].Hash
	for _, row := range rows[4:] {
		row.Outcome = aiaudit.OutcomeFailed
		require.NoError(t, aiaudit.Seal(row, prev, nil))
		prev = row.Hash
	}
	ledger.Mu.Lock()
	for _, seal := range ledger.Seals[s.tenant] {
		for _, row := range rows {
			if row.Seq == seal.ToSeq {
				seal.HeadHash = row.Hash
			}
		}
	}
	ledger.Heads[s.tenant].LastHash = prev
	ledger.Mu.Unlock()

	verifier := testVerifier(ledger, aiaudittest.TestKeyring(true), &aiaudittest.FakeNotifier{})
	result, err := verifier.VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)

	assert.Equal(t, aiaudit.VerificationMismatch, result.Status)
	require.NotNil(t, result.FailedSeq)
	assert.Equal(t, rows[4].Seq, *result.FailedSeq)
}

func TestVerifier_AnUnsignedHistoryFollowedBySignedRowsVerifies(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, false)
	rows := ledger.All(s.tenant)
	require.Greater(t, len(rows), 5)

	keyring := aiaudittest.TestKeyring(true)
	key, ok := keyring.Key(keyring.ActiveKeyID())
	require.True(t, ok)
	prev := rows[3].Hash
	for _, row := range rows[4:] {
		require.NoError(t, aiaudit.Seal(row, prev, key))
		prev = row.Hash
	}
	ledger.Mu.Lock()
	for _, seal := range ledger.Seals[s.tenant] {
		for _, row := range rows {
			if row.Seq == seal.ToSeq {
				seal.HeadHash = row.Hash
			}
		}
	}
	ledger.Heads[s.tenant].LastHash = prev
	ledger.Mu.Unlock()

	verifier := testVerifier(ledger, keyring, &aiaudittest.FakeNotifier{})
	result, err := verifier.VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)

	assert.Equal(t, aiaudit.VerificationVerified, result.Status)
}

func TestVerifier_ARemovedRowFailsTheChain(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)
	rows := ledger.Events[s.tenant]
	ledger.Events[s.tenant] = append(rows[:3:3], rows[4:]...)

	result, err := testVerifier(
		ledger,
		aiaudittest.TestKeyring(true),
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
		aiaudittest.TestKeyring(false),
		nil,
	).VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)

	assert.Equal(t, aiaudit.VerificationKeyMissing, result.Status)
	assert.Equal(t, int64(1), *result.FailedSeq)
}

func TestVerifier_APrunedChainVerifiesFromItsSeal(t *testing.T) {
	t.Parallel()

	s := newScenario()
	ledger := aiaudittest.NewFakeLedger()
	projector := testProjector(ledger, s.source, aiaudittest.TestKeyring(true))
	for range 10 {
		_, err := projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
	}
	late := *s.source.Usage[0]
	late.ID = pulid.MustNew("aiu_")
	late.CreatedAt = aiaudittest.FixedNow - 50
	s.source.Usage = append(s.source.Usage, &late)
	_, err := projector.RunOnce(t.Context(), nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(ledger.Seals[s.tenant]), 2)

	cut := ledger.Seals[s.tenant][0]
	deleted, err := ledger.Prune(t.Context(), pruneThrough(s, cut.ToSeq))
	require.NoError(t, err)
	require.Positive(t, deleted)

	result, err := testVerifier(
		ledger,
		aiaudittest.TestKeyring(true),
		nil,
	).VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)
	assert.Equal(t, aiaudit.VerificationVerified, result.Status)
}

func TestRetention_PrunesOnlyAtSealsPastTheRetentionPeriod(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)
	before := len(ledger.All(s.tenant))

	clock := time.Unix(aiaudittest.FixedNow, 0)
	retention := aiauditretention.New(&aiauditretention.Params{
		Ledger:    ledger,
		Retention: aiaudittest.EmptyRetention{},
		Now:       func() time.Time { return clock },
		Logger:    zap.NewNop(),
	})
	result, err := retention.PruneAll(t.Context(), nil)
	require.NoError(t, err)
	assert.Zero(t, result.Deleted, "rows inside seven years are kept")

	clock = time.Unix(aiaudittest.FixedNow, 0).AddDate(8, 0, 0)
	result, err = retention.PruneAll(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, before, result.Deleted)
	assert.NotEmpty(t, ledger.Seals[s.tenant], "seals outlive the rows they cover")

	verified, err := testVerifier(
		ledger,
		aiaudittest.TestKeyring(true),
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

	ledger.Entries = []*audit.Entry{
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

	linked, err := aiauditcorrelate.Correlate(
		t.Context(),
		ledger,
		s.tenant,
		[]*aiaudit.AIAuditEvent{executed},
	)
	require.NoError(t, err)
	require.Len(t, linked[executed.ID], 1)
	assert.Equal(t, "ae_match", linked[executed.ID][0].ID.String())
}
