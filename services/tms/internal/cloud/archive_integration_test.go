//go:build integration

package cloud_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/bootstrap/envelopes"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantcopyrepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantpurgerepository"
	"github.com/emoss08/trenova/internal/cloud/app/archiveengine"
	"github.com/emoss08/trenova/internal/cloud/archivecrypt"
	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/archivestream"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/aiauditrepository"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditkeyring"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiaudittest"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditverify"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditsealrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditintegrity"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
	"github.com/emoss08/trenova/internal/testutil/metricstest"
	"github.com/emoss08/trenova/internal/testutil/objectstoretest"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	archiveInstance = "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	archiveCommand  = "7d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	exportCommand   = "8d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	sourceKey       = "source-install-test-key-not-a-real-secret"
	destinationKey  = "destination-install-test-key-not-a-real-secret"
	envelopeMarker  = "trenova-envelope:v1:"
	totpSecret      = "JBSWY3DPEHPK3PXPTESTONLY"
)

type memoryBucket struct {
	mu      sync.Mutex
	objects map[string][]byte
	discard bool
}

func newBucket() *memoryBucket {
	return &memoryBucket{objects: map[string][]byte{}}
}

func (b *memoryBucket) Open(
	*tenantarchive.Location,
	*tenantarchive.Credentials,
) (archiveports.Bucket, error) {
	return b, nil
}

func (b *memoryBucket) Put(_ context.Context, key string, body io.Reader) error {
	if b.discard {
		h := sha256.New()
		n, err := io.Copy(h, body)
		b.mu.Lock()
		b.objects[key] = []byte(fmt.Sprintf("%d:%x", n, h.Sum(nil)))
		b.mu.Unlock()
		return err
	}
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.objects[key] = raw
	return nil
}

func (b *memoryBucket) Get(_ context.Context, key string) (io.ReadCloser, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	raw, ok := b.objects[key]
	if !ok {
		return nil, archiveports.ErrMissing
	}
	return io.NopCloser(bytes.NewReader(raw)), nil
}

func (b *memoryBucket) List(_ context.Context, prefix string, fn func(string, int64) error) error {
	b.mu.Lock()
	keys := make([]string, 0)
	for k := range b.objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	b.mu.Unlock()
	slices.Sort(keys)
	for _, k := range keys {
		if err := fn(k, int64(len(b.objects[k]))); err != nil {
			return err
		}
	}
	return nil
}

func (b *memoryBucket) Children(context.Context, string) ([]string, error) {
	return []string{}, nil
}

func (b *memoryBucket) DeletePrefix(_ context.Context, prefix string) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var n int64
	for k := range b.objects {
		if strings.HasPrefix(k, prefix) {
			delete(b.objects, k)
			n++
		}
	}
	return n, nil
}

func (b *memoryBucket) clone() *memoryBucket {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := newBucket()
	for k, v := range b.objects {
		out.objects[k] = bytes.Clone(v)
	}
	return out
}

type install struct {
	ctx     context.Context
	admin   *keel.DB
	conn    *postgres.Connection
	rows    archiveports.Rows
	store   *objectstoretest.Store
	cipher  *encryptionservice.Service
	engine  *archiveengine.Engine
	buckets *memoryBucket
	chain   *aiauditconfig.Config
}

func chainFor(key string) *aiauditconfig.Config {
	id := strings.SplitN(key, "-", 2)[0] + "-chain"
	return &aiauditconfig.Config{Chain: aiauditconfig.ChainConfig{
		Keys:        []aiauditconfig.ChainKey{{ID: id, Secret: key + "-chain"}},
		ActiveKeyID: id,
	}}
}

func newInstall(t *testing.T, seeded bool, key string) *install {
	t.Helper()
	var (
		ctx     context.Context
		admin   *keel.DB
		dsn     string
		cleanup func()
	)
	if seeded {
		ctx, admin, dsn, cleanup = seededtest.SetupTestDBWithDSN(t)
	} else {
		ctx, admin, dsn, cleanup = seedtest.SetupTestDBWithDSN(t)
	}
	t.Cleanup(cleanup)
	_ = ctx
	ctx = t.Context()
	conn, dbCfg := enforcedConnection(t, admin, dsn, &statementCounter{})
	in := &install{
		ctx:     ctx,
		admin:   admin,
		conn:    conn,
		store:   objectstoretest.New(),
		cipher:  encryptionservice.NewWithKeyManager(encryptionservice.NewLocalKeyManager(key)),
		buckets: newBucket(),
		chain:   chainFor(key),
	}
	in.rows = tenantcopyrepository.New(tenantcopyrepository.Params{
		DB: conn, App: &appconfig.Config{Name: "archive-test"}, Database: dbCfg,
		AIAudit: in.chain,
	})
	in.engine = in.engineWith(in.buckets)
	return in
}

func (in *install) engineWith(buckets archiveports.Buckets) *archiveengine.Engine {
	return archiveengine.New(archiveengine.Params{
		Rows:      in.rows,
		Buckets:   buckets,
		Storage:   in.store,
		Secrets:   encryptionservice.NewSecretCipher(in.cipher),
		Blobs:     encryptionservice.NewBlobCipher(in.cipher),
		Envelopes: envelopes.Sources(true),
		App:       &appconfig.Config{Version: "2026.11.0-test"},
		Reporting: &reportingconfig.Config{},
		Clock:     func() int64 { return time.Now().Unix() },
		Logger:    zap.NewNop(),
	})
}

func (in *install) query(t *testing.T, sql string, args ...any) []string {
	t.Helper()
	rows, err := keel.QueryAll(in.ctx, in.admin, pgx.RowTo[string], sql, args...)
	require.NoError(t, err, sql)
	return rows
}

func (in *install) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	_, err := keel.Exec(in.ctx, in.admin, sql, args...)
	require.NoError(t, err, sql)
}

func envelopeTables() map[string][]string {
	out := map[string][]string{}
	for _, column := range envelopes.Sources(false).Columns {
		table, index, ok := keel.ColumnOf(column.Value)
		if ok {
			out[table.Name] = append(out[table.Name], table.Columns[index].Name)
		}
	}
	return out
}

