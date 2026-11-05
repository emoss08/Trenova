package documentservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type oneDocument struct {
	documentports.DocumentRepository

	doc *document.Document
}

func (o *oneDocument) GetByID(
	context.Context,
	documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return o.doc, nil
}

func (o *oneDocument) GetByIDs(
	context.Context,
	documentports.GetDocumentsByIDsRequest,
) ([]*document.Document, error) {
	return []*document.Document{o.doc}, nil
}

func (o *oneDocument) ListVersions(
	context.Context,
	documentports.ListDocumentVersionsRequest,
) ([]*document.Document, error) {
	return []*document.Document{o.doc}, nil
}

func TestEveryPathThatNamesADocument_RefusesAnotherPersonsConversationFile(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		LineageID:    pulid.MustNew("doc_"),
		ResourceType: document.ConversationResourceType,
		ResourceID:   pulid.MustNew("athr_").String(),
		UploadedByID: pulid.MustNew("usr_"),
	}
	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	actor := permissionports.UserActor(tenant)
	service := documentservice.New(documentservice.Params{
		Logger:      zap.NewNop(),
		Repo:        &oneDocument{doc: doc},
		Permissions: &allowEveryRead{},
	})
	attach := &documentservice.AttachLineageRequest{
		DocumentID:   doc.ID,
		ResourceType: "shipment",
		ResourceID:   pulid.MustNew("shp_").String(),
		TenantInfo:   tenant,
		Actor:        actor,
	}

	for name, call := range map[string]func(context.Context) error{
		"list versions": func(ctx context.Context) error {
			_, err := service.ListVersions(ctx, doc.ID, tenant, actor)
			return err
		},
		"plan restore": func(ctx context.Context) error {
			_, err := service.PlanRestoreVersion(ctx, doc.ID, tenant, actor)
			return err
		},
		"restore": func(ctx context.Context) error {
			_, err := service.RestoreVersion(ctx, doc.ID, tenant, actor)
			return err
		},
		"preview attach": func(ctx context.Context) error {
			_, err := service.PreviewAttachLineage(ctx, attach)
			return err
		},
		"attach": func(ctx context.Context) error {
			_, err := service.AttachLineageToResource(ctx, attach)
			return err
		},
		"delete": func(ctx context.Context) error {
			return service.Delete(ctx, documentports.DeleteDocumentRequest{
				ID:         doc.ID,
				TenantInfo: tenant,
			}, actor)
		},
		"plan delete": func(ctx context.Context) error {
			_, err := service.PlanDelete(ctx, &documentservice.BulkDeleteRequest{
				IDs:        []pulid.ID{doc.ID},
				TenantInfo: tenant,
				Actor:      actor,
			})
			return err
		},
		"review": func(ctx context.Context) error {
			_, err := service.PlanReview(ctx, &documentservice.ReviewRequest{
				DocumentID: doc.ID,
				TenantInfo: tenant,
				Decision:   document.ReviewDecisionApprove,
				Actor:      *actor,
			})
			return err
		},
		"signature": func(ctx context.Context) error {
			_, err := service.SetSignature(ctx, &documentservice.SetSignatureRequest{
				DocumentID: doc.ID,
				TenantInfo: tenant,
				Status:     document.SignatureStatusSigned,
				Actor:      *actor,
			})
			return err
		},
		"delivery facts": func(ctx context.Context) error {
			_, err := service.SetDeliveryFacts(ctx, &documentservice.SetDeliveryFactsRequest{
				DocumentID: doc.ID,
				TenantInfo: tenant,
				Actor:      *actor,
			})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := call(t.Context())

			require.Error(t, err)
			assert.True(t, errortypes.IsNotFoundError(err), err.Error())
		})
	}
}

type allowEveryRead struct {
	permissionports.PermissionEngine
}

func (*allowEveryRead) Check(
	context.Context,
	*permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{Allowed: true}, nil
}

func TestAttach_AnAgentRunWithoutAPersonCannotMoveAConversationFile(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		ResourceType: document.ConversationResourceType,
		UploadedByID: userID,
	}
	service := documentservice.New(documentservice.Params{
		Logger: zap.NewNop(),
		Repo:   &oneDocument{doc: doc},
	})

	_, err := service.PreviewAttachLineage(t.Context(), &documentservice.AttachLineageRequest{
		DocumentID:   doc.ID,
		ResourceType: "shipment",
		ResourceID:   pulid.MustNew("shp_").String(),
		Actor: &permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeAgent,
			UserID:        userID,
		},
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}
