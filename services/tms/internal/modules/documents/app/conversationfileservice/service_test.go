package conversationfileservice_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/app/conversationfileservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRepo struct {
	documentports.DocumentRepository

	doc *document.Document
}

func (r *stubRepo) GetByID(
	_ context.Context,
	_ documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return r.doc, nil
}

type stubStorage struct {
	kernelports.Client

	objects   map[string][]byte
	downloads int
}

func (s *stubStorage) Download(_ context.Context, key string) (*kernelports.DownloadResult, error) {
	s.downloads++

	return &kernelports.DownloadResult{Body: io.NopCloser(bytes.NewReader(s.objects[key]))}, nil
}

func reader(
	t *testing.T,
	doc *document.Document,
	tenant pagination.TenantInfo,
	plaintext []byte,
) (documentports.ConversationFileReader, *stubStorage) {
	t.Helper()

	encryption := encryptionservice.New(encryptionservice.Params{Config: &config.Config{
		Security: config.SecurityConfig{Encryption: config.EncryptionConfig{
			Key: "unit-test-encryption-key-with-at-least-32-bytes",
		}},
	}})
	stored, err := encryption.EncryptBytesWithAAD(plaintext, storedobject.AAD(tenant, doc.StoragePath))
	require.NoError(t, err)
	storage := &stubStorage{objects: map[string][]byte{doc.StoragePath: []byte(stored)}}

	return conversationfileservice.New(conversationfileservice.Params{
		Repo:       &stubRepo{doc: doc},
		Storage:    storage,
		Encryption: encryption,
	}), storage
}

// A file the person attached comes back decrypted, under the associated data
// that binds it to its tenant and path.
func TestReadConversationFile_ReadsAnAttachmentDecrypted(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		FileType:     "application/pdf",
		StoragePath:  "org/bu/assistant_thread/ratecon.pdf",
		ResourceType: document.ConversationResourceType,
		ResourceID:   pulid.MustNew("ath_").String(),
	}
	files, _ := reader(t, doc, tenant, []byte("%PDF-1.7 rate confirmation"))

	file, err := files.ReadConversationFile(t.Context(), documentports.GetDocumentByIDRequest{
		ID: doc.ID, TenantInfo: tenant,
	})

	require.NoError(t, err)
	assert.Equal(t, []byte("%PDF-1.7 rate confirmation"), file.Data)
	assert.Same(t, doc, file.Document)

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: tenant.BuID}
	_, err = files.ReadConversationFile(t.Context(), documentports.GetDocumentByIDRequest{
		ID: doc.ID, TenantInfo: other,
	})
	require.Error(t, err, "bytes stored for one tenant do not decrypt for another")
}

// Only a conversation's files are read this way: any other document is not
// found, and its bytes are never fetched.
func TestReadConversationFile_FindsNoOtherDocument(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		FileType:     "application/pdf",
		StoragePath:  "org/bu/shipment/bol.pdf",
		ResourceType: "shipment",
		ResourceID:   pulid.MustNew("shp_").String(),
	}
	files, storage := reader(t, doc, tenant, []byte("%PDF-1.7 bill of lading"))

	file, err := files.ReadConversationFile(t.Context(), documentports.GetDocumentByIDRequest{
		ID: doc.ID, TenantInfo: tenant,
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Nil(t, file)
	assert.Zero(t, storage.downloads)
}