func (in *install) plan(t *testing.T) *archiveplan.Plan {
	t.Helper()
	plan, err := in.rows.PlanFor(in.ctx, tenantarchive.PurposeFinal, envelopeTables())
	require.NoError(t, err)
	return plan
}

func (in *install) counts(t *testing.T, tenant dbscope.Tenant) map[string]int64 {
	t.Helper()
	type count struct {
		Table string `db:"table_name"`
		Rows  int64  `db:"row_count"`
	}
	rows, err := keel.QueryAll(
		in.ctx,
		in.admin,
		pgx.RowToStructByName[count],
		in.plan(t).CountQuery(tenant),
	)
	require.NoError(t, err)
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Table] = r.Rows
	}
	return out
}

func (in *install) holding(t *testing.T, tenant dbscope.Tenant) []string {
	t.Helper()
	held, err := in.rows.Holding(in.ctx, in.plan(t), tenant)
	require.NoError(t, err)
	return held
}

func (in *install) tenantNamed(t *testing.T, name string) dbscope.Tenant {
	t.Helper()
	row := in.query(
		t,
		"SELECT id || ' ' || business_unit_id FROM organizations WHERE name = $1",
		name,
	)
	require.Len(t, row, 1, name)
	parts := strings.Fields(row[0])
	return dbscope.Tenant{OrganizationID: pulid.ID(parts[0]), BusinessUnitID: pulid.ID(parts[1])}
}

func purgeOrganization(t *testing.T, in *install, tenant dbscope.Tenant) {
	t.Helper()
	purge := tenantpurgerepository.New(
		tenantpurgerepository.Params{DB: in.conn, Logger: zap.NewNop()},
	)
	for range 200 {
		result, err := purge.PurgeRows(
			in.ctx,
			&cloudtenantports.PurgeRowsRequest{Tenant: tenant, BatchSize: 2000},
		)
		require.NoError(t, err)
		if result.Complete || result.Deleted == 0 {
			break
		}
	}
	deleted, err := purge.DeleteOrganization(in.ctx, tenant)
	require.NoError(t, err)
	require.True(t, deleted.Deleted, "the second organization is gone: %s", deleted.BlockedBy)
}

func tenantContext(in *install, tenant dbscope.Tenant) context.Context {
	return dbscope.WithValidTenant(in.ctx, tenant)
}

func (in *install) auditSeals() auditports.SealRepository {
	return auditsealrepository.New(auditsealrepository.Params{DB: in.conn, Logger: zap.NewNop()})
}

func (in *install) aiLedger() aiauditports.AIAuditRepository {
	return aiauditrepository.New(aiauditrepository.Params{
		Entitlements: entitlementtest.Instance(), DB: in.conn, Logger: zap.NewNop(),
	})
}

func (in *install) aiKeyring() *aiauditkeyring.Keyring {
	return aiauditkeyring.New(&in.chain.Chain, zap.NewNop())
}

func writeTrails(t *testing.T, in *install, tenant dbscope.Tenant) {
	t.Helper()
	now := time.Now()
	sys := dbscope.WithSystem(in.ctx, "write a tenant's audit trail for the archive test")
	for i := range 3 {
		require.NoError(t, auditdb.Entries.Create(sys, in.admin, &audit.Entry{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
			Timestamp:      now.Add(-30*time.Hour + time.Duration(i)*time.Minute).Unix(),
			Resource:       permission.ResourceOrganization,
			ResourceID:     tenant.OrganizationID.String(),
			Operation:      permission.OpUpdate,
			Category:       audit.CategorySecurity,
			PrincipalType:  "system",
			PrincipalID:    "system",
			Comment:        fmt.Sprintf("archived change %d", i),
			Metadata:       map[string]any{"n": i, "big": "9007199254740993"},
			Critical:       true,
		}))
	}
	sealer := auditintegrity.NewSealer(&auditintegrity.SealerParams{
		Seals:   in.auditSeals(),
		Keyring: auditintegrity.NewKeyring(in.chain),
		Metrics: metricstest.Disabled(),
		Now:     func() time.Time { return now },
		Logger:  zap.NewNop(),
	})
	cutoff := auditintegrity.Cutoff(now)
	sealed, err := sealer.SealTenant(in.ctx, auditports.TenantKey{
		OrganizationID: tenant.OrganizationID, BusinessUnitID: tenant.BusinessUnitID,
	}, cutoff-int64((72*time.Hour).Seconds()), cutoff)
	require.NoError(t, err)
	require.Positive(t, sealed)

	info := pagination.TenantInfo{OrgID: tenant.OrganizationID, BuID: tenant.BusinessUnitID}
	ring := in.aiKeyring()
	for batch := range 2 {
		events := make([]*aiaudit.AIAuditEvent, 0, 3)
		for i := range 3 {
			events = append(events, &aiaudit.AIAuditEvent{
				SourceKey:     fmt.Sprintf("usage:archive:%d:%d", batch, i),
				OccurredAt:    now.Add(-time.Duration(10-batch*3-i) * time.Hour).Unix(),
				Kind:          aiaudit.KindModelCall,
				Outcome:       aiaudit.OutcomeSucceeded,
				PrincipalType: aiaudit.PrincipalUser,
				PrincipalID:   "usr_archive",
				Arguments:     map[string]any{"n": i, "big": 9007199254740993},
				Purpose:       aiaudit.PurposeLive,
			})
		}
		_, err = in.aiLedger().
			Append(tenantContext(in, tenant), &aiauditports.AppendAIAuditEventsRequest{
				TenantInfo: info,
				Events:     events,
				Sign:       ring.Signer(),
				KeyID:      ring.ActiveKeyID(),
				Now:        now.Unix(),
			})
		require.NoError(t, err)
	}
}

type trailStatus struct {
	audit audit.VerificationStatus
	ai    aiaudit.VerificationStatus
	links int64
}

