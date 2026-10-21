package documentservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/contentscanportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentuploadportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	storageutil "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newScannedService(t *testing.T, gate contentscanports.Gate) *Service {
	t.Helper()

	storageCfg := &config.StorageConfig{
		AllowedMIMETypes:   []string{"application/pdf"},
		MaxFileSize:        50 * 1024 * 1024,
		PresignedURLExpiry: 15 * time.Minute,
	}
	svc := NewTestService(
		zap.NewNop(),
		documentportsmock.NewMockDocumentRepository(t),
		customerportsmock.NewMockDocumentCacheRepository(t),
		documentuploadportsmock.NewMockDocumentUploadSessionRepository(t),
		kernelportsmock.NewMockClient(t),
		NewValidator(ValidatorParams{Config: &config.Config{Storage: *storageCfg}}),
		&mocks.NoopAuditService{},
		storageCfg,
		thumbnailservice.NewGenerator(nil),
		nil,
	)
	svc.contentGate = gate
	return svc
}

func scannedTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func TestUpload_RefusesAnInfectedFileBeforeItIsStored(t *testing.T) {
	t.Parallel()

	gate := contentscanportsmock.NewInfectedGate("Win.Test.EICAR_HDB-1")
	svc := newScannedService(t, gate)
	content := []byte("%PDF-1.7 infected")

	result, err := svc.Upload(t.Context(), &documentports.UploadRequest{
		TenantInfo:   scannedTenant(),
		File:         storageutil.NewMockFileHeader("bol.pdf", content, "application/pdf"),
		ResourceID:   "res_123",
		ResourceType: "trailer",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errortypes.IsError(err))
	require.Len(t, gate.Requests(), 1)
	assert.Equal(t, contentscanports.SourceUpload, gate.Requests()[0].Source)
	assert.Equal(t, "bol.pdf", gate.Requests()[0].FileName)
	assert.Equal(t, content, gate.Requests()[0].Content)
}

func TestUpload_RefusesAFileTheScannerCannotCheck(t *testing.T) {
	t.Parallel()

	svc := newScannedService(
		t,
		contentscanportsmock.NewRefusingGate(contentscanports.ErrScannerUnavailable),
	)

	result, err := svc.Upload(t.Context(), &documentports.UploadRequest{
		TenantInfo: scannedTenant(),
		File: storageutil.NewMockFileHeader(
			"bol.pdf",
			[]byte("%PDF-1.7"),
			"application/pdf",
		),
		ResourceID:   "res_123",
		ResourceType: "trailer",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.ErrorIs(t, err, contentscanports.ErrScannerUnavailable)
}
