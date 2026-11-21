//go:build integration

package cloud_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantcopyrepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantpurgerepository"
	"github.com/emoss08/trenova/internal/cloud/app/archiveengine"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	genuineCommand = "1a1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	planted        = "2a1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	alteredCommand = "3a1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	forgedCommand  = "4a1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	futureCommand  = "5a1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	forgerKey      = "forger-install-test-key-not-a-real-secret"
	plantedComment = "a provenance-shaped row planted in the archive"
)

func holding(active string, also ...string) *aiauditconfig.Config {
	cfg := chainFor(active)
	for _, key := range also {
		cfg.Chain.Keys = append(cfg.Chain.Keys, chainFor(key).Chain.Keys...)
	}
	return cfg
}

type trailRig struct {
	src   *install
	facts *sourceFacts
	keys  []tenantarchive.Key
}

func (r *trailRig) take(t *testing.T, command string) string {
	t.Helper()
	return export(t, r.src, r.facts.tenant, tenantarchive.PurposeFinal, command, r.keys).SHA256
}

func (r *trailRig) load(
	t *testing.T,
	dst *install,
	command, sha string,
	restoring bool,
) (*archiveengine.ImportResult, error) {
	t.Helper()
	dst.buckets = r.src.buckets.clone()
	dst.engine = dst.engineWith(dst.buckets)
	req := &archiveengine.ImportRequest{
		InstanceID:   archiveInstance,
		ExpectSHA256: sha,
		Access:       access(tenantarchive.CommandPrefix(archiveInstance, command), r.keys),
	}
	if restoring {
		req.Tenant = &r.facts.tenant
		req.Restoring = true
	}
	return dst.engine.Import(dst.ctx, req)
}

func (r *trailRig) clear(t *testing.T, dst *install) {
	t.Helper()
	purgeOrganization(t, dst, r.facts.tenant)
	_, err := tenantpurgerepository.New(
		tenantpurgerepository.Params{DB: dst.conn, Logger: zap.NewNop()},
	).DeleteBusinessUnit(dst.ctx, r.facts.tenant)
	require.NoError(t, err)
}

func (r *trailRig) write(t *testing.T, at time.Time, action, comment string) {
	t.Helper()
	sys := dbscope.WithSystem(r.src.ctx, "write a tenant's audit trail for the archive test")
	require.NoError(t, auditdb.Entries.Create(sys, r.src.admin, &audit.Entry{
		OrganizationID: r.facts.tenant.OrganizationID,
		BusinessUnitID: r.facts.tenant.BusinessUnitID,
		Timestamp:      at.Unix(),
		Resource:       permission.ResourceOrganization,
		ResourceID:     r.facts.tenant.OrganizationID.String(),
		Operation:      permission.OpUpdate,
		Action:         action,
		Category:       audit.CategorySecurity,
		PrincipalType:  "system",
		PrincipalID:    "system",
		ActorName:      "Trenova",
		Comment:        comment,
		Critical:       true,
		Metadata: map[string]any{"archive": map[string]any{
			"auditSeals": map[string]any{"sourceVerified": true, "outcome": "kept"},
		}},
	}))
}