func verifyTrails(t *testing.T, in *install, tenant dbscope.Tenant) trailStatus {
	t.Helper()
	info := pagination.TenantInfo{OrgID: tenant.OrganizationID, BuID: tenant.BusinessUnitID}
	ctx := tenantContext(in, tenant)
	seals := in.auditSeals()
	head, err := seals.GetHead(ctx, info)
	require.NoError(t, err)
	verifier := auditintegrity.NewVerifier(&auditintegrity.VerifierParams{
		Seals:   seals,
		Keyring: auditintegrity.NewKeyring(in.chain),
		Metrics: metricstest.Disabled(),
		Logger:  zap.NewNop(),
	})
	checked, err := verifier.VerifyTenant(ctx, head)
	require.NoError(t, err)
	ai, err := aiauditverify.New(&aiauditverify.Params{
		Ledger:   in.aiLedger(),
		Keyring:  in.aiKeyring(),
		Notifier: &aiaudittest.FakeNotifier{},
		Audit:    &aiaudittest.FakeAudit{},
		Logger:   zap.NewNop(),
	}).VerifyTenant(ctx, info, nil)
	require.NoError(t, err)
	return trailStatus{
		audit: checked.LastVerificationStatus,
		ai:    ai.Status,
		links: checked.LastSeq + ai.Rows,
	}
}

func withProvenance(counts map[string]int64, n int64) map[string]int64 {
	out := make(map[string]int64, len(counts))
	for k, v := range counts {
		out[k] = v
	}
	out["audit_entries"] += n
	return out
}

var resealedColumns = map[string][]string{
	"audit_seals": {"prev_hash", "hash", "hash_key_id", "hash_version", "source_prev_hash",
		"source_hash", "source_hash_key_id", "source_hash_version", "resealed_at"},
	"ai_audit_events": {"prev_hash", "hash", "hash_key_id", "hash_version", "source_prev_hash",
		"source_hash", "source_hash_key_id", "source_hash_version", "resealed_at"},
	"ai_audit_seals": {"head_hash", "hash_key_id", "source_head_hash", "source_hash_key_id"},
	"audit_chain_heads": {"last_hash", "hash_key_id", "last_verified_seq", "last_verified_at",
		"last_verification_status", "last_verification_failed_seq", "last_verification_detail",
		"verify_cursor_seq", "version", "updated_at"},
	"ai_audit_chain_heads": {"last_hash", "hash_key_id", "last_verified_seq", "last_verified_at",
		"last_verification_status", "last_verification_failed_seq", "last_verification_detail",
		"version", "updated_at"},
}

func withoutProvenance(rows []string) []string {
	return slices.DeleteFunc(rows, func(r string) bool {
		return strings.Contains(r, tenantcopyrepository.ActionImported) ||
			strings.Contains(r, tenantcopyrepository.ActionRestored)
	})
}

func archiveKeys(material byte) []tenantarchive.Key {
	key := bytes.Repeat([]byte{material}, tenantarchive.KeySize)
	return []tenantarchive.Key{{ID: "region-1", Material: key, Current: true}}
}

func access(prefix string, keys []tenantarchive.Key) *tenantarchive.Access {
	return &tenantarchive.Access{
		Location: tenantarchive.Location{
			Endpoint: "https://objects.test", Region: "us-east-1", Bucket: "archives", Prefix: prefix,
		},
		Credentials: tenantarchive.Credentials{AccessKey: "test-access", SecretKey: "test-secret"},
		Keys:        keys,
	}
}

type sourceFacts struct {
	tenant    dbscope.Tenant
	other     dbscope.Tenant
	user      string
	objects   map[string][]byte
	sealed    map[string]bool
	otherBlob []byte
}

func prepareSource(t *testing.T, src *install) *sourceFacts {
	t.Helper()
	facts := &sourceFacts{objects: map[string][]byte{}, sealed: map[string]bool{}}
	facts.tenant = src.tenantNamed(t, "Trenova Logistics")
	purgeOrganization(t, src, src.tenantNamed(t, "Trenova Transportation"))
	other := seedtest.SeedAdditionalTenant(t, src.ctx, src.admin, "ZQ")
	facts.other = dbscope.Tenant{
		OrganizationID: other.Organization.ID,
		BusinessUnitID: other.BusinessUnit.ID,
	}

	facts.user = src.query(t, "SELECT id FROM users WHERE business_unit_id = $1 ORDER BY created_at, id LIMIT 1",
		facts.tenant.BusinessUnitID.String())[0]
	sealed, err := encryptionservice.NewSecretCipher(src.cipher).
		Seal(src.ctx, totpSecret, mfaScope(facts))
	require.NoError(t, err)
	src.exec(t, `DELETE FROM mfa_authenticators WHERE user_id = $1`, facts.user)
	src.exec(
		t,
		`INSERT INTO mfa_authenticators (id, user_id, organization_id, type, name, secret_cipher, enabled, verified_at)
VALUES ($1, $2, $3, 'totp', 'Phone', $4, true, 1790000000)`,
		pulid.MustNew("mfa_").String(),
		facts.user,
		facts.tenant.OrganizationID.String(),
		sealed,
	)
	src.exec(
		t,
		`INSERT INTO mfa_recovery_codes (id, user_id, organization_id, code_hash, hash_version)
VALUES ($1, $2, $3, $4, 2)`,
		pulid.MustNew("mrc_").String(),
		facts.user,
		facts.tenant.OrganizationID.String(),
		strings.Repeat("e", 64),
	)

	tenantInfo := pagination.TenantInfo{
		OrgID: facts.tenant.OrganizationID,
		BuID:  facts.tenant.BusinessUnitID,
	}
	paths := src.query(
		t,
		`SELECT storage_path FROM documents WHERE organization_id = $1 AND storage_path IS NOT NULL ORDER BY id LIMIT 25`,
		facts.tenant.OrganizationID.String(),
	)
	for i, path := range paths {
		plain := make([]byte, 3000+len(path))
		_, _ = rand.Read(plain)
		blob, serr := src.cipher.SealBlob(
			src.ctx,
			bytes.NewReader(plain),
			int64(len(plain)),
			encryptionservice.AAD{
				Purpose: encryptionservice.PurposeDocument, OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID, ResourceID: path,
			},
		)
		require.NoError(t, serr)
		stored, rerr := io.ReadAll(blob)
		require.NoError(t, rerr)
		src.store.PutTyped(path, documentTypes[i%len(documentTypes)], stored)
		facts.objects[path] = plain
		facts.sealed[path] = true
	}
	plainKey := facts.tenant.OrganizationID.String() + "/exports/notes.txt"
	facts.objects[plainKey] = []byte("plain object of the tenant")
	src.store.PutTyped(plainKey, "text/plain; charset=utf-8", facts.objects[plainKey])
	writeTrails(t, src, facts.tenant)
	facts.otherBlob = []byte(
		"an object of the other tenant: " + facts.other.OrganizationID.String(),
	)
	src.store.Put(facts.other.OrganizationID.String()+"/documents/x.pdf", facts.otherBlob)
	return facts
}

