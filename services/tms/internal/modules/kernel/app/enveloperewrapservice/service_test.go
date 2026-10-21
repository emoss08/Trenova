package enveloperewrapservice

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	oldKey = "the-previous-local-encryption-key-of-32-chars"
	newKey = "the-current-local-encryption-key-of-32-chars!"
)

type rows struct {
	tenant pagination.TenantInfo
	values map[string]*kernelports.EnvelopeRow
	order  []string
}

type fakeRepository struct {
	tenants  []pagination.TenantInfo
	columns  map[string]*rows
	versions map[string]string
	present  bool
	swaps    int
	conflict bool
}

func newFakeRepository(tenant pagination.TenantInfo) *fakeRepository {
	return &fakeRepository{
		tenants:  []pagination.TenantInfo{tenant},
		columns:  make(map[string]*rows),
		versions: make(map[string]string),
		present:  true,
	}
}

func (f *fakeRepository) add(column string, tenant pagination.TenantInfo, id, sealed string) {
	table, ok := f.columns[column]
	if !ok {
		table = &rows{tenant: tenant, values: make(map[string]*kernelports.EnvelopeRow)}
		f.columns[column] = table
	}
	table.values[id] = &kernelports.EnvelopeRow{
		ID:     id,
		Tenant: tenant,
		Sealed: sealed,
		Extra:  map[string]string{},
	}
	table.order = append(table.order, id)
}

func (f *fakeRepository) value(column, id string) string {
	return f.columns[column].values[id].Sealed
}

func (f *fakeRepository) ListTenants(context.Context) ([]pagination.TenantInfo, error) {
	return f.tenants, nil
}

func (f *fakeRepository) ListRows(
	_ context.Context,
	req *kernelports.ListEnvelopeRowsRequest,
) ([]*kernelports.EnvelopeRow, error) {
	table, ok := f.columns[req.Column.Name]
	if !ok || table.tenant != req.Tenant {
		return nil, nil
	}
	out := make([]*kernelports.EnvelopeRow, 0, req.Limit)
	for _, id := range table.order {
		if id <= req.After {
			continue
		}
		row := *table.values[id]
		out = append(out, &row)
		if len(out) == req.Limit {
			break
		}
	}
	return out, nil
}

func (f *fakeRepository) Present(
	context.Context,
	*kernelports.EnvelopeColumn,
	*kernelports.EnvelopeRow,
) (bool, error) {
	return f.present, nil
}

func (f *fakeRepository) Swap(_ context.Context, req *kernelports.SwapEnvelopeRequest) (bool, error) {
	if f.conflict {
		return false, nil
	}
	current := f.columns[req.Column.Name].values[req.Row.ID]
	if current.Sealed != req.Row.Sealed {
		return false, nil
	}
	current.Sealed = req.Replacement
	f.swaps++
	return true, nil
}

func (f *fakeRepository) SwapVersion(
	_ context.Context,
	req *kernelports.SwapEnvelopeVersionRequest,
) error {
	f.versions[req.Row.ID] = req.Replacement
	return nil
}

type fakeStorage struct {
	kernelports.Client

	objects  map[string][]byte
	uploads  int
	versions int
}

func (f *fakeStorage) Download(_ context.Context, key string) (*kernelports.DownloadResult, error) {
	return &kernelports.DownloadResult{
		Body:        io.NopCloser(bytes.NewReader(f.objects[key])),
		ContentType: "application/pdf",
		Metadata:    map[string]string{"crypto_mode": encryptionservice.CryptoModeEnvelopeV1},
	}, nil
}

func (f *fakeStorage) Upload(
	_ context.Context,
	params *kernelports.UploadParams,
) (*kernelports.FileInfo, error) {
	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	f.objects[params.Key] = body
	f.uploads++
	f.versions++
	return &kernelports.FileInfo{Key: params.Key, VersionID: "v" + strconv.Itoa(f.versions)}, nil
}

func keyring(t *testing.T, active string, retired ...string) *encryptionservice.Service {
	t.Helper()
	manager, err := encryptionservice.NewLocalKeyring(active, retired)
	require.NoError(t, err)
	return encryptionservice.NewWithKeyManager(manager)
}

