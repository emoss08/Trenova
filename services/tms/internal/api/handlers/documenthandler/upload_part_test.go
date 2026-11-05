package documenthandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/documenthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentuploadservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/contentscanportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentuploadportsmock"
	"github.com/emoss08/trenova/internal/testutil/objectstoretest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The browser never names the part path: it sends the bytes to whatever url and method the parts
// endpoint returned. Each strategy's target must therefore reach a route that stores the part.
func TestDocumentHandler_UploadPart_TheIssuedTargetStoresTheFile(t *testing.T) {
	t.Parallel()

	for _, strategy := range []documentupload.Strategy{
		documentupload.StrategySingle,
		documentupload.StrategyMultipart,
	} {
		t.Run(string(strategy), func(t *testing.T) {
			t.Parallel()

			content := []byte("%PDF-1.7 rate confirmation")
			session := &documentupload.DocumentUploadSession{
				ID:                pulid.MustNew("dus_"),
				OrganizationID:    testutil.TestOrgID,
				BusinessUnitID:    testutil.TestBuID,
				Strategy:          strategy,
				Status:            documentupload.StatusInitiated,
				FileSize:          int64(len(content)),
				OriginalName:      "rate-confirmation.pdf",
				ContentType:       "application/pdf",
				StoragePath:       "org/documents/rate-confirmation.pdf",
				ProcessingProfile: document.ProcessingProfileNone,
			}

			sessions := documentuploadportsmock.NewMockDocumentUploadSessionRepository(t)
			sessions.EXPECT().GetByID(mock.Anything, mock.Anything).Return(session, nil).Times(2)
			sessions.EXPECT().
				Update(mock.Anything, mock.Anything).
				RunAndReturn(func(
					_ context.Context,
					updated *documentupload.DocumentUploadSession,
				) (*documentupload.DocumentUploadSession, error) {
					return updated, nil
				}).
				Once()
			storage := objectstoretest.New()

			ginCtx := testutil.NewGinTestContext().WithDefaultAuthContext()
			uploadPartHandler(sessions, storage).RegisterRoutes(ginCtx.Engine.Group("/api/v1"))

			targets := httptest.NewRecorder()
			ginCtx.Engine.ServeHTTP(targets, httptest.NewRequestWithContext(
				t.Context(),
				http.MethodPost,
				"/api/v1/documents/uploads/"+session.ID.String()+"/parts/",
				bytes.NewReader([]byte(`{"partNumbers":[1]}`)),
			))
			require.Equal(t, http.StatusOK, targets.Code, targets.Body.String())

			var issued struct {
				Parts []struct {
					PartNumber int    `json:"partNumber"`
					URL        string `json:"url"`
					Method     string `json:"method"`
				} `json:"parts"`
			}
			require.NoError(t, json.Unmarshal(targets.Body.Bytes(), &issued))
			require.Len(t, issued.Parts, 1)

			upload := httptest.NewRecorder()
			ginCtx.Engine.ServeHTTP(upload, httptest.NewRequestWithContext(
				t.Context(),
				issued.Parts[0].Method,
				issued.Parts[0].URL,
				bytes.NewReader(content),
			))
			require.Equal(t, http.StatusOK, upload.Code, upload.Body.String())

			var uploaded documentupload.DocumentUploadSession
			require.NoError(t, json.Unmarshal(upload.Body.Bytes(), &uploaded))
			assert.Equal(t, session.ID, uploaded.ID)
			assert.Equal(t, documentupload.StatusUploaded, uploaded.Status)
			assert.Equal(t, []string{"org/documents/rate-confirmation.pdf"}, storage.Keys())
		})
	}
}

func uploadPartHandler(
	sessions *documentuploadportsmock.MockDocumentUploadSessionRepository,
	storage *objectstoretest.Store,
) *documenthandler.Handler {
	logger := zap.NewNop()
	storageConfig := &storageconfig.Config{}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &appconfig.Config{Debug: true},
	})

	return documenthandler.New(documenthandler.Params{
		UploadService: documentuploadservice.New(documentuploadservice.Params{
			Logger:        logger,
			SessionRepo:   sessions,
			Storage:       storage,
			StorageConfig: storageConfig,
			Encryption: encryptionservice.NewWithKeyManager(
				encryptionservice.NewLocalKeyManager(
					"unit-test-encryption-key-with-at-least-32-bytes",
				),
			),
			ContentGate: contentscanportsmock.NewCleanGate(),
		}),
		ErrorHandler: errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: &mocks.AllowAllPermissionEngine{},
				ErrorHandler:     errorHandler,
			},
		),
		StorageConfig: storageConfig,
		Logger:        logger,
	})
}

func TestDocumentHandler_GetUploadSession_AConversationUploadIsItsOwners(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		owner  pulid.ID
		status int
	}{
		"the person who started it": {owner: testutil.TestUserID, status: http.StatusOK},
		"anyone else":               {owner: pulid.MustNew("usr_"), status: http.StatusNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			session := &documentupload.DocumentUploadSession{
				ID:             pulid.MustNew("dus_"),
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				ResourceType:   document.ConversationResourceType,
				UploadedByID:   tc.owner,
				Strategy:       documentupload.StrategySingle,
				Status:         documentupload.StatusInitiated,
			}
			sessions := documentuploadportsmock.NewMockDocumentUploadSessionRepository(t)
			sessions.EXPECT().GetByID(mock.Anything, mock.Anything).Return(session, nil).Once()

			ginCtx := testutil.NewGinTestContext().WithDefaultAuthContext()
			uploadPartHandler(sessions, objectstoretest.New()).
				RegisterRoutes(ginCtx.Engine.Group("/api/v1"))

			recorder := httptest.NewRecorder()
			ginCtx.Engine.ServeHTTP(recorder, httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/api/v1/documents/uploads/"+session.ID.String()+"/",
				http.NoBody,
			))

			assert.Equal(t, tc.status, recorder.Code)
		})
	}
}