func TestAnImportedTrailSaysWhatThisServerCouldVerifyAboutIt(t *testing.T) {
	src := newInstall(t, true, sourceKey)
	r := &trailRig{src: src, facts: prepareSource(t, src), keys: archiveKeys(0x21)}
	org := r.facts.tenant.OrganizationID.String()

	genuine := r.take(t, genuineCommand)

	time.Sleep(2 * time.Second)
	r.write(t, time.Now(), tenantcopyrepository.ActionImported, plantedComment)
	sealThroughNow(t, src, src.chain, r.facts.tenant)
	withPlanted := r.take(t, planted)

	src.exec(t, `DO $$ BEGIN SET LOCAL session_replication_role = replica;
UPDATE audit_entries SET comment = 'rewritten after it was sealed'
WHERE organization_id = '`+org+`' AND comment = 'archived change 0'; END $$`)
	altered := r.take(t, alteredCommand)

	src.exec(t, `DO $$ BEGIN SET LOCAL session_replication_role = replica;
UPDATE audit_entries SET comment = 'rewritten, and the seals made again'
WHERE organization_id = '`+org+`' AND comment = 'archived change 2';
DELETE FROM audit_seals WHERE organization_id = '`+org+`';
DELETE FROM audit_chain_heads WHERE organization_id = '`+org+`'; END $$`)
	sealThroughNow(t, src, chainFor(forgerKey), r.facts.tenant)
	forged := r.take(t, forgedCommand)

	r.write(t, time.Now().Add(2*time.Hour), "organization.updated", "dated after the load")
	future := r.take(t, futureCommand)

	held := newInstallWithChain(t, false, destinationKey, holding(destinationKey, sourceKey))
	stranger := newInstall(t, false, destinationKey)

	t.Run("a genuine archive whose key this server holds verifies and is marked verified",
		func(t *testing.T) {
			result, err := r.load(t, held, genuineCommand, genuine, false)
			require.NoError(t, err)
			assert.Equal(t, archiveports.SourceVerified, result.Trails.AuditSeals.Source)
			assert.Equal(t, archiveports.SourceVerified, result.Trails.AIAudit.Source)
			assert.Zero(t, result.Trails.AuditSeals.Marked)
			assert.Zero(t, result.Trails.AIAudit.Marked)
			assert.Nil(t, result.Trails.Tail)
			status := verifyTrails(t, held, r.facts.tenant)
			assert.Equal(t, audit.VerificationVerified, status.audit)
			assert.Equal(t, aiaudit.VerificationVerified, status.ai)
			assert.Equal(t, []string{"true"}, held.query(t,
				`SELECT metadata->'archive'->'auditSeals'->>'sourceVerified' FROM audit_entries
                 WHERE organization_id = $1 AND action = $2 AND "timestamp" >= (
                     SELECT max(to_ts) FROM audit_seals WHERE organization_id = $1)`,
				org, tenantcopyrepository.ActionImported))
			r.clear(t, held)
		})

	t.Run("rows altered after they were sealed fail verification", func(t *testing.T) {
		result, err := r.load(t, held, alteredCommand, altered, false)
		require.NoError(t, err)
		assert.Equal(t, archiveports.SourceVerified, result.Trails.AuditSeals.Source,
			"the seals themselves verify")
		status := verifyTrails(t, held, r.facts.tenant)
		assert.Equal(t, audit.VerificationBroken, status.audit,
			"the entries no longer match what was sealed")
		r.clear(t, held)
	})

	t.Run("seals a forger made again under an unknown key are never reported as verified",
		func(t *testing.T) {
			result, err := r.load(t, held, forgedCommand, forged, false)
			require.NoError(t, err)
			assert.Equal(t, archiveports.SourceNotVerifiable, result.Trails.AuditSeals.Source)
			assert.True(t, result.Trails.AuditSeals.KeyMissing)
			assert.Equal(t, archiveports.AttestationNotVerified, result.Trails.AuditSeals.Refused,
				"the source could not verify seals made under a key it does not hold, and said so")
			assert.Equal(t, result.Trails.AuditSeals.Links, result.Trails.AuditSeals.Marked)
			assert.Equal(t, []string{"0"}, held.query(t,
				`SELECT count(*)::text FROM audit_seals WHERE organization_id = $1
                 AND hash_version NOT IN (3, 4)`, org),
				"every seal of the imported history is marked as not verified at its source")
			status := verifyTrails(t, held, r.facts.tenant)
			assert.Equal(t, audit.VerificationImported, status.audit)
			held.exec(t, `DO $$ BEGIN SET LOCAL session_replication_role = replica;
UPDATE audit_seals SET hash_version = 1 WHERE organization_id = '`+org+`' AND hash_version = 3; END $$`)
			stripped := verifyTrails(t, held, r.facts.tenant)
			assert.Equal(t, audit.VerificationBroken, stripped.audit,
				"the mark cannot be removed without breaking the chain")
			r.clear(t, held)
		})

	t.Run("a provenance row the archive carried never reads as this server's statement",
		func(t *testing.T) {
			result, err := r.load(t, stranger, planted, withPlanted, false)
			require.NoError(t, err)
			assert.Equal(t, archiveports.SourceNotVerifiable, result.Trails.AuditSeals.Source)
			covered := stranger.query(t,
				`SELECT s.hash_version::text FROM audit_entries e
                 JOIN audit_seals s ON s.organization_id = e.organization_id
                     AND s.business_unit_id = e.business_unit_id
                     AND e."timestamp" >= s.from_ts AND e."timestamp" < s.to_ts
                 WHERE e.organization_id = $1 AND e.comment = $2`,
				org, plantedComment)
			assert.Equal(t, []string{"5"}, covered,
				"the planted row lies inside the imported history, sealed as its archive states")
			own := stranger.query(t,
				`SELECT metadata->'archive'->>'manifestSha256' FROM audit_entries e
                 WHERE e.organization_id = $1 AND e.action = $2 AND e.comment <> $3
                 AND NOT EXISTS (SELECT 1 FROM audit_seals s WHERE s.organization_id = e.organization_id
                     AND e."timestamp" >= s.from_ts AND e."timestamp" < s.to_ts)`,
				org, tenantcopyrepository.ActionImported, plantedComment)
			assert.Equal(t, []string{withPlanted}, own,
				"only the row this load wrote lies after the imported history")
			status := verifyTrails(t, stranger, r.facts.tenant)
			assert.Equal(t, audit.VerificationSourceVerified, status.audit)
			assert.Equal(t, aiaudit.VerificationSourceVerified, status.ai)
			r.clear(t, stranger)
		})

	t.Run("a restore onto a server without the trail's key reports it and marks the trail",
		func(t *testing.T) {
			result, err := r.load(t, stranger, genuineCommand, genuine, true)
			require.NoError(t, err)
			assert.Equal(t, archiveports.ChainResealed, result.Trails.AuditSeals.Outcome)
			assert.True(t, result.Trails.AuditSeals.KeyMissing)
			assert.Positive(t, result.Trails.AuditSeals.Attested)
			assert.Zero(t, result.Trails.AuditSeals.Marked)
			note := result.Trails.RestoreNote()
			assert.Contains(t, note, "marked as verified at their source, as the archive states")
			assert.LessOrEqual(t, len(note), 500)
			assert.NotContains(t, note, "\n")
			status := verifyTrails(t, stranger, r.facts.tenant)
			assert.Equal(t, audit.VerificationSourceVerified, status.audit)
			assert.Equal(t, aiaudit.VerificationSourceVerified, status.ai)
			r.clear(t, stranger)
		})

	t.Run("an archive dated after the load is refused", func(t *testing.T) {
		_, err := r.load(t, stranger, futureCommand, future, false)
		assert.Equal(t, tenantarchive.RefusalManifest, refusalOf(t, err))
		assert.Empty(t, stranger.holding(t, r.facts.tenant), "nothing was written")
	})
}