func secretColumn(name string, scopes kernelports.EnvelopeScopes) kernelports.EnvelopeColumn {
	return kernelports.EnvelopeColumn{
		Name:           name,
		Table:          buncolgen.TableInfo{Name: "secrets", Alias: "sec"},
		ID:             buncolgen.NewColumn("id", "sec"),
		OrganizationID: buncolgen.NewColumn("organization_id", "sec"),
		BusinessUnitID: buncolgen.NewColumn("business_unit_id", "sec"),
		Value:          buncolgen.NewColumn("secret", "sec"),
		Scopes:         scopes,
	}
}

func boundAAD(tenant pagination.TenantInfo, resourceID string) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeIntegrationSecret,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ResourceID:     resourceID,
	}
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestRun_MovesSingleValuesToTheActiveKey(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	old := keyring(t, oldKey)
	rotated := keyring(t, newKey, oldKey)
	newOnly := keyring(t, newKey)

	repo := newFakeRepository(tenant)
	sealed := make([]string, 0, 3)
	for i := range 3 {
		id := "row_" + strconv.Itoa(i)
		value, err := old.EncryptStringWithAAD("secret-"+id, boundAAD(tenant, id))
		require.NoError(t, err)
		repo.add("secrets", tenant, id, value)
		sealed = append(sealed, value)
	}
	current, err := rotated.EncryptStringWithAAD("secret-row_3", boundAAD(tenant, "row_3"))
	require.NoError(t, err)
	repo.add("secrets", tenant, "row_3", current)
	repo.add("secrets", tenant, "row_4", "plaintext-from-before-encryption")

	column := secretColumn("secrets", func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
		return []kernelports.SecretScope{boundAAD(row.Tenant, row.ID).SecretScope()}
	})
	service := New(Deps{Repository: repo, Rewrapper: encryptionservice.NewEnvelopeRewrapper(rotated)})

	report, err := service.Run(
		t.Context(),
		&kernelports.EnvelopeSources{Columns: []kernelports.EnvelopeColumn{column}},
		Options{BatchSize: 2},
	)
	require.NoError(t, err)
	require.Len(t, report.Sources, 1)
	assert.Equal(t, SourceReport{Name: "secrets", Scanned: 5, Rewrapped: 3, Current: 1, Skipped: 1},
		*report.Sources[0])
	assert.Zero(t, report.Remaining())

	for i := range 3 {
		id := "row_" + strconv.Itoa(i)
		assert.NotEqual(t, sealed[i], repo.value("secrets", id))
		plaintext, err := newOnly.DecryptStringWithAAD(repo.value("secrets", id), boundAAD(tenant, id))
		require.NoError(t, err)
		assert.Equal(t, "secret-"+id, plaintext)
	}
	assert.Equal(t, current, repo.value("secrets", "row_3"))
}

func TestRun_DryRunWritesNothing(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	value, err := keyring(t, oldKey).EncryptStringWithAAD("secret", boundAAD(tenant, "row_1"))
	require.NoError(t, err)
	repo := newFakeRepository(tenant)
	repo.add("secrets", tenant, "row_1", value)

	column := secretColumn("secrets", func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
		return []kernelports.SecretScope{boundAAD(row.Tenant, row.ID).SecretScope()}
	})
	service := New(Deps{
		Repository: repo,
		Rewrapper:  encryptionservice.NewEnvelopeRewrapper(keyring(t, newKey, oldKey)),
	})

	report, err := service.Run(
		t.Context(),
		&kernelports.EnvelopeSources{Columns: []kernelports.EnvelopeColumn{column}},
		Options{DryRun: true},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Sources[0].Rewrapped)
	assert.Equal(t, 1, report.Remaining())
	assert.Zero(t, repo.swaps)
	assert.Equal(t, value, repo.value("secrets", "row_1"))
}

func TestRun_RebindsALegacyValueOntoItsFirstScope(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	legacy, err := keyring(t, oldKey).EncryptString("legacy-api-key")
	require.NoError(t, err)
	repo := newFakeRepository(tenant)
	repo.add("keys", tenant, "row_1", legacy)

	column := secretColumn("keys", func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
		return []kernelports.SecretScope{boundAAD(row.Tenant, row.ID).SecretScope(), {}}
	})
	column.Rebind = true
	newOnly := keyring(t, newKey)
	service := New(Deps{
		Repository: repo,
		Rewrapper:  encryptionservice.NewEnvelopeRewrapper(keyring(t, newKey, oldKey)),
	})

	report, err := service.Run(
		t.Context(),
		&kernelports.EnvelopeSources{Columns: []kernelports.EnvelopeColumn{column}},
		Options{},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Sources[0].Rewrapped)

	stored := repo.value("keys", "row_1")
	assert.False(t, encryptionservice.EnvelopeHasAAD(stored, encryptionservice.AAD{}))
	plaintext, err := newOnly.DecryptStringWithAAD(stored, boundAAD(tenant, "row_1"))
	require.NoError(t, err)
	assert.Equal(t, "legacy-api-key", plaintext)
}

