package emailservice

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	kmspb "cloud.google.com/go/kms/apiv1/kmspb"

	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/mocks/emailportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	gax "github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type failingAfter struct {
	data []byte
	err  error
}

func (r *failingAfter) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func sealedAttachment(
	t *testing.T,
	h *sealHarness,
	plaintext []byte,
) (*email.Attachment, []byte) {
	t.Helper()

	tenant := testTenantInfo()
	key := tenant.OrgID.String() + "/shipment/pod.pdf"
	sealed, err := sealBlob(t, h.crypto, plaintext, encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ResourceID:     key,
	})
	require.NoError(t, err)
	return &email.Attachment{
		ID:             pulid.MustNew("emlatt_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		FileName:       "pod.pdf",
		ObjectKey:      key,
		Seal:           email.AttachmentSealDocument,
	}, sealed
}

func TestAttachmentContentKeepsAStorageReadErrorRetryable(t *testing.T) {
	t.Parallel()

	plaintext := bytes.Repeat([]byte("POD "), 3*encryptionservice.BlobChunkSize/4)
	reset := errors.New("connection reset by peer")

	for name, cut := range map[string]func(sealed []byte) int{
		"inside the header":  func([]byte) int { return 20 },
		"between the chunks": func(sealed []byte) int { return len(sealed) / 2 },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := newSealHarness(t)
			attachment, sealed := sealedAttachment(t, h, plaintext)
			h.storage.EXPECT().Download(mock.Anything, attachment.ObjectKey).Return(&kernelports.DownloadResult{
				Body: io.NopCloser(&failingAfter{data: sealed[:cut(sealed)], err: reset}),
				Size: int64(len(sealed)),
			}, nil).Once()

			_, err := h.svc.attachmentContent(t.Context(), attachment)

			require.ErrorIs(t, err, reset)
			assert.False(t, errors.Is(err, ErrNonRetryableSend), "a storage read error is retried")
		})
	}
}

func TestAttachmentContentRefusesATamperedBlobForGood(t *testing.T) {
	t.Parallel()

	h := newSealHarness(t)
	attachment, sealed := sealedAttachment(t, h, []byte("%PDF-1.7 proof of delivery"))
	sealed[len(sealed)-1] ^= 0xff
	h.stored(t, attachment.ObjectKey, sealed)

	_, err := h.svc.attachmentContent(t.Context(), attachment)

	require.ErrorIs(t, err, encryptionservice.ErrBlobTampered)
	assert.True(t, errors.Is(err, ErrNonRetryableSend))
}

type unavailableKMS struct {
	decryptErr error
}

func (c *unavailableKMS) Encrypt(
	_ context.Context,
	req *kmspb.EncryptRequest,
	_ ...gax.CallOption,
) (*kmspb.EncryptResponse, error) {
	return &kmspb.EncryptResponse{Ciphertext: req.GetPlaintext()}, nil
}

func (c *unavailableKMS) Decrypt(
	context.Context,
	*kmspb.DecryptRequest,
	...gax.CallOption,
) (*kmspb.DecryptResponse, error) {
	return nil, c.decryptErr
}

func (c *unavailableKMS) Close() error {
	return nil
}

func TestAttachmentContentRetriesAKeyServiceOutage(t *testing.T) {
	t.Parallel()

	h := newSealHarness(t)
	client := &unavailableKMS{}
	h.crypto = encryptionservice.NewWithKeyManager(encryptionservice.NewGCPAutokeyManager(
		client,
		"projects/test/locations/us/keyRings/autokey/cryptoKeys/trenova",
		time.Second,
	))
	h.svc.encryptionService = h.crypto
	attachment, sealed := sealedAttachment(t, h, []byte("%PDF-1.7 proof of delivery"))
	client.decryptErr = status.Error(codes.Unavailable, "kms is down")
	h.stored(t, attachment.ObjectKey, sealed)

	_, err := h.svc.attachmentContent(t.Context(), attachment)

	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.False(t, errors.Is(err, ErrNonRetryableSend), "a key service outage is retried")
}

func TestAttachmentContentRefusesAnUnknownKeyForGood(t *testing.T) {
	t.Parallel()

	h := newSealHarness(t)
	sealer := encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("a-key-this-deployment-no-longer-holds-32b"),
	)
	attachment, _ := sealedAttachment(t, h, []byte("%PDF"))
	sealed, err := sealBlob(t, sealer, []byte("%PDF"), encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: attachment.OrganizationID,
		BusinessUnitID: attachment.BusinessUnitID,
		ResourceID:     attachment.ObjectKey,
	})
	require.NoError(t, err)
	h.stored(t, attachment.ObjectKey, sealed)

	_, err = h.svc.attachmentContent(t.Context(), attachment)

	require.ErrorIs(t, err, encryptionservice.ErrUnknownKeyID)
	assert.True(t, errors.Is(err, ErrNonRetryableSend))
}

func TestAttachmentContentRetriesATruncatedDownload(t *testing.T) {
	t.Parallel()

	plaintext := bytes.Repeat([]byte("POD "), 3*encryptionservice.BlobChunkSize/4)
	for name, cut := range map[string]func(sealed []byte) int{
		"inside the header": func([]byte) int { return 20 },
		"inside a chunk":    func(sealed []byte) int { return len(sealed)/2 + 7 },
		"before the tag":    func(sealed []byte) int { return len(sealed) - 1 },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := newSealHarness(t)
			attachment, sealed := sealedAttachment(t, h, plaintext)
			for ending, end := range map[string]error{"EOF": io.EOF, "unexpected EOF": io.ErrUnexpectedEOF} {
				h.storage.EXPECT().Download(mock.Anything, attachment.ObjectKey).Return(&kernelports.DownloadResult{
					Body: io.NopCloser(&failingAfter{data: sealed[:cut(sealed)], err: end}),
					Size: int64(len(sealed)),
				}, nil).Once()

				_, err := h.svc.attachmentContent(t.Context(), attachment)

				require.ErrorIs(t, err, encryptionservice.ErrBlobTruncated, ending)
				assert.False(t, errors.Is(err, ErrNonRetryableSend), "a cut-off download is retried: %s", ending)
			}
		})
	}
}

func TestAttachmentContentRefusesAFlippedCiphertextByteForGood(t *testing.T) {
	t.Parallel()

	h := newSealHarness(t)
	attachment, sealed := sealedAttachment(t, h, []byte("%PDF-1.7 proof of delivery"))
	sealed[len(sealed)/2] ^= 0x01
	h.stored(t, attachment.ObjectKey, sealed)

	_, err := h.svc.attachmentContent(t.Context(), attachment)

	require.ErrorIs(t, err, encryptionservice.ErrBlobRejected)
	assert.True(t, errors.Is(err, ErrNonRetryableSend))
}
