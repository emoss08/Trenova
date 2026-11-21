//go:build integration

package cloud_test

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/app/archiveengine"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditintegrity"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/metricstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	statedCommand    = "6a1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	truncatedCommand = "6b1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	forgedStatement  = "6c1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	brokenStatement  = "6d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
)

func (r *trailRig) loadStated(
	t *testing.T,
	dst *install,
	command string,
	state func(*tenantarchive.Manifest),
) *archiveengine.ImportResult {
	t.Helper()
	dst.buckets = r.src.buckets.clone()
	dst.engine = dst.engineWith(dst.buckets)
	prefix := tenantarchive.CommandPrefix(archiveInstance, command)
	key := prefix + tenantarchive.ManifestName
	m, err := tenantarchive.DecodeManifest(dst.buckets.objects[key])
	require.NoError(t, err)
	if state != nil {
		state(m)
	}
	body, err := m.Encode()
	require.NoError(t, err)
	dst.buckets.objects[key] = body
	result, err := dst.engine.Import(dst.ctx, &archiveengine.ImportRequest{
		InstanceID:   archiveInstance,
		ExpectSHA256: tenantarchive.ManifestSHA256(body),
		Access:       access(prefix, r.keys),
	})
	require.NoError(t, err)
	return result
}

func (r *trailRig) liveSeals(t *testing.T) string {
	t.Helper()
	return r.src.query(t,
		`SELECT count(*)::text || ' ' || COALESCE(max(seq), 0)::text FROM audit_seals
         WHERE organization_id = $1`, r.facts.tenant.OrganizationID.String())[0]
}