func TestRun_RewrapsEveryEnvelopeInAJSONMap(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	old := keyring(t, oldKey)
	apiKey, err := old.EncryptStringWithAAD("api-key", boundAAD(tenant, "integration:x:apiKey"))
	require.NoError(t, err)
	webhook, err := old.EncryptStringWithAAD("hook", boundAAD(tenant, "integration:x:webhookSecret"))
	require.NoError(t, err)
	stored, err := sonic.MarshalString(map[string]any{
		"apiKey":        apiKey,
		"webhookSecret": webhook,
		"region":        "us-east",
		"pollMinutes":   5,
	})
	require.NoError(t, err)

	repo := newFakeRepository(tenant)
	repo.add("integration secrets", tenant, "row_1", stored)
	column := secretColumn(
		"integration secrets",
		func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
			return []kernelports.SecretScope{boundAAD(row.Tenant, "integration:x:"+field).SecretScope()}
		},
	)
	column.JSONMap = true
	service := New(Deps{
		Repository: repo,
		Rewrapper:  encryptionservice.NewEnvelopeRewrapper(keyring(t, newKey, oldKey)),
	})

	report, err := service.Run(
		t.Context(),
		&kernelports.EnvelopeSources{Columns: []kernelports.EnvelopeColumn{column}},
		Options{},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Sources[0].Rewrapped)

	fields := make(map[string]any)
	require.NoError(t, sonic.UnmarshalString(repo.value("integration secrets", "row_1"), &fields))
	assert.Equal(t, "us-east", fields["region"])
	assert.InDelta(t, 5, fields["pollMinutes"], 0)
	newOnly := keyring(t, newKey)
	for field, want := range map[string]string{"apiKey": "api-key", "webhookSecret": "hook"} {
		sealed, ok := fields[field].(string)
		require.True(t, ok)
		plaintext, err := newOnly.DecryptStringWithAAD(
			sealed,
			boundAAD(tenant, "integration:x:"+field),
		)
		require.NoError(t, err, field)
		assert.Equal(t, want, plaintext)
	}
}

func TestRun_ReportsValuesItCannotOpenAndKeepsGoing(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	unknown, err := keyring(t, "a-key-nobody-kept-around-of-at-least-32").
		EncryptStringWithAAD("lost", boundAAD(tenant, "row_1"))
	require.NoError(t, err)
	movable, err := keyring(t, oldKey).EncryptStringWithAAD("kept", boundAAD(tenant, "row_2"))
	require.NoError(t, err)
	repo := newFakeRepository(tenant)
	repo.add("secrets", tenant, "row_1", unknown)
	repo.add("secrets", tenant, "row_2", movable)

	column := secretColumn("secrets", func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
		return []kernelports.SecretScope{boundAAD(row.Tenant, row.ID).SecretScope()}
	})
	service := New(Deps{
		Repository: repo,
		Rewrapper:  encryptionservice.NewEnvelopeRewrapper(keyring(t, newKey, oldKey)),
	})

	report, err := service.Run(
		t.Context(),
		&kernelports.EnvelopeSources{Columns: []kernelports.EnvelopeColumn{column}},
		Options{},
	)
	require.NoError(t, err)
	source := report.Sources[0]
	assert.Equal(t, 1, source.Failed)
	assert.Equal(t, 1, source.Rewrapped)
	require.Len(t, source.Errors, 1)
	assert.Contains(t, source.Errors[0], "row_1")
	assert.Equal(t, 1, report.Failed())
	assert.Equal(t, unknown, repo.value("secrets", "row_1"))
}

