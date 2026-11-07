//go:build integration

package auditintegrity

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditsealrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type integrityFixture struct {
	ctx      context.Context
	db       *keel.DB
	entries  auditports.AuditRepository
	seals    auditports.SealRepository
	sealer   *Sealer
	verifier *Verifier
	tenant   auditports.TenantKey
	userID   pulid.ID
	now      time.Time
}

func newIntegrityFixture(t *testing.T, keyring *Keyring) *integrityFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.WrapDB(db)
	entries := auditrepository.New(auditrepository.Params{DB: conn, Logger: zap.NewNop()})
	seals := auditsealrepository.New(auditsealrepository.Params{DB: conn, Logger: zap.NewNop()})
	now := time.Now().UTC()
	clock := func() time.Time { return now }
	registry := &metrics.Registry{}

	return &integrityFixture{
		ctx:     data.Ctx(ctx),
		db:      db,
		entries: entries,
		seals:   seals,
		sealer: NewSealer(&SealerParams{
			Entries: entries, Seals: seals, Keyring: keyring, Metrics: registry, Now: clock,
			Logger: zap.NewNop(),
		}),
		verifier: NewVerifier(&VerifierParams{
			Seals: seals, Keyring: keyring, Metrics: registry, Now: clock, Logger: zap.NewNop(),
		}),
		tenant: auditports.TenantKey{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
		},
		userID: data.User.ID,
		now:    now,
	}
}

func signedKeyring() *Keyring {
	return NewKeyring(&aiauditconfig.Config{Chain: aiauditconfig.ChainConfig{
		Keys:        []aiauditconfig.ChainKey{{ID: "k1", Secret: strings.Repeat("s", 32)}},
		ActiveKeyID: "k1",
	}})
}

func (f *integrityFixture) write(t *testing.T, hoursAgo int, class audit.RetentionClass) *audit.Entry {
	t.Helper()

	operation := permission.OpUpdate
	if class == audit.RetentionAccess {
		operation = permission.OpRead
	}
	e := &audit.Entry{
		ID:             pulid.MustNew("ae_"),
		OrganizationID: f.tenant.OrganizationID,
		BusinessUnitID: f.tenant.BusinessUnitID,
		Timestamp:      f.now.Add(-time.Duration(hoursAgo) * time.Hour).Unix(),
		Resource:       permission.ResourceShipment,
		ResourceID:     pulid.MustNew("shp_").String(),
		Operation:      operation,
		Category:       audit.CategoryUser,
		PrincipalType:  "session_user",
		PrincipalID:    f.userID,
		UserID:         f.userID,
		Comment:        "original",
	}
	e.Normalize()
	require.NoError(t, f.entries.Insert(f.ctx, []*audit.Entry{e}))
	return e
}

func (f *integrityFixture) head(t *testing.T) *audit.ChainHead {
	t.Helper()

	heads, err := f.seals.ListHeads(f.ctx)
	require.NoError(t, err)
	for _, head := range heads {
		if head.OrganizationID == f.tenant.OrganizationID {
			return head
		}
	}
	t.Fatal("the tenant has no chain head")
	return nil
}

func TestSealAndVerify_AnUntouchedTrailVerifies(t *testing.T) {
	f := newIntegrityFixture(t, signedKeyring())
	f.write(t, 30, audit.RetentionOperational)
	f.write(t, 5, audit.RetentionAccess)

	result, err := f.sealer.SealAll(f.ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Tenants)
	assert.GreaterOrEqual(t, result.Seals, 1)

	again, err := f.sealer.SealAll(f.ctx, nil)
	require.NoError(t, err)
	assert.Zero(t, again.Seals, "an hour is sealed once")

	verified, err := f.verifier.VerifyTenant(f.ctx, f.head(t))
	require.NoError(t, err)
	assert.Equal(t, audit.VerificationVerified, verified.LastVerificationStatus, verified.LastVerificationDetail)
}

func TestSealAndVerify_FindsAChangedEntry(t *testing.T) {
	f := newIntegrityFixture(t, signedKeyring())
	tampered := f.write(t, 10, audit.RetentionOperational)
	_, err := f.sealer.SealAll(f.ctx, nil)
	require.NoError(t, err)

	_, err = keel.Exec(f.ctx, f.db, "ALTER TABLE audit_entries DISABLE TRIGGER enforce_audit_append_only")
	require.NoError(t, err)
	_, err = keel.Exec(f.ctx, f.db, "UPDATE audit_entries SET comment = 'rewritten' WHERE id = $1", tampered.ID.String())
	require.NoError(t, err)
	_, err = keel.Exec(f.ctx, f.db, "ALTER TABLE audit_entries ENABLE TRIGGER enforce_audit_append_only")
	require.NoError(t, err)

	verified, err := f.verifier.VerifyTenant(f.ctx, f.head(t))
	require.NoError(t, err)
	assert.Equal(t, audit.VerificationBroken, verified.LastVerificationStatus)
	assert.Contains(t, verified.LastVerificationDetail, "changed")
}

func TestSealAndVerify_FindsAnEntryAddedToASealedHour(t *testing.T) {
	f := newIntegrityFixture(t, signedKeyring())
	f.write(t, 10, audit.RetentionOperational)
	_, err := f.sealer.SealAll(f.ctx, nil)
	require.NoError(t, err)

	f.write(t, 10, audit.RetentionOperational)

	verified, err := f.verifier.VerifyTenant(f.ctx, f.head(t))
	require.NoError(t, err)
	assert.Equal(t, audit.VerificationBroken, verified.LastVerificationStatus)
	assert.Contains(t, verified.LastVerificationDetail, "added")
}

func TestSealAndVerify_FindsAnAlteredSeal(t *testing.T) {
	f := newIntegrityFixture(t, signedKeyring())
	f.write(t, 10, audit.RetentionOperational)
	_, err := f.sealer.SealAll(f.ctx, nil)
	require.NoError(t, err)

	_, err = keel.Exec(f.ctx, f.db, "UPDATE audit_seals SET row_count = row_count + 1 WHERE organization_id = $1",
		f.tenant.OrganizationID.String())
	require.NoError(t, err)

	verified, err := f.verifier.VerifyTenant(f.ctx, f.head(t))
	require.NoError(t, err)
	assert.Equal(t, audit.VerificationBroken, verified.LastVerificationStatus)
	assert.Contains(t, verified.LastVerificationDetail, "altered")
}

func TestSealAndVerify_AMissingKeyIsReportedNotTreatedAsTampering(t *testing.T) {
	f := newIntegrityFixture(t, signedKeyring())
	f.write(t, 10, audit.RetentionOperational)
	_, err := f.sealer.SealAll(f.ctx, nil)
	require.NoError(t, err)

	f.verifier.keyring = NewKeyring(nil)
	verified, err := f.verifier.VerifyTenant(f.ctx, f.head(t))
	require.NoError(t, err)
	assert.Equal(t, audit.VerificationKeyMissing, verified.LastVerificationStatus)
}
