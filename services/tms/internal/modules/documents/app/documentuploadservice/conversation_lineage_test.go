package documentuploadservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentuploadportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type lineageVersions struct {
	documentports.DocumentRepository

	versions []*document.Document
}

func (l *lineageVersions) ListVersions(
	context.Context,
	documentports.ListDocumentVersionsRequest,
) ([]*document.Document, error) {
	return l.versions, nil
}

func TestCreateSession_RefusesANewVersionOfAnotherPersonsConversationFile(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	thread := pulid.MustNew("athr_").String()
	svc := &Service{
		l:           zap.NewNop(),
		sessionRepo: documentuploadportsmock.NewMockDocumentUploadSessionRepository(t),
		documentRepo: &lineageVersions{versions: []*document.Document{{
			ResourceType: document.ConversationResourceType,
			ResourceID:   thread,
			UploadedByID: pulid.MustNew("usr_"),
		}}},
		validator: documentservice.NewValidator(documentservice.ValidatorParams{
			StorageConfig: &storageconfig.Config{
				AllowedMIMETypes: []string{"application/pdf"},
				MaxFileSize:      1024 * 1024,
			},
		}),
	}

	_, err := svc.CreateSession(t.Context(), &documentuploadports.CreateSessionRequest{
		TenantInfo:   tenant,
		Actor:        *permissionports.UserActor(tenant),
		ResourceID:   thread,
		ResourceType: document.ConversationResourceType,
		FileName:     "rate-confirmation.pdf",
		FileSize:     2048,
		ContentType:  "application/pdf",
		LineageID:    pulid.MustNew("doc_").String(),
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), err.Error())
}