var documentTypes = []string{"application/pdf", "image/png", ""}

func mfaScope(facts *sourceFacts) *kernelports.SecretScope {
	for _, column := range envelopes.Sources(false).Columns {
		if column.Name != "MFA authenticator secret" {
			continue
		}
		scopes := column.Scopes(&kernelports.EnvelopeRow{
			Tenant: pagination.TenantInfo{
				OrgID: facts.tenant.OrganizationID,
				BuID:  facts.tenant.BusinessUnitID,
			},
			Extra: map[string]string{"user_id": facts.user},
		}, "")
		return &scopes[0]
	}
	panic("the MFA authenticator secret is not in the envelope registry")
}

func export(
	t *testing.T,
	src *install,
	tenant dbscope.Tenant,
	purpose tenantarchive.Purpose,
	command string,
	keys []tenantarchive.Key,
) *archiveengine.ExportResult {
	t.Helper()
	result, err := src.engine.Export(src.ctx, &archiveengine.ExportRequest{
		Purpose:    purpose,
		InstanceID: archiveInstance,
		Tenant:     tenant,
		Access:     access(tenantarchive.CommandPrefix(archiveInstance, command), keys),
	})
	require.NoError(t, err)
	return result
}

func plainParts(
	t *testing.T,
	bucket *memoryBucket,
	m *tenantarchive.Manifest,
	prefix string,
	keys []tenantarchive.Key,
) map[string][]byte {
	t.Helper()
	var dataKey *archivecrypt.DataKey
	if m.Encryption != nil {
		var err error
		dataKey, err = archivecrypt.Unwrap(
			m.Encryption.WrappedDataKey,
			keys[0].Material,
			m.InstanceID,
		)
		require.NoError(t, err)
	}
	out := map[string][]byte{}
	read := func(key string, sum archivestream.Sum, compress bool) {
		r, err := archivestream.NewReader(bytes.NewReader(bucket.objects[prefix+key]), sum,
			&archivestream.Options{Part: key, Key: dataKey, Compress: compress})
		require.NoError(t, err)
		raw, err := io.ReadAll(r)
		require.NoError(t, err, key)
		r.Close()
		out[key] = raw
	}
	for _, table := range m.Tables {
		read(table.Key, archivestream.Sum{Bytes: table.SizeBytes, SHA256: table.SHA256}, true)
	}
	for _, doc := range m.Documents {
		read(doc.Key, archivestream.Sum{Bytes: doc.SizeBytes, SHA256: doc.SHA256}, false)
	}
	return out
}

func rowsOf(
	t *testing.T,
	in *install,
	table *archiveplan.PlannedTable,
	tenant dbscope.Tenant,
	skip []string,
) []string {
	t.Helper()
	exprs := make([]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		if slices.Contains(skip, column) {
			continue
		}
		expr := "a." + archiveplan.QuoteIdent(column)
		if global, ok := table.Mapped[column]; ok {
			key, _ := archiveplan.GlobalKey(global)
			parts := make([]string, 0, len(key))
			for _, k := range key {
				parts = append(parts, "g."+archiveplan.QuoteIdent(k)+"::text")
			}
			expr = "(SELECT concat_ws('/', " + strings.Join(parts, ", ") + ") FROM " +
				archiveplan.Qualified(global) + " g WHERE g.id = " + expr + ")"
		}
		exprs = append(exprs, "to_jsonb("+expr+")")
	}
	order := archiveplan.JoinIdents(table.PrimaryKey, "a")
	return in.query(t, "SELECT jsonb_build_array("+strings.Join(exprs, ", ")+")::text FROM "+
		archiveplan.Qualified(table.Name)+" a WHERE "+table.Where("a", tenant)+" ORDER BY "+order)
}

func rekeyReferences(t *testing.T, dst *install) {
	t.Helper()
	dst.exec(t, `DO $$
DECLARE fk record;
BEGIN
    SET LOCAL session_replication_role = replica;
    FOR fk IN
        SELECT con.conrelid::regclass AS child, att.attname AS col
        FROM pg_constraint con
        JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = con.conkey[1]
        WHERE con.contype = 'f' AND con.confrelid IN ('us_states'::regclass, 'ifta_jurisdictions'::regclass)
    LOOP
        EXECUTE format('UPDATE %s SET %I = %I || ''_d'' WHERE %I IS NOT NULL', fk.child, fk.col, fk.col, fk.col);
    END LOOP;
    UPDATE us_states SET id = id || '_d';
    UPDATE ifta_jurisdictions SET id = id || '_d';
END $$`)
}

func refusalOf(t *testing.T, err error) tenantarchive.RefusalCode {
	t.Helper()
	require.Error(t, err)
	refusal, ok := tenantarchive.RefusalOf(err)
	require.True(t, ok, "a refusal, not a failure: %v", err)
	return refusal.Code
}

