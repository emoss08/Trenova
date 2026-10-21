package emailservice

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/mocks/emailportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func sealTestEncryption() *encryptionservice.Service {
	return encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("unit-test-encryption-key-with-at-least-32-bytes"),
	)
}

type sealHarness struct {
	svc     *Service
	repo    *emailportsmock.MockEmailRepository
	storage *kernelportsmock.MockClient
	crypto  *encryptionservice.Service
}

func newSealHarness(t *testing.T) *sealHarness {
	t.Helper()

	repo := emailportsmock.NewMockEmailRepository(t)
	storage := kernelportsmock.NewMockClient(t)
	crypto := sealTestEncryption()

	return &sealHarness{
		svc: &Service{
			l:                 zap.NewNop(),
			repo:              repo,
			storage:           storage,
			encryptionService: crypto,
		},
		repo:    repo,
		storage: storage,
		crypto:  crypto,
	}
}

func (h *sealHarness) stored(t *testing.T, key string, body []byte) {
	t.Helper()

	h.storage.EXPECT().Download(mock.Anything, key).Return(&kernelports.DownloadResult{
		Body: io.NopCloser(bytes.NewReader(body)),
		Size: int64(len(body)),
	}, nil).Once()
}

func TestProviderAttachmentsUnsealsADocumentObject(t *testing.T) {
	t.Parallel()

	h := newSealHarness(t)
	tenant := testTenantInfo()
	messageID := pulid.MustNew("emlmsg_")
	key := tenant.OrgID.String() + "/shipment/eld.pdf"
	plaintext := []byte("%PDF-1.7 ELD record")
	sealed, err := sealBlob(t, h.crypto, plaintext, encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ResourceID:     key,
	})
	require.NoError(t, err)

	h.repo.EXPECT().ListAttachments(mock.Anything, emailports.ListEmailAttachmentsRequest{
		MessageID:  messageID,
		TenantInfo: tenant,
	}).Return([]*email.Attachment{{
		ID:             pulid.MustNew("emlatt_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		MessageID:      messageID,
		FileName:       "eld.pdf",
		ContentType:    "application/pdf",
		ObjectKey:      key,
		Seal:           email.AttachmentSealDocument,
	}}, nil).Once()
	h.stored(t, key, sealed)

	attachments, err := h.svc.providerAttachments(t.Context(), tenant, messageID)

	require.NoError(t, err)
	require.Len(t, attachments, 1)
	assert.Equal(t, plaintext, attachments[0].Content)
	assert.Equal(t, "eld.pdf", attachments[0].FileName)
}

func TestProviderAttachmentsRefusesADocumentSealedForAnotherTenant(t *testing.T) {
	t.Parallel()

	h := newSealHarness(t)
	tenant := testTenantInfo()
	other := testTenantInfo()
	messageID := pulid.MustNew("emlmsg_")
	key := other.OrgID.String() + "/shipment/eld.pdf"
	sealed, err := sealBlob(t, h.crypto, []byte("%PDF"), encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: other.OrgID,
		BusinessUnitID: other.BuID,
		ResourceID:     key,
	})
	require.NoError(t, err)

	h.repo.EXPECT().ListAttachments(mock.Anything, mock.Anything).Return([]*email.Attachment{{
		ID:             pulid.MustNew("emlatt_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		MessageID:      messageID,
		FileName:       "eld.pdf",
		ObjectKey:      key,
		Seal:           email.AttachmentSealDocument,
	}}, nil).Once()
	h.stored(t, key, sealed)

	_, err = h.svc.providerAttachments(t.Context(), tenant, messageID)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNonRetryableSend))
}

func TestProviderAttachmentsPassesAPlainObjectThrough(t *testing.T) {
	t.Parallel()

	h := newSealHarness(t)
	tenant := testTenantInfo()
	messageID := pulid.MustNew("emlmsg_")
	key := "email-attachments/report.csv"

	h.repo.EXPECT().ListAttachments(mock.Anything, mock.Anything).Return([]*email.Attachment{{
		ID:          pulid.MustNew("emlatt_"),
		FileName:    "report.csv",
		ContentType: "text/csv",
		ObjectKey:   key,
	}}, nil).Once()
	h.stored(t, key, []byte("a,b"))

	attachments, err := h.svc.providerAttachments(t.Context(), tenant, messageID)

	require.NoError(t, err)
	require.Len(t, attachments, 1)
	assert.Equal(t, []byte("a,b"), attachments[0].Content)
}

func TestPersistAttachmentKeepsTheSealAndRefusesSealedContent(t *testing.T) {
	t.Parallel()

	h := newSealHarness(t)
	tenant := testTenantInfo()
	msg := &email.Message{
		ID:             pulid.MustNew("emlmsg_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}

	stored, err := h.svc.persistAttachment(t.Context(), msg, emailports.EmailAttachment{
		FileName:    "eld.pdf",
		ContentType: "application/pdf",
		ObjectKey:   "org/shipment/eld.pdf",
		SizeBytes:   2048,
		Seal:        email.AttachmentSealDocument,
	})
	require.NoError(t, err)
	assert.Equal(t, email.AttachmentSealDocument, stored.Seal)
	assert.Equal(t, "org/shipment/eld.pdf", stored.ObjectKey)

	_, err = h.svc.persistAttachment(t.Context(), msg, emailports.EmailAttachment{
		FileName: "eld.pdf",
		Content:  []byte("%PDF"),
		Seal:     email.AttachmentSealDocument,
	})
	require.Error(t, err)

	_, err = h.svc.persistAttachment(t.Context(), msg, emailports.EmailAttachment{
		FileName:  "eld.pdf",
		ObjectKey: "org/shipment/eld.pdf",
		Seal:      email.AttachmentSeal("Vault"),
	})
	require.Error(t, err)
}

func sealBlob(
	t *testing.T,
	crypto *encryptionservice.Service,
	plaintext []byte,
	aad encryptionservice.AAD,
) ([]byte, error) {
	t.Helper()
	sealed, err := crypto.SealBlob(t.Context(), bytes.NewReader(plaintext), int64(len(plaintext)), aad)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(sealed)
}