func TestRun_ReportsARowChangedUnderneathAsAConflict(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	value, err := keyring(t, oldKey).EncryptStringWithAAD("secret", boundAAD(tenant, "row_1"))
	require.NoError(t, err)
	repo := newFakeRepository(tenant)
	repo.add("secrets", tenant, "row_1", value)
	repo.conflict = true

	column := secretColumn("secrets", func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
		return []kernelports.SecretScope{boundAAD(row.Tenant, row.ID).SecretScope()}
	})
	service := New(Deps{
		Repository: repo,
		Rewrapper:  encryptionservice.NewEnvelopeRewrapper(keyring(t, newKey, oldKey)),
	})

	report, err := service.Run(
		t.Context(),
		&kernelports.EnvelopeSources{Columns: []kernelports.EnvelopeColumn{column}},
		Options{},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Sources[0].Conflicts)
	assert.Equal(t, 1, report.Remaining())
}

func documentObject() kernelports.EnvelopeObject {
	version := buncolgen.NewColumn("storage_version_id", "doc")
	column := secretColumn("documents", func(row *kernelports.EnvelopeRow, key string) []kernelports.SecretScope {
		return []kernelports.SecretScope{encryptionservice.AAD{
			Purpose:        encryptionservice.PurposeDocument,
			OrganizationID: row.Tenant.OrgID,
			BusinessUnitID: row.Tenant.BuID,
			ResourceID:     key,
		}.SecretScope()}
	})
	column.Extra = []buncolgen.Column{version}
	return kernelports.EnvelopeObject{Column: column, Version: &version}
}

func TestRun_RewritesDocumentObjectsAndRecordsTheirNewVersion(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	key := tenant.OrgID.String() + "/documents/invoice.pdf"
	aad := encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ResourceID:     key,
	}
	sealed, err := keyring(t, oldKey).EncryptBytesWithAAD([]byte("%PDF-1.7"), aad)
	require.NoError(t, err)

	repo := newFakeRepository(tenant)
	repo.add("documents", tenant, "doc_1", key)
	storage := &fakeStorage{objects: map[string][]byte{key: []byte(sealed)}}
	service := New(Deps{
		Repository: repo,
		Rewrapper:  encryptionservice.NewEnvelopeRewrapper(keyring(t, newKey, oldKey)),
		Storage:    storage,
	})

	report, err := service.Run(
		t.Context(),
		&kernelports.EnvelopeSources{Objects: []kernelports.EnvelopeObject{documentObject()}},
		Options{},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Sources[0].Rewrapped)
	assert.Equal(t, "v1", repo.versions["doc_1"])

	plaintext, err := keyring(t, newKey).DecryptBytesWithAAD(string(storage.objects[key]), aad)
	require.NoError(t, err)
	assert.Equal(t, []byte("%PDF-1.7"), plaintext)

	report, err = service.Run(
		t.Context(),
		&kernelports.EnvelopeSources{Objects: []kernelports.EnvelopeObject{documentObject()}},
		Options{},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Sources[0].Current)
	assert.Equal(t, 1, storage.uploads)
}

func TestRun_NeverRewritesTheObjectOfADeletedDocument(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	key := tenant.OrgID.String() + "/documents/deleted.pdf"
	sealed, err := keyring(t, oldKey).EncryptBytesWithAAD([]byte("%PDF-1.7"), encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ResourceID:     key,
	})
	require.NoError(t, err)

	repo := newFakeRepository(tenant)
	repo.add("documents", tenant, "doc_1", key)
	repo.present = false
	storage := &fakeStorage{objects: map[string][]byte{key: []byte(sealed)}}
	service := New(Deps{
		Repository: repo,
		Rewrapper:  encryptionservice.NewEnvelopeRewrapper(keyring(t, newKey, oldKey)),
		Storage:    storage,
	})

	report, err := service.Run(
		t.Context(),
		&kernelports.EnvelopeSources{Objects: []kernelports.EnvelopeObject{documentObject()}},
		Options{},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Sources[0].Conflicts)
	assert.Zero(t, storage.uploads)
	assert.Equal(t, sealed, string(storage.objects[key]))
}

func TestRun_RefusesObjectsWithoutStorage(t *testing.T) {
	t.Parallel()

	service := New(Deps{Repository: newFakeRepository(testTenant())})
	_, err := service.Run(
		t.Context(),
		&kernelports.EnvelopeSources{Objects: []kernelports.EnvelopeObject{documentObject()}},
		Options{},
	)
	require.ErrorIs(t, err, ErrStorageRequired)
}