func TestATenantArchiveCarriesOneTenantWholeToAnotherServerAndBack(t *testing.T) {
	src := newInstall(t, true, sourceKey)
	facts := prepareSource(t, src)
	before := src.counts(t, facts.tenant)
	otherBefore := src.counts(t, facts.other)
	keys := archiveKeys(0x11)

	final := export(t, src, facts.tenant, tenantarchive.PurposeFinal, archiveCommand, keys)
	assert.Equal(t, otherBefore, src.counts(t, facts.other), "taking an archive changes nothing")
	prefix := tenantarchive.CommandPrefix(archiveInstance, archiveCommand)
	manifestBody := src.buckets.objects[prefix+tenantarchive.ManifestName]
	assert.Equal(t, tenantarchive.ManifestSHA256(manifestBody), final.SHA256)
	require.NoError(t, final.Manifest.Check())
	assert.Equal(t, "Trenova Logistics", final.Manifest.OrganizationName)
	assert.Len(t, final.Manifest.Documents, len(facts.objects),
		"every object of the tenant and none of the other tenant's")

	t.Run(
		"the archive holds nothing of another tenant and no install-sealed value",
		func(t *testing.T) {
			for key, plain := range plainParts(t, src.buckets, final.Manifest, prefix, keys) {
				for _, foreign := range []string{
					facts.other.OrganizationID.String(), facts.other.BusinessUnitID.String(), envelopeMarker,
				} {
					assert.NotContains(t, string(plain), foreign, key)
				}
				assert.False(t, bytes.Contains(plain, facts.otherBlob), key)
			}
			for key := range src.buckets.objects {
				assert.True(t, strings.HasPrefix(key, prefix), key)
				assert.NotContains(t, key, "export.tar")
			}
		},
	)

	t.Run("an encrypted part does not read without the archive's key", func(t *testing.T) {
		table := final.Manifest.Tables[0]
		stored := src.buckets.objects[prefix+table.Key]
		r, err := archivestream.NewReader(bytes.NewReader(stored),
			archivestream.Sum{Bytes: table.SizeBytes, SHA256: table.SHA256},
			&archivestream.Options{Part: table.Key, Compress: true})
		if err == nil {
			_, err = io.ReadAll(r)
		}
		require.Error(t, err, "the stored bytes are ciphertext, not the rows")
	})

	exported := export(t, src, facts.tenant, tenantarchive.PurposeExport, exportCommand, nil)
	t.Run("an export leaves out the secrets and is readable", func(t *testing.T) {
		m := exported.Manifest
		assert.Nil(t, m.Encryption)
		assert.Contains(t, m.Omitted, tenantarchive.Omission{
			Table: "public.users", Column: "password", Reason: "password hashes",
		})
		assert.Contains(t, m.Omitted, tenantarchive.Omission{
			Table: "public.mfa_authenticators", Reason: "second-factor secrets",
		})
		for key, plain := range plainParts(t, src.buckets, m, tenantarchive.CommandPrefix(archiveInstance, exportCommand), nil) {
			assert.NotContains(t, string(plain), envelopeMarker, key)
			assert.NotContains(t, string(plain), totpSecret, key)
			assert.NotContains(t, string(plain), "$2a$", key)
		}
	})

	dst := newInstall(t, false, destinationKey)
	rekeyReferences(t, dst)
	stranger := seedtest.SeedAdditionalTenant(t, dst.ctx, dst.admin, "ZR")
	strangerTenant := dbscope.Tenant{
		OrganizationID: stranger.Organization.ID,
		BusinessUnitID: stranger.BusinessUnit.ID,
	}
	strangerBefore := dst.counts(t, strangerTenant)
	dst.buckets = src.buckets.clone()
	dst.engine = dst.engineWith(dst.buckets)

	request := func(keys []tenantarchive.Key, expect string) *archiveengine.ImportRequest {
		return &archiveengine.ImportRequest{
			InstanceID:   archiveInstance,
			ExpectSHA256: expect,
			Access:       access(prefix, keys),
		}
	}
	nothingWritten := func(t *testing.T) {
		t.Helper()
		assert.Empty(t, dst.holding(t, facts.tenant), "no row of the tenant is on the server")
		for _, key := range dst.store.Keys() {
			assert.False(t, strings.HasPrefix(key, facts.tenant.OrganizationID.String()), key)
		}
	}

	t.Run("each broken archive is refused before anything is written", func(t *testing.T) {
		_, err := dst.engine.Import(dst.ctx, request(archiveKeys(0x12), final.SHA256))
		assert.Equal(t, tenantarchive.RefusalKey, refusalOf(t, err), "a wrong archive key")
		nothingWritten(t)

		_, err = dst.engine.Import(dst.ctx, request(keys, strings.Repeat("0", 64)))
		assert.Equal(t, tenantarchive.RefusalChecksum, refusalOf(t, err), "another manifest")

		broken := dst.buckets.clone()
		part := prefix + final.Manifest.Tables[3].Key
		broken.objects[part][len(broken.objects[part])/2] ^= 0x01
		_, err = dst.engineWith(broken).Import(dst.ctx, request(keys, final.SHA256))
		assert.Equal(t, tenantarchive.RefusalChecksum, refusalOf(t, err), "a flipped byte")
		nothingWritten(t)

		cut := dst.buckets.clone()
		cut.objects[part] = cut.objects[part][:len(cut.objects[part])-9]
		_, err = dst.engineWith(cut).Import(dst.ctx, request(keys, final.SHA256))
		assert.Equal(t, tenantarchive.RefusalChecksum, refusalOf(t, err), "a truncated part")

		missing := dst.buckets.clone()
		delete(missing.objects, prefix+tenantarchive.ManifestName)
		_, err = dst.engineWith(missing).Import(dst.ctx, request(keys, final.SHA256))
		assert.Equal(t, tenantarchive.RefusalManifest, refusalOf(t, err), "no manifest")

		other := dst.buckets.clone()
		changed := *final.Manifest
		changed.SchemaVersion = "20200101000000"
		body, merr := changed.Encode()
		require.NoError(t, merr)
		other.objects[prefix+tenantarchive.ManifestName] = body
		_, err = dst.engineWith(other).
			Import(dst.ctx, request(keys, tenantarchive.ManifestSHA256(body)))
		assert.Equal(t, tenantarchive.RefusalSchema, refusalOf(t, err), "another schema version")
		assert.Contains(t, err.Error(), "20200101000000")

		_, err = dst.engine.Import(dst.ctx, &archiveengine.ImportRequest{
			InstanceID: "6e2b4d2f-9d2a-4b4a-8b54-6c1e8d5f3a21", ExpectSHA256: final.SHA256,
			Access: access(prefix, keys),
		})
		assert.Equal(
			t,
			tenantarchive.RefusalLocation,
			refusalOf(t, err),
			"an archive of tenant A for tenant B",
		)

		exportPrefix := tenantarchive.CommandPrefix(archiveInstance, exportCommand)
		_, err = dst.engine.Import(dst.ctx, &archiveengine.ImportRequest{
			InstanceID: archiveInstance, ExpectSHA256: exported.SHA256, Access: access(exportPrefix, nil),
		})
		assert.Equal(
			t,
			tenantarchive.RefusalPurpose,
			refusalOf(t, err),
			"an export is never imported",
		)
		nothingWritten(t)
	})

	t.Run(
		"a taken address, an existing id and a missing reference are refused before any write",
		func(t *testing.T) {
			email := src.query(t, "SELECT email_address FROM users WHERE id = $1", facts.user)[0]
			takenID := pulid.MustNew("usr_").String()
			dst.exec(
				t,
				`INSERT INTO users (id, business_unit_id, current_organization_id, username, email_address, name, password, status, timezone)
SELECT $1, $2, $3, 'taken-name', $4, 'Someone Else', 'x', 'Active', 'UTC'`,
				takenID,
				strangerTenant.BusinessUnitID.String(),
				strangerTenant.OrganizationID.String(),
				strings.ToUpper(email),
			)
			_, err := dst.engine.Import(dst.ctx, request(keys, final.SHA256))
			assert.Equal(t, tenantarchive.RefusalTaken, refusalOf(t, err))
			assert.NotContains(
				t,
				strings.ToLower(err.Error()),
				strings.ToLower(email),
				"the holder is not named",
			)
			assert.NotContains(t, err.Error(), "Someone Else")
			dst.exec(t, "DELETE FROM users WHERE id = $1", takenID)
			nothingWritten(t)

			dst.exec(
				t,
				`INSERT INTO users (id, business_unit_id, current_organization_id, username, email_address, name, password, status, timezone)
SELECT $1, $2, $3, 'another-name', 'another@stranger.test', 'Another', 'x', 'Active', 'UTC'`,
				facts.user,
				strangerTenant.BusinessUnitID.String(),
				strangerTenant.OrganizationID.String(),
			)
			_, err = dst.engine.Import(dst.ctx, request(keys, final.SHA256))
			assert.Equal(t, tenantarchive.RefusalExists, refusalOf(t, err))
			dst.exec(t, "DELETE FROM users WHERE id = $1", facts.user)
			nothingWritten(t)

			state := src.query(t, `SELECT s.abbreviation FROM us_states s JOIN organizations o ON o.state_id = s.id WHERE o.id = $1`,
				facts.tenant.OrganizationID.String())[0]
			saved := dst.query(t, `SELECT row_to_json(s)::text FROM us_states s WHERE abbreviation = $1`, state)[0]
			dst.exec(
				t,
				`DO $$ BEGIN SET LOCAL session_replication_role = replica; DELETE FROM us_states WHERE abbreviation = '`+state+`'; END $$`,
			)
			_, err = dst.engine.Import(dst.ctx, request(keys, final.SHA256))
			assert.Equal(t, tenantarchive.RefusalReference, refusalOf(t, err))
			assert.Contains(t, err.Error(), state)
			dst.exec(
				t,
				`DO $$ BEGIN SET LOCAL session_replication_role = replica; INSERT INTO us_states SELECT * FROM json_populate_record(NULL::us_states, '`+saved+`'::json); END $$`,
			)
			nothingWritten(t)
		},
	)

	t.Run(
		"an import stopped midway leaves no row and no document of the tenant",
		func(t *testing.T) {
			ctx, cancel := context.WithCancel(dst.ctx)
			defer cancel()
			req := request(keys, final.SHA256)
			req.Progress = func(p tenantcommand.Progress) {
				if p.Phase == tenantcommand.PhaseLoading && p.TablesDone >= 20 {
					cancel()
				}
			}
			_, err := dst.engine.Import(ctx, req)
			require.Error(t, err)
			nothingWritten(t)
		},
	)

	result, err := dst.engine.Import(dst.ctx, request(keys, final.SHA256))
	require.NoError(t, err)
	assert.Equal(t, facts.tenant, result.Tenant)

	t.Run("every carried table holds the same rows", func(t *testing.T) {
		assert.Equal(t, withProvenance(before, 1), dst.counts(t, facts.tenant))
		plan := dst.plan(t)
		envelope := envelopeTables()
		for _, table := range plan.Tables {
			skip := append(slices.Clone(envelope[table.Name]), resealedColumns[table.Name]...)
			if table.Name == "mfa_recovery_codes" {
				skip = append(skip, "hash_version")
			}
			assert.Equal(
				t,
				rowsOf(t, src, table, facts.tenant, skip),
				withoutProvenance(rowsOf(t, dst, table, facts.tenant, skip)),
				table.Name,
			)
		}
		for _, name := range []string{"invoice_search_projections", "document_storage_totals",
			"customer_ledger_invoice_balances", "customer_ledger_monthly_balances"} {
			left := &archiveplan.PlannedTable{Name: name, Filter: archiveplan.FilterOrganization,
				PrimaryKey: []string{"organization_id"}}
			assert.Equal(
				t,
				src.query(
					t,
					"SELECT to_jsonb(a)::text FROM "+archiveplan.Qualified(
						name,
					)+" a WHERE "+left.Where(
						"a",
						facts.tenant,
					)+" ORDER BY 1",
				),
				dst.query(
					t,
					"SELECT to_jsonb(a)::text FROM "+archiveplan.Qualified(
						name,
					)+" a WHERE "+left.Where(
						"a",
						facts.tenant,
					)+" ORDER BY 1",
				),
				"%s is rebuilt by its triggers to what it was",
				name,
			)
		}
	})

	t.Run(
		"sealed values open with the destination's key and peppered codes are replaced",
		func(t *testing.T) {
			sealed := dst.query(t, "SELECT secret_cipher FROM mfa_authenticators WHERE user_id = $1", facts.user)[0]
			plain, oerr := encryptionservice.NewSecretCipher(dst.cipher).
				Open(dst.ctx, sealed, mfaScope(facts))
			require.NoError(t, oerr)
			assert.Equal(t, totpSecret, plain)
			_, oerr = encryptionservice.NewSecretCipher(src.cipher).
				Open(dst.ctx, sealed, mfaScope(facts))
			require.Error(t, oerr, "the source install's key no longer opens it")
			versions := dst.query(
				t,
				"SELECT hash_version::text FROM mfa_recovery_codes WHERE user_id = $1 AND used_at IS NULL",
				facts.user,
			)
			assert.Equal(
				t,
				[]string{"0"},
				versions,
				"codes hashed with another install's pepper must be made again",
			)
		},
	)

	t.Run(
		"the imported audit trail verifies under this server's key and says where it came from",
		func(t *testing.T) {
			require.NotNil(t, result.Trails)
			assert.Equal(t, archiveports.ChainResealed, result.Trails.AuditSeals.Outcome)
			assert.Equal(t, archiveports.ChainResealed, result.Trails.AIAudit.Outcome)
			assert.Equal(t, "destination-chain", result.Trails.AuditSeals.KeyID)
			status := verifyTrails(t, dst, facts.tenant)
			assert.Equal(t, audit.VerificationVerified, status.audit)
			assert.Equal(t, aiaudit.VerificationVerified, status.ai)
			org := facts.tenant.OrganizationID.String()
			assert.Equal(t, []string{"0"}, dst.query(t,
				`SELECT count(*)::text FROM audit_seals WHERE organization_id = $1
                 AND (source_hash IS NULL OR source_hash_key_id <> 'source-chain' OR hash_key_id <> 'destination-chain')`,
				org), "every seal is signed here and keeps the source's signature beside it")
			assert.Equal(
				t,
				src.query(
					t,
					`SELECT string_agg(hash, ',' ORDER BY seq) FROM ai_audit_events WHERE organization_id = $1`,
					org,
				),
				dst.query(
					t,
					`SELECT string_agg(source_hash, ',' ORDER BY seq) FROM ai_audit_events WHERE organization_id = $1`,
					org,
				),
				"the AI chain's original hashes are kept",
			)
			provenance := dst.query(
				t,
				`SELECT metadata->'archive'->>'manifestSha256' || ' ' || (metadata->'archive'->>'sourceInstanceId')
                     || ' ' || (metadata->'archive'->'rows'->>'audit_seals')
                 FROM audit_entries WHERE organization_id = $1 AND action = $2`,
				org,
				tenantcopyrepository.ActionImported,
			)
			require.Len(t, provenance, 1)
			sealRows := src.query(t, `SELECT count(*)::text FROM audit_seals WHERE organization_id = $1`, org)[0]
			assert.Equal(t, final.SHA256+" "+archiveInstance+" "+sealRows, provenance[0])
		},
	)

	t.Run("documents keep their content types and open as before", func(t *testing.T) {
		types := map[string]string{}
		for _, doc := range final.Manifest.Documents {
			key, _ := tenantarchive.ObjectKey(doc.Key)
			types[key] = doc.ContentType
		}
		seen := map[string]bool{}
		for key := range facts.objects {
			want := src.store.ContentType(key)
			assert.Equal(t, want, types[key], key)
			stored := want
			if stored == "" {
				stored = tenantarchive.DefaultContentType
			}
			assert.Equal(t, stored, dst.store.ContentType(key), key)
			seen[want] = true
			if !facts.sealed[key] {
				continue
			}
			object, oerr := storedobject.Open(dst.ctx, dst.store, dst.cipher, key, storedobject.AAD(
				pagination.TenantInfo{
					OrgID: facts.tenant.OrganizationID,
					BuID:  facts.tenant.BusinessUnitID,
				},
				key,
			))
			require.NoError(t, oerr, key)
			got, rerr := io.ReadAll(object.Body)
			require.NoError(t, rerr)
			object.Body.Close()
			assert.Equal(t, stored, object.ContentType, key)
			assert.True(t, bytes.Equal(facts.objects[key], got), key)
		}
		assert.True(
			t,
			seen["application/pdf"] && seen["image/png"],
			"a PDF and an image are carried",
		)
	})

	t.Run("every document is back byte for byte, sealed for the destination", func(t *testing.T) {
		tenantInfo := pagination.TenantInfo{
			OrgID: facts.tenant.OrganizationID,
			BuID:  facts.tenant.BusinessUnitID,
		}
		for key, want := range facts.objects {
			stored, ok := dst.store.Object(key)
			require.True(t, ok, key)
			if !facts.sealed[key] {
				assert.Equal(t, want, stored, key)
				continue
			}
			opened, oerr := dst.cipher.OpenBlob(
				dst.ctx,
				bytes.NewReader(stored),
				int64(len(stored)),
				encryptionservice.AAD{
					Purpose: encryptionservice.PurposeDocument, OrganizationID: tenantInfo.OrgID,
					BusinessUnitID: tenantInfo.BuID, ResourceID: key,
				},
			)
			require.NoError(t, oerr, key)
			got, rerr := io.ReadAll(opened)
			require.NoError(t, rerr)
			assert.True(t, bytes.Equal(want, got), key)
		}
		assert.Equal(
			t,
			strangerBefore,
			dst.counts(t, strangerTenant),
			"another tenant on the server is untouched",
		)
	})

	t.Run("a changed imported audit row or AI row is caught", func(t *testing.T) {
		org := facts.tenant.OrganizationID.String()
		dst.exec(t, `DO $$ BEGIN SET LOCAL session_replication_role = replica;
UPDATE audit_entries SET comment = 'rewritten after the import'
WHERE organization_id = '`+org+`' AND comment = 'archived change 1';
UPDATE ai_audit_events SET principal_id = 'usr_someone_else'
WHERE organization_id = '`+org+`' AND source_key = 'usage:archive:0:1'; END $$`)
		status := verifyTrails(t, dst, facts.tenant)
		assert.Equal(t, audit.VerificationBroken, status.audit)
		assert.Equal(t, aiaudit.VerificationMismatch, status.ai)
	})

	t.Run("a restore brings back the archived data and nothing else changes", func(t *testing.T) {
		dst.exec(t, "UPDATE organizations SET name = 'Changed After The Archive' WHERE id = $1",
			facts.tenant.OrganizationID.String())
		dst.store.Put(
			facts.tenant.OrganizationID.String()+"/after/new.txt",
			[]byte("written after"),
		)
		purgeOrganization(t, dst, facts.tenant)
		purge := tenantpurgerepository.New(
			tenantpurgerepository.Params{DB: dst.conn, Logger: zap.NewNop()},
		)
		_, derr := purge.DeleteBusinessUnit(dst.ctx, facts.tenant)
		require.NoError(t, derr)

		req := request(keys, final.SHA256)
		req.Tenant = &facts.tenant
		req.Restoring = true
		_, verr := dst.engine.Verify(dst.ctx, req)
		require.NoError(t, verr)
		_, err := dst.engine.Import(dst.ctx, req)
		require.NoError(t, err)
		assert.Equal(
			t,
			[]string{"Trenova Logistics"},
			dst.query(
				t,
				"SELECT name FROM organizations WHERE id = $1",
				facts.tenant.OrganizationID.String(),
			),
		)
		_, found := dst.store.Object(facts.tenant.OrganizationID.String() + "/after/new.txt")
		assert.False(t, found, "data written after the archive is gone")
		assert.Equal(t, withProvenance(before, 1), dst.counts(t, facts.tenant))
		assert.Equal(t, strangerBefore, dst.counts(t, strangerTenant))
		status := verifyTrails(t, dst, facts.tenant)
		assert.Equal(t, audit.VerificationVerified, status.audit,
			"a restore onto a server that did not sign the trail signs it again")
		assert.Equal(t, aiaudit.VerificationVerified, status.ai)
		assert.Equal(
			t,
			[]string{"2"},
			dst.query(
				t,
				"SELECT hash_version::text FROM mfa_recovery_codes WHERE user_id = $1",
				facts.user,
			),
			"a restore on the same server keeps the codes",
		)
	})

	t.Run("a restore on the server that signed the trail keeps its signatures", func(t *testing.T) {
		original := src.query(
			t,
			`SELECT string_agg(hash, ',' ORDER BY seq) FROM audit_seals WHERE organization_id = $1`,
			facts.tenant.OrganizationID.String(),
		)
		purgeOrganization(t, src, facts.tenant)
		purge := tenantpurgerepository.New(
			tenantpurgerepository.Params{DB: src.conn, Logger: zap.NewNop()},
		)
		_, derr := purge.DeleteBusinessUnit(src.ctx, facts.tenant)
		require.NoError(t, derr)
		req := request(keys, final.SHA256)
		req.Tenant = &facts.tenant
		req.Restoring = true
		restored, err := src.engineWith(src.buckets).Import(src.ctx, req)
		require.NoError(t, err)
		assert.Equal(t, archiveports.ChainKept, restored.Trails.AuditSeals.Outcome)
		assert.Equal(t, archiveports.ChainKept, restored.Trails.AIAudit.Outcome)
		assert.Equal(t, original, src.query(t,
			`SELECT string_agg(hash, ',' ORDER BY seq) FROM audit_seals WHERE organization_id = $1`,
			facts.tenant.OrganizationID.String()))
		assert.Equal(t, []string{"0"}, src.query(
			t,
			`SELECT count(*)::text FROM ai_audit_events WHERE organization_id = $1 AND source_hash IS NOT NULL`,
			facts.tenant.OrganizationID.String(),
		))
		status := verifyTrails(t, src, facts.tenant)
		assert.Equal(t, audit.VerificationVerified, status.audit)
		assert.Equal(t, aiaudit.VerificationVerified, status.ai)
	})

	t.Run("an archive of another organization is refused for a restore", func(t *testing.T) {
		req := request(keys, final.SHA256)
		req.Tenant = &strangerTenant
		req.Restoring = true
		_, err := dst.engine.Verify(dst.ctx, req)
		assert.Equal(t, tenantarchive.RefusalInstance, refusalOf(t, err))
	})
}

