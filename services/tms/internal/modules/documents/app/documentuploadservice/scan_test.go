package documentuploadservice

import (
	"bytes"
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/testutil/mocks/contentscanportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentuploadportsmock"
	"github.com/emoss08/trenova/internal/testutil/objectstoretest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func uploadSession(
	tenant pagination.TenantInfo,
	content []byte,
) *documentupload.DocumentUploadSession {
	return &documentupload.DocumentUploadSession{
		ID:                pulid.MustNew("dus_"),
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		Status:            documentupload.StatusInitiated,
		FileSize:          int64(len(content)),
		OriginalName:      "remittance.pdf",
		StoragePath:       "org/documents/remittance.pdf",
		ProcessingProfile: document.ProcessingProfileInboundAttachment,
	}
}

func TestUploadPart_QuarantinesAnInfectedFileBeforeItIsStored(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	content := []byte("%PDF-1.7 infected")
	session := uploadSession(tenant, content)

	sessions := documentuploadportsmock.NewMockDocumentUploadSessionRepository(t)
	sessions.EXPECT().GetByID(mock.Anything, mock.Anything).Return(session, nil).Once()
	var saved *documentupload.DocumentUploadSession
	sessions.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			updated *documentupload.DocumentUploadSession,
		) (*documentupload.DocumentUploadSession, error) {
			saved = updated
			return updated, nil
		}).
		Once()
	storage := objectstoretest.New()
	gate := contentscanportsmock.NewInfectedGate("Win.Test.EICAR_HDB-1")

	svc := &Service{
		l:           zap.NewNop(),
		sessionRepo: sessions,
		storage:     storage,
		encryption:  scanTestEncryption(),
		contentGate: gate,
	}
	_, err := svc.UploadPart(t.Context(), &documentuploadports.UploadPartRequest{
		TenantInfo: tenant,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(content),
		Size:       int64(len(content)),
	})

	require.Error(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, documentupload.StatusQuarantined, saved.Status)
	assert.Equal(t, documentupload.FailureMalwareDetected.String(), saved.FailureCode)
	assert.Equal(t, "Win.Test.EICAR_HDB-1", saved.ThreatName)
	require.Len(t, gate.Requests(), 1)
	assert.Equal(t, contentscanports.SourceInboundAttachment, gate.Requests()[0].Source)
	assert.Equal(t, content, gate.Requests()[0].Content)
	assert.Empty(t, storage.Keys(), "the infected file never became an object")
}

func TestUploadPart_LeavesTheSessionOpenWhenTheScannerCannotCheck(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	content := []byte("%PDF-1.7")
	session := uploadSession(tenant, content)

	sessions := documentuploadportsmock.NewMockDocumentUploadSessionRepository(t)
	sessions.EXPECT().GetByID(mock.Anything, mock.Anything).Return(session, nil).Once()
	storage := objectstoretest.New()
	svc := &Service{
		l:           zap.NewNop(),
		sessionRepo: sessions,
		storage:     storage,
		encryption:  scanTestEncryption(),
		contentGate: contentscanportsmock.NewRefusingGate(contentscanports.ErrScannerUnavailable),
	}

	_, err := svc.UploadPart(t.Context(), &documentuploadports.UploadPartRequest{
		TenantInfo: tenant,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(content),
		Size:       int64(len(content)),
	})

	require.ErrorIs(t, err, contentscanports.ErrScannerUnavailable)
	assert.Equal(t, documentupload.StatusInitiated, session.Status)
	assert.Empty(t, storage.Keys())
}

func scanTestEncryption() *encryptionservice.Service {
	return encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("upload-scan-test-key-material"),
	)
}

