package tenantarchive_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const instance = "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"

func manifest() *tenantarchive.Manifest {
	m := &tenantarchive.Manifest{
		FormatVersion:    1,
		SchemaVersion:    "20261231023500",
		ReleaseVersion:   "2026.11.0",
		InstanceID:       instance,
		OrganizationName: "Acme Freight",
		TakenAt:          time.Date(2026, 11, 20, 7, 0, 0, 0, time.UTC),
		Purpose:          tenantarchive.PurposeFinal,
		Encryption: &tenantarchive.Encryption{
			Algorithm:      tenantarchive.EncryptionAlgorithm,
			SegmentBytes:   tenantarchive.SegmentBytes,
			KeyID:          "region-1",
			WrappedDataKey: "AAAA",
		},
		Tables: []tenantarchive.Table{
			{Name: "public.business_units", Rows: 1, Key: "tables/0001-business_units.copy.zst",
				SizeBytes: 40, SHA256: strings.Repeat("a", 64), Encoding: tenantarchive.TableEncoding},
			{Name: "public.shipments", Rows: 9, Key: "tables/0002-shipments.copy.zst",
				SizeBytes: 900, SHA256: strings.Repeat("b", 64), Encoding: tenantarchive.TableEncoding},
		},
		Documents: []tenantarchive.Document{
			{
				Key:       "documents/org_1/shipment/doc_1.pdf",
				SizeBytes: 77,
				SHA256:    strings.Repeat("c", 64),
			},
		},
	}
	m.Seal()
	return m
}

func TestTheContentDigestIsSha256sumOfThePartsTablesFirst(t *testing.T) {
	t.Parallel()

	m := manifest()
	lines := strings.Repeat("a", 64) + "  tables/0001-business_units.copy.zst\n" +
		strings.Repeat("b", 64) + "  tables/0002-shipments.copy.zst\n" +
		strings.Repeat("c", 64) + "  documents/org_1/shipment/doc_1.pdf\n"
	sum := sha256.Sum256([]byte(lines))
	assert.Equal(t, hex.EncodeToString(sum[:]), m.ContentSHA256)
	assert.Equal(t, int64(1017), m.TotalSizeBytes)
	require.NoError(t, m.Check())

	body, err := m.Encode()
	require.NoError(t, err)
	decoded, err := tenantarchive.DecodeManifest(body)
	require.NoError(t, err)
	require.NoError(t, decoded.Check())
	assert.NotContains(t, string(body), "omitted", "only an export lists omissions")
}

func TestAManifestThatDisagreesWithItselfIsRefused(t *testing.T) {
	t.Parallel()

	for name, change := range map[string]func(*tenantarchive.Manifest){
		"a part's digest":      func(m *tenantarchive.Manifest) { m.Tables[1].SHA256 = strings.Repeat("d", 64) },
		"a part out of order":  func(m *tenantarchive.Manifest) { m.Tables[0], m.Tables[1] = m.Tables[1], m.Tables[0] },
		"the total":            func(m *tenantarchive.Manifest) { m.TotalSizeBytes++ },
		"an unencrypted final": func(m *tenantarchive.Manifest) { m.Encryption = nil },
		"another encoding":     func(m *tenantarchive.Manifest) { m.Tables[0].Encoding = "csv" },
		"a document outside":   func(m *tenantarchive.Manifest) { m.Documents[0].Key = "documents/../x" },
		"a future format":      func(m *tenantarchive.Manifest) { m.FormatVersion = 2 },
		"a header in a content type": func(m *tenantarchive.Manifest) {
			m.Documents[0].ContentType = "text/html\r\nX-Injected: 1"
		},
		"a content type that is not a type": func(m *tenantarchive.Manifest) {
			m.Documents[0].ContentType = "pdf"
		},
	} {
		m := manifest()
		change(m)
		err := m.Check()
		refusal, ok := tenantarchive.RefusalOf(err)
		require.True(t, ok, name)
		assert.Equal(t, tenantarchive.RefusalManifest, refusal.Code, name)
	}
}

func TestADocumentKeyThatIsNotACleanObjectKeyIsRefused(t *testing.T) {
	t.Parallel()

	for _, key := range []string{
		"documents/org_1//x.pdf",
		"documents/org_1/a\\b.pdf",
		"documents//org_1/x.pdf",
		"documents/org_1/x/",
		"documents/org_1/./x.pdf",
		"documents/org_1/x\x00.pdf",
		"documents/",
	} {
		m := manifest()
		m.Documents[0].Key = key
		m.Seal()
		refusal, ok := tenantarchive.RefusalOf(m.Check())
		require.True(t, ok, "%q", key)
		assert.Equal(t, tenantarchive.RefusalManifest, refusal.Code, "%q", key)
	}
}

func TestAccessNeedsOneCurrentKeyToWriteAndNoneForAnExport(t *testing.T) {
	t.Parallel()

	key := make([]byte, tenantarchive.KeySize)
	access := &tenantarchive.Access{
		Location: tenantarchive.Location{
			Endpoint: "https://objects.test", Bucket: "archives",
			Prefix: tenantarchive.CommandPrefix(instance, instance),
		},
		Credentials: tenantarchive.Credentials{AccessKey: "a", SecretKey: "b"},
		Keys: []tenantarchive.Key{
			{ID: "old", Material: key},
			{ID: "new", Material: key, Current: true},
		},
	}
	require.NoError(t, access.CheckWrite(instance, tenantarchive.PurposeFinal))
	assert.Equal(t, "new", access.CurrentKey().ID)
	require.ErrorIs(
		t,
		access.CheckWrite(instance, tenantarchive.PurposeExport),
		tenantarchive.ErrKeyNotExpected,
	)
	found, err := access.KeyByID("old")
	require.NoError(t, err)
	assert.Equal(t, "old", found.ID)
	_, err = access.KeyByID("gone")
	require.ErrorIs(t, err, tenantarchive.ErrKeyUnknown)

	access.Keys[0].Current = true
	require.ErrorIs(
		t,
		access.CheckWrite(instance, tenantarchive.PurposeFinal),
		tenantarchive.ErrKeyInvalid,
	)
	access.Location.Prefix = "tenants/" + "6e2b4d2f-9d2a-4b4a-8b54-6c1e8d5f3a21" + "/archives/x/"
	require.ErrorIs(t, access.CheckRead(instance), tenantarchive.ErrLocationInvalid,
		"an archive of another tenant is out of reach")
}

func TestADocumentsContentTypeIsOptionalAndDefaultsToOctetStream(t *testing.T) {
	t.Parallel()

	m := manifest()
	require.NoError(t, m.Check())
	assert.Equal(t, tenantarchive.DefaultContentType, m.Documents[0].StoredContentType())

	for _, contentType := range []string{
		"application/pdf", "image/png", "text/plain; charset=utf-8",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	} {
		m.Documents[0].ContentType = contentType
		require.NoError(t, m.Check(), contentType)
		assert.Equal(t, contentType, m.Documents[0].StoredContentType())
	}
	body, err := m.Encode()
	require.NoError(t, err)
	decoded, err := tenantarchive.DecodeManifest(body)
	require.NoError(t, err)
	assert.Equal(t, m.Documents[0].ContentType, decoded.Documents[0].ContentType)
}