func TestArchivingALargeTableKeepsMemoryBounded(t *testing.T) {
	in := newInstall(t, false, sourceKey)
	data := seedtest.SeedFullTestData(t, in.ctx, in.admin)
	tenant := dbscope.Tenant{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
	}
	in.exec(
		t,
		`INSERT INTO equipment_manufacturers (id, business_unit_id, organization_id, status, name, description, version, created_at, updated_at)
SELECT 'emf_' || lpad(g::text, 26, '0'), $1, $2, 'Active', 'Maker ' || g,
       (SELECT string_agg(md5(g::text || ':' || r::text), '') FROM generate_series(1, 320) r), 0, 0, 0
FROM generate_series(1, 30000) g`,
		tenant.BusinessUnitID.String(),
		tenant.OrganizationID.String(),
	)
	size := in.query(t, "SELECT pg_total_relation_size('equipment_manufacturers')::text")[0]
	t.Logf("equipment_manufacturers holds %s bytes", size)

	discard := newBucket()
	discard.discard = true
	engine := in.engineWith(discard)

	runtime.GC()
	var base runtime.MemStats
	runtime.ReadMemStats(&base)
	var peak atomic.Uint64
	stop := make(chan struct{})
	sampled := make(chan struct{})
	go func() {
		defer close(sampled)
		var m runtime.MemStats
		for {
			select {
			case <-stop:
				return
			case <-time.After(20 * time.Millisecond):
				runtime.ReadMemStats(&m)
				if m.HeapInuse > peak.Load() {
					peak.Store(m.HeapInuse)
				}
			}
		}
	}()
	result, err := engine.Export(in.ctx, &archiveengine.ExportRequest{
		Purpose:    tenantarchive.PurposeNightly,
		InstanceID: archiveInstance,
		Tenant:     tenant,
		Access: access(
			tenantarchive.NightlyPrefix(archiveInstance, time.Now()),
			archiveKeys(0x21),
		),
	})
	close(stop)
	<-sampled
	require.NoError(t, err)
	table, ok := result.Manifest.Table("public.equipment_manufacturers")
	require.True(t, ok)
	assert.Equal(t, int64(30000), table.Rows)
	growth := int64(peak.Load()) - int64(base.HeapInuse)
	t.Logf("heap grew by %d MiB while archiving", growth>>20)
	assert.Less(t, growth, int64(160<<20),
		"the archive streams: a 300 MB table never sits in memory")
}