func TestUploadPart_FailsTheSessionOfAFileTooLargeToScan(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	content := []byte("%PDF-1.7 large")
	session := uploadSession(tenant, content)

	sessions := documentuploadportsmock.NewMockDocumentUploadSessionRepository(t)
	sessions.EXPECT().GetByID(mock.Anything, mock.Anything).Return(session, nil).Once()
	var saved *documentupload.DocumentUploadSession
	sessions.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			updated *documentupload.DocumentUploadSession,
		) (*documentupload.DocumentUploadSession, error) {
			saved = updated
			return updated, nil
		}).
		Once()
	storage := objectstoretest.New()
	svc := &Service{
		l:           zap.NewNop(),
		sessionRepo: sessions,
		storage:     storage,
		encryption:  scanTestEncryption(),
		contentGate: contentscanportsmock.NewRefusingGate(
			&contentscanports.TooLargeError{Size: int64(len(content)), Limit: 4},
		),
	}

	_, err := svc.UploadPart(t.Context(), &documentuploadports.UploadPartRequest{
		TenantInfo: tenant,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(content),
		Size:       int64(len(content)),
	})

	require.Error(t, err)
	assert.NotErrorIs(t, err, contentscanports.ErrScannerUnavailable)
	require.NotNil(t, saved)
	assert.Equal(t, documentupload.StatusFailed, saved.Status)
	assert.Equal(t, documentupload.FailureTooLargeToScan.String(), saved.FailureCode)
	assert.Empty(t, saved.ThreatName)
	assert.Empty(t, storage.Keys())
}

func TestCreateSession_RecordsWhetherTheUploadWillBeScanned(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		gate contentscanports.Gate
		want bool
	}{
		{name: "scanner configured", gate: contentscanportsmock.NewCleanGate(), want: true},
		{name: "scanner off", gate: contentscanportsmock.NewDisabledGate(), want: false},
		{name: "no gate", gate: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tenant := pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			}
			sessions := documentuploadportsmock.NewMockDocumentUploadSessionRepository(t)
			sessions.EXPECT().
				Create(mock.Anything, mock.Anything).
				RunAndReturn(func(
					_ context.Context,
					created *documentupload.DocumentUploadSession,
				) (*documentupload.DocumentUploadSession, error) {
					return created, nil
				}).
				Once()
			svc := &Service{
				l:           zap.NewNop(),
				sessionRepo: sessions,
				validator: documentservice.NewValidator(documentservice.ValidatorParams{
					Config: &config.Config{
						Storage: config.StorageConfig{
							AllowedMIMETypes: []string{"application/pdf"},
							MaxFileSize:      1024 * 1024,
						},
					},
				}),
				contentGate: tt.gate,
			}

			session, err := svc.CreateSession(t.Context(), &documentuploadports.CreateSessionRequest{
				TenantInfo:   tenant,
				ResourceID:   pulid.MustNew("shp_").String(),
				ResourceType: "shipment",
				FileName:     "bol.pdf",
				FileSize:     2048,
				ContentType:  "application/pdf",
			})

			require.NoError(t, err)
			assert.Equal(t, tt.want, session.MalwareScan)
		})
	}
}

// A finished session's storage path is the object its document reads. A
// cancel, which needs only document create, must not delete it.
func TestCancel_LeavesTheFileOfAFinishedUploadAlone(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	documentID := pulid.MustNew("doc_")
	for _, finished := range []*documentupload.DocumentUploadSession{
		{Status: documentupload.StatusAvailable, DocumentID: &documentID},
		{Status: documentupload.StatusCompleted},
		{Status: documentupload.StatusFinalizing},
		{Status: documentupload.StatusQuarantined},
	} {
		session := uploadSession(tenant, []byte("%PDF-1.7 pod"))
		session.Status = finished.Status
		session.DocumentID = finished.DocumentID

		sessions := documentuploadportsmock.NewMockDocumentUploadSessionRepository(t)
		sessions.EXPECT().GetByID(mock.Anything, mock.Anything).Return(session, nil).Once()
		storage := objectstoretest.New()
		storage.Put(session.StoragePath, []byte("%PDF-1.7 pod"))

		svc := &Service{l: zap.NewNop(), sessionRepo: sessions, storage: storage}
		err := svc.Cancel(t.Context(), &documentuploadports.CancelRequest{
			TenantInfo: tenant,
			SessionID:  session.ID,
		})

		require.Error(t, err, string(finished.Status))
		_, kept := storage.Object(session.StoragePath)
		assert.True(t, kept, "the %s session's document still reads its file", finished.Status)
	}
}