func TestAnArchiveStatesWhatItsSourceVerifiedAndAnImportReliesOnItOnlyWhenWhatArrivedAgrees(
	t *testing.T,
) {
	src := newInstall(t, true, sourceKey)
	r := &trailRig{src: src, facts: prepareSource(t, src), keys: archiveKeys(0x31)}
	org := r.facts.tenant.OrganizationID.String()

	time.Sleep(2 * time.Second)
	after := time.Now()
	r.write(t, after, "organization.updated", "written after the last hourly seal")
	before := r.liveSeals(t)
	headBefore := src.query(t, `SELECT last_seq::text || ' ' || last_hash || ' ' || sealed_through::text
        FROM audit_chain_heads WHERE organization_id = $1`, org)

	hourly := auditintegrity.NewSealer(&auditintegrity.SealerParams{
		Seals:   src.auditSeals(),
		Keyring: auditintegrity.NewKeyring(src.chain),
		Metrics: metricstest.Disabled(),
		Logger:  zap.NewNop(),
	})
	var (
		wg        sync.WaitGroup
		sealed    int
		sealedErr error
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		now := time.Now()
		sealed, sealedErr = hourly.SealTenant(src.ctx, auditports.TenantKey{
			OrganizationID: r.facts.tenant.OrganizationID,
			BusinessUnitID: r.facts.tenant.BusinessUnitID,
		}, auditintegrity.Cutoff(now)-int64((72*time.Hour).Seconds()), now.Unix()+1)
	}()
	stated := export(t, src, r.facts.tenant, tenantarchive.PurposeFinal, statedCommand, r.keys)
	wg.Wait()
	require.NoError(t, sealedErr)
	require.Positive(t, sealed)

	t.Run("the source seals what its last seal did not cover inside the archive, not in its live chain",
		func(t *testing.T) {
			require.NotNil(t, stated.Manifest.Trails)
			trail := stated.Manifest.Trails.Audit
			require.NotNil(t, trail)
			assert.True(t, trail.Verified)
			assert.Greater(t, trail.SealedThrough.Unix(), after.Unix(),
				"the archive is sealed past the entry written after the last hourly seal")
			ai := stated.Manifest.Trails.AIAudit
			require.NotNil(t, ai)
			assert.True(t, ai.Verified)
			assert.Equal(t, src.query(t,
				`SELECT count(*)::text FROM ai_audit_events WHERE organization_id = $1`, org)[0],
				strconv.FormatInt(ai.Links, 10))
			assert.NotEqual(t, before, r.liveSeals(t),
				"the hourly sealer ran beside the archive and sealed the live chain itself")
			status := verifyTrails(t, src, r.facts.tenant)
			assert.Equal(t, audit.VerificationVerified, status.audit,
				"taking the archive never wrote a seal into the live chain, so nothing is sealed twice")
			assert.NotEqual(t, headBefore, src.query(t,
				`SELECT last_seq::text || ' ' || last_hash || ' ' || sealed_through::text
                 FROM audit_chain_heads WHERE organization_id = $1`, org),
				"only the hourly sealer moved the live head")
		})

	stranger := newInstall(t, false, destinationKey)

	t.Run("a genuine move is verified at its source, as its archive states, on both trails",
		func(t *testing.T) {
			result := r.loadStated(t, stranger, statedCommand, nil)
			assert.Nil(t, result.Trails.Tail, "no entry arrived unsealed")
			assert.Equal(t, archiveports.EstablishedAtSource, result.Trails.AuditSeals.Established)
			assert.Equal(t, archiveports.EstablishedAtSource, result.Trails.AIAudit.Established)
			assert.Equal(t, stated.Manifest.Trails.Audit.Links, result.Trails.AuditSeals.Attested)
			status := verifyTrails(t, stranger, r.facts.tenant)
			assert.Equal(t, audit.VerificationSourceVerified, status.audit)
			assert.Equal(t, aiaudit.VerificationSourceVerified, status.ai)
			assert.Equal(t, []string{"verified_at_source " + stated.SHA256}, stranger.query(t,
				`SELECT (metadata->'archive'->'auditSeals'->>'established') || ' ' ||
                     (metadata->'archive'->'auditSeals'->'attestation'->>'manifestSha256')
                 FROM audit_entries WHERE organization_id = $1 AND action = 'organization.imported'`,
				org), "the provenance entry names what it relied on")
			r.clear(t, stranger)
		})

	mismatches := []struct {
		name    string
		state   func(*tenantarchive.Manifest)
		audit   archiveports.AttestationRefusal
		aiAudit archiveports.AttestationRefusal
	}{
		{
			name:  "a count of links that differs from what arrived",
			state: func(m *tenantarchive.Manifest) { m.Trails.Audit.Links++ },
			audit: archiveports.AttestationLinks,
		},
		{
			name: "a sealed-through moment before the last seal's end",
			state: func(m *tenantarchive.Manifest) {
				m.Trails.Audit.SealedThrough = m.Trails.Audit.SealedThrough.Add(-time.Second)
			},
			audit: archiveports.AttestationSealedThrough,
		},
		{
			name:  "a statement that the source could not verify the chain",
			state: func(m *tenantarchive.Manifest) { m.Trails.Audit.Verified = false },
			audit: archiveports.AttestationNotVerified,
		},
		{
			name: "AI rows recorded after the sealed-through moment",
			state: func(m *tenantarchive.Manifest) {
				m.Trails.AIAudit.SealedThrough = time.Unix(1, 0).UTC()
			},
			aiAudit: archiveports.AttestationDatedAfter,
		},
		{
			name:    "an AI chain with a count of links that differs",
			state:   func(m *tenantarchive.Manifest) { m.Trails.AIAudit.Links-- },
			aiAudit: archiveports.AttestationLinks,
		},
		{
			name:    "no statement at all",
			state:   func(m *tenantarchive.Manifest) { m.Trails = nil },
			audit:   archiveports.AttestationAbsent,
			aiAudit: archiveports.AttestationAbsent,
		},
	}
	for _, tc := range mismatches {
		t.Run("falls back to not verified on "+tc.name, func(t *testing.T) {
			result := r.loadStated(t, stranger, statedCommand, tc.state)
			status := verifyTrails(t, stranger, r.facts.tenant)
			if tc.audit != "" {
				assert.Equal(t, tc.audit, result.Trails.AuditSeals.Refused)
				assert.Equal(t, archiveports.EstablishedNotVerified, result.Trails.AuditSeals.Established)
				assert.Zero(t, result.Trails.AuditSeals.Attested)
				assert.Equal(t, audit.VerificationImported, status.audit)
			} else {
				assert.Equal(t, audit.VerificationSourceVerified, status.audit)
			}
			if tc.aiAudit != "" {
				assert.Equal(t, tc.aiAudit, result.Trails.AIAudit.Refused)
				assert.Zero(t, result.Trails.AIAudit.Attested)
				assert.Equal(t, aiaudit.VerificationImported, status.ai)
			} else {
				assert.Equal(t, aiaudit.VerificationSourceVerified, status.ai)
			}
			r.clear(t, stranger)
		})
	}

	src.exec(t, `DO $$ BEGIN SET LOCAL session_replication_role = replica;
DELETE FROM ai_audit_seals WHERE organization_id = '`+org+`'
    AND to_seq = (SELECT max(seq) FROM ai_audit_events WHERE organization_id = '`+org+`');
DELETE FROM ai_audit_events WHERE organization_id = '`+org+`'
    AND seq = (SELECT max(seq) FROM ai_audit_events WHERE organization_id = '`+org+`'); END $$`)
	truncated := export(t, src, r.facts.tenant, tenantarchive.PurposeFinal, truncatedCommand, r.keys)

	t.Run("an AI chain whose head names a row the archive does not carry falls back", func(t *testing.T) {
		require.True(t, truncated.Manifest.Trails.AIAudit.Verified,
			"every row the source still holds verifies")
		result := r.loadStated(t, stranger, truncatedCommand, nil)
		assert.Equal(t, archiveports.AttestationUnsealedTail, result.Trails.AIAudit.Refused)
		assert.Equal(t, aiaudit.VerificationMismatch, verifyTrails(t, stranger, r.facts.tenant).ai,
			"rows are missing from the end of the chain, which its own verification reports")
		r.clear(t, stranger)
	})

	src.exec(t, `DO $$ BEGIN SET LOCAL session_replication_role = replica;
DELETE FROM audit_seals WHERE organization_id = '`+org+`';
DELETE FROM audit_chain_heads WHERE organization_id = '`+org+`'; END $$`)
	sealThroughNow(t, src, chainFor(forgerKey), r.facts.tenant)
	forged := export(t, src, r.facts.tenant, tenantarchive.PurposeFinal, forgedStatement, r.keys)

	t.Run("a forged statement is believed only as the archive's, and never as verified here",
		func(t *testing.T) {
			require.False(t, forged.Manifest.Trails.Audit.Verified,
				"the source cannot verify seals made under a key it does not hold")
			result := r.loadStated(t, stranger, forgedStatement, func(m *tenantarchive.Manifest) {
				m.Trails.Audit.Verified = true
			})
			assert.Equal(t, archiveports.EstablishedAtSource, result.Trails.AuditSeals.Established)
			status := verifyTrails(t, stranger, r.facts.tenant)
			assert.Equal(t, audit.VerificationSourceVerified, status.audit)
			assert.NotEqual(t, audit.VerificationVerified, status.audit)
			r.clear(t, stranger)
		})

	src.exec(t, `DO $$ BEGIN SET LOCAL session_replication_role = replica;
UPDATE audit_seals SET prev_hash = repeat('0', 64) WHERE organization_id = '`+org+`'
    AND seq = (SELECT max(seq) FROM audit_seals WHERE organization_id = '`+org+`'); END $$`)
	broken := export(t, src, r.facts.tenant, tenantarchive.PurposeFinal, brokenStatement, r.keys)

	t.Run("a statement over a chain that breaks inside the archive falls back", func(t *testing.T) {
		result := r.loadStated(t, stranger, brokenStatement, func(m *tenantarchive.Manifest) {
			m.Trails.Audit.Verified = true
		})
		require.NotNil(t, broken.Manifest.Trails.Audit)
		assert.Equal(t, archiveports.AttestationBroken, result.Trails.AuditSeals.Refused)
		assert.Equal(t, archiveports.EstablishedNotVerified, result.Trails.AuditSeals.Established)
		assert.Equal(t, audit.VerificationImported, verifyTrails(t, stranger, r.facts.tenant).audit)
		r.clear(t, stranger)
	})
}
