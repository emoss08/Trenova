package documentresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentimportassistantports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contentService struct {
	searched   []string
	reextract  []pulid.ID
	reextracts error
	tenant     pagination.TenantInfo
}

func (s *contentService) GetContent(
	_ context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*documentcontent.Content, error) {
	s.tenant = tenantInfo
	return &documentcontent.Content{DocumentID: documentID, Status: "Indexed"}, nil
}

func (s *contentService) GetShipmentDraft(
	_ context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	s.tenant = tenantInfo
	return &documentshipmentdraft.DocumentShipmentDraft{DocumentID: documentID}, nil
}

func (s *contentService) SearchDocuments(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType, resourceID, query string,
) ([]*document.Document, error) {
	s.tenant = tenantInfo
	s.searched = []string{resourceType, resourceID, query}
	return []*document.Document{{ID: pulid.MustNew("doc_"), ResourceID: resourceID}}, nil
}

func (s *contentService) Reextract(
	_ context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	s.tenant = tenantInfo
	s.reextract = append(s.reextract, documentID)
	return s.reextracts
}

func (s *contentService) EnqueueExtraction(context.Context, *document.Document, pulid.ID) error {
	return nil
}

type importAssistant struct {
	archived  []string
	completed []string
	history   *shipmentimportassistantports.ShipmentImportChatHistoryResponse
}

func (a *importAssistant) GetHistory(
	_ context.Context,
	documentID string,
	_ pagination.TenantInfo,
) (*shipmentimportassistantports.ShipmentImportChatHistoryResponse, error) {
	if a.history != nil {
		return a.history, nil
	}
	return &shipmentimportassistantports.ShipmentImportChatHistoryResponse{
		DocumentID: documentID,
	}, nil
}

func (a *importAssistant) ArchiveHistory(
	_ context.Context,
	documentID string,
	_ pagination.TenantInfo,
) error {
	a.archived = append(a.archived, documentID)
	return nil
}

func (a *importAssistant) CompleteHistory(
	_ context.Context,
	documentID string,
	_ pagination.TenantInfo,
) error {
	a.completed = append(a.completed, documentID)
	return nil
}

type pageAssistant struct {
	request *assistantports.OpenPageThreadRequest
	actor   *permissionports.RequestActor
}

func (p *pageAssistant) OpenPageThread(
	_ context.Context,
	req *assistantports.OpenPageThreadRequest,
	actor *permissionports.RequestActor,
) (*assistantports.PageThread, error) {
	p.request = req
	p.actor = actor
	return &assistantports.PageThread{
		Thread: &conversation.Thread{ID: pulid.MustNew("athr_"), SubjectID: req.SubjectID},
		Agent:  assistantports.PageAgent{Name: "Import assistant"},
	}, nil
}

type documentReader struct {
	reading map[string]any
}

func (r *documentReader) ReadDocument(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (map[string]any, error) {
	return r.reading, nil
}

func allowAll() *resolvercore.Core {
	return &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}
}

func TestDocumentResolvers_CheckTheDocumentPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("doc_").String()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"document", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).Document(ctx, id)
			return err
		}},
		{"documentVersions", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DocumentVersions(ctx, id)
			return err
		}},
		{"documentsByResource", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DocumentsByResource(
				ctx,
				gqlmodel.DocumentsByResourceInput{ResourceType: "shipment", ResourceID: "shp_1"},
			)
			return err
		}},
		{"documentPacketSummary", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DocumentPacketSummary(ctx, "trailer", "tr_1")
			return err
		}},
		{"documentContent", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DocumentContent(ctx, id)
			return err
		}},
		{"documentShipmentDraft", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DocumentShipmentDraft(ctx, id)
			return err
		}},
		{"documentReading", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DocumentReading(ctx, id)
			return err
		}},
		{
			"documentExtractionStatus",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).DocumentExtractionStatus(ctx)
				return err
			},
		},
		{
			"documentImportAssistantHistory",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).DocumentImportAssistantHistory(ctx, id)
				return err
			},
		},
		{"deleteDocument", permission.OpDelete, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).DeleteDocument(ctx, id)
			return err
		}},
		{"bulkDeleteDocuments", permission.OpDelete, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).BulkDeleteDocuments(ctx, []string{id})
			return err
		}},
		{
			"restoreDocumentVersion",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).RestoreDocumentVersion(ctx, id)
				return err
			},
		},
		{"approveDocument", permission.OpApprove, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ApproveDocument(ctx, id)
			return err
		}},
		{"rejectDocument", permission.OpReject, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).RejectDocument(
				ctx,
				gqlmodel.RejectDocumentInput{DocumentID: id, Reason: "Blurry"},
			)
			return err
		}},
		{"setDocumentSignature", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SetDocumentSignature(
				ctx,
				gqlmodel.SetDocumentSignatureInput{
					DocumentID: id,
					Status:     document.SignatureStatusSigned,
				},
			)
			return err
		}},
		{
			"setDocumentDeliveryFacts",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).SetDocumentDeliveryFacts(
					ctx,
					gqlmodel.SetDocumentDeliveryFactsInput{
						DocumentID: id,
						Facts:      &gqlmodel.DocumentDeliveryFactsInput{},
					},
				)
				return err
			},
		},
		{
			"attachDocumentToShipment",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).AttachDocumentToShipment(
					ctx,
					gqlmodel.AttachDocumentToShipmentInput{
						DocumentID: id,
						ShipmentID: pulid.MustNew("shp_").String(),
					},
				)
				return err
			},
		},
		{"reextractDocument", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ReextractDocument(ctx, id)
			return err
		}},
		{
			"openDocumentImportAssistantThread",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).OpenDocumentImportAssistantThread(ctx, id)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceDocument,
				call.operation,
			)
		})
	}
}

func TestOpenDocumentImportAssistantThread_NeedsTheAssistant(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
		permission.ResourceDocument.String() + "|" + string(permission.OpRead): true,
	}}
	assistant := &pageAssistant{}
	deps := &Deps{
		Core:          &resolvercore.Core{PermissionEngine: engine},
		PageAssistant: assistant,
	}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&MutationResolver{deps}).OpenDocumentImportAssistantThread(
		ctx,
		pulid.MustNew("doc_").String(),
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
	require.Len(t, engine.Requests, 2)
	assert.Equal(t, permission.ResourceAssistant.String(), engine.Requests[1].Resource)
	assert.Equal(t, permission.OpCreate, engine.Requests[1].Operation)
	assert.Nil(t, assistant.request)
}

func TestOpenDocumentImportAssistantThread_OpensTheDocumentsConversation(t *testing.T) {
	t.Parallel()

	assistant := &pageAssistant{}
	deps := &Deps{Core: allowAll(), PageAssistant: assistant}
	authCtx := resolvertest.UserAuthContext()
	documentID := pulid.MustNew("doc_")

	opened, err := (&MutationResolver{deps}).OpenDocumentImportAssistantThread(
		resolvertest.AuthedContext(t, authCtx),
		documentID.String(),
	)

	require.NoError(t, err)
	thread, ok := opened.(*assistantports.PageThread)
	require.True(t, ok)
	assert.Equal(t, documentID, thread.Thread.SubjectID)
	assert.Equal(t, conversation.ThreadOriginImport, assistant.request.Origin)
	assert.Equal(t, agent.SubjectDocument, assistant.request.SubjectType)
	assert.Equal(t, documentID, assistant.request.SubjectID)
	assert.Equal(t, authCtx.OrganizationID, assistant.request.TenantInfo.OrgID)
	assert.Equal(t, authCtx.UserID, assistant.request.TenantInfo.UserID)
	assert.Equal(t, authCtx.UserID, assistant.actor.UserID)
}

func TestOpenDocumentImportAssistantThread_RefusesAnIDThatIsNotOne(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: allowAll(), PageAssistant: &pageAssistant{}}

	_, err := (&MutationResolver{deps}).OpenDocumentImportAssistantThread(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		"not-an-id",
	)

	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr), "%v", err)
	assert.Equal(t, "documentId", validationErr.Field)
}

func TestDocumentsByResource_SearchesTheExtractedTextWhenAsked(t *testing.T) {
	t.Parallel()

	content := &contentService{}
	deps := &Deps{Core: allowAll(), DocumentContentService: content}
	authCtx := resolvertest.UserAuthContext()
	query := "  bill of lading "

	found, err := (&QueryResolver{deps}).DocumentsByResource(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.DocumentsByResourceInput{
			ResourceType: "shipment",
			ResourceID:   "shp_1",
			Query:        &query,
		},
	)

	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, []string{"shipment", "shp_1", "bill of lading"}, content.searched)
	assert.Equal(t, authCtx.OrganizationID, content.tenant.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, content.tenant.BuID)
}

func TestReextractDocument_ArchivesTheImportConversation(t *testing.T) {
	t.Parallel()

	content := &contentService{}
	assistant := &importAssistant{}
	deps := &Deps{
		Core:                    allowAll(),
		DocumentContentService:  content,
		ShipmentImportAssistant: assistant,
	}
	documentID := pulid.MustNew("doc_")
	authCtx := resolvertest.UserAuthContext()

	ok, err := (&MutationResolver{deps}).ReextractDocument(
		resolvertest.AuthedContext(t, authCtx),
		documentID.String(),
	)

	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []pulid.ID{documentID}, content.reextract)
	assert.Equal(t, authCtx.UserID, content.tenant.UserID)
	assert.Equal(t, []string{documentID.String()}, assistant.archived)
}

func TestReextractDocument_KeepsTheConversationWhenExtractionFails(t *testing.T) {
	t.Parallel()

	content := &contentService{reextracts: errortypes.NewBusinessError("no converter")}
	assistant := &importAssistant{}
	deps := &Deps{
		Core:                    allowAll(),
		DocumentContentService:  content,
		ShipmentImportAssistant: assistant,
	}

	_, err := (&MutationResolver{deps}).ReextractDocument(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("doc_").String(),
	)

	require.Error(t, err)
	assert.Empty(t, assistant.archived)
}

func TestDocumentReading_ReturnsTheExtractionCard(t *testing.T) {
	t.Parallel()

	deps := &Deps{
		Core:           allowAll(),
		DocumentReader: &documentReader{reading: map[string]any{"fields": []any{}}},
	}

	reading, err := (&QueryResolver{deps}).DocumentReading(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("doc_").String(),
	)

	require.NoError(t, err)
	assert.Equal(t, map[string]any{"fields": []any{}}, reading)
}

func TestDocumentReading_ADocumentNothingWasReadFromIsEmpty(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: allowAll(), DocumentReader: &documentReader{}}

	reading, err := (&QueryResolver{deps}).DocumentReading(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("doc_").String(),
	)

	require.NoError(t, err)
	assert.Equal(t, map[string]any{}, reading)
}

func TestDocumentReading_SaysSoWithoutAReader(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: allowAll()}

	_, err := (&QueryResolver{deps}).DocumentReading(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("doc_").String(),
	)

	var businessErr *errortypes.BusinessError
	require.True(t, errors.As(err, &businessErr), "%v", err)
}

func TestDocumentImportAssistantHistory_ReadsTheDocumentsConversation(t *testing.T) {
	t.Parallel()

	assistant := &importAssistant{}
	deps := &Deps{Core: allowAll(), ShipmentImportAssistant: assistant}
	documentID := pulid.MustNew("doc_")

	history, err := (&QueryResolver{deps}).DocumentImportAssistantHistory(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		documentID.String(),
	)

	require.NoError(t, err)
	assert.Equal(t, documentID.String(), history.DocumentID)
}

func TestDocumentContent_ReadsInTheCallersTenant(t *testing.T) {
	t.Parallel()

	content := &contentService{}
	deps := &Deps{Core: allowAll(), DocumentContentService: content}
	authCtx := resolvertest.UserAuthContext()
	documentID := pulid.MustNew("doc_")
	ctx := resolvertest.AuthedContext(t, authCtx)

	read, err := (&QueryResolver{deps}).DocumentContent(ctx, documentID.String())
	require.NoError(t, err)
	assert.Equal(t, documentID, read.DocumentID)

	status, err := (&DocumentContentResolver{deps}).Status(ctx, read)
	require.NoError(t, err)
	assert.Equal(t, "Indexed", status)
	assert.Equal(t, authCtx.OrganizationID, content.tenant.OrgID)
	assert.True(t, content.tenant.UserID.IsNil())
}

func TestDeliveryFactsRequest_MapsTheFactsAPersonRecords(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	documentID := pulid.MustNew("doc_")
	chargeID := pulid.MustNew("acc_")
	signedBy := "J. Smith"
	amount := "125.50"
	charge := chargeID.String()
	arrived, departed := 1_000, 2_000
	osd := document.OSDShort

	request, err := deliveryFactsRequest(authCtx, &gqlmodel.SetDocumentDeliveryFactsInput{
		DocumentID: documentID.String(),
		Facts: &gqlmodel.DocumentDeliveryFactsInput{
			SignedBy:   &signedBy,
			Osd:        &osd,
			ArrivedAt:  &arrived,
			DepartedAt: &departed,
			Amount:     &amount,
			ChargeID:   &charge,
		},
		Version: 4,
	})

	require.NoError(t, err)
	assert.Equal(t, documentID, request.DocumentID)
	assert.Equal(t, int64(4), request.Version)
	assert.Equal(t, authCtx.UserID, request.Actor.UserID)
	assert.Equal(t, authCtx.UserID, request.TenantInfo.UserID)
	assert.Equal(t, "J. Smith", request.Facts.SignedBy)
	assert.Equal(t, document.OSDShort, request.Facts.OSD)
	assert.Equal(t, int64(1_000), *request.Facts.ArrivedAt)
	assert.Equal(t, int64(2_000), *request.Facts.DepartedAt)
	assert.True(t, request.Facts.Amount.Valid)
	assert.True(t, decimal.RequireFromString("125.50").Equal(request.Facts.Amount.Decimal))
	assert.Equal(t, chargeID, request.Facts.ChargeID)
}

func TestDeliveryFactsRequest_RefusesAnAmountThatIsNotANumber(t *testing.T) {
	t.Parallel()

	amount := "lots"
	_, err := deliveryFactsRequest(
		resolvertest.UserAuthContext(),
		&gqlmodel.SetDocumentDeliveryFactsInput{
			DocumentID: pulid.MustNew("doc_").String(),
			Facts:      &gqlmodel.DocumentDeliveryFactsInput{Amount: &amount},
		},
	)

	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr), "%v", err)
	assert.Equal(t, "facts.amount", validationErr.Field)
}

func TestSignatureRequest_MapsTheReading(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	documentID := pulid.MustNew("doc_")
	signedAt := 1_700_000_000

	request, err := signatureRequest(authCtx, &gqlmodel.SetDocumentSignatureInput{
		DocumentID: documentID.String(),
		Status:     document.SignatureStatusSigned,
		SignedAt:   &signedAt,
		Version:    2,
	})

	require.NoError(t, err)
	assert.Equal(t, documentID, request.DocumentID)
	assert.Equal(t, document.SignatureStatusSigned, request.Status)
	assert.Equal(t, int64(1_700_000_000), *request.SignedAt)
	assert.Equal(t, int64(2), request.Version)
	assert.Equal(t, authCtx.UserID, request.Actor.UserID)
}

func TestReviewRequest_CarriesTheDecisionAndReason(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	documentID := pulid.MustNew("doc_")

	request, err := reviewRequest(
		authCtx,
		documentID.String(),
		document.ReviewDecisionReject,
		"Unreadable",
	)

	require.NoError(t, err)
	assert.Equal(t, documentID, request.DocumentID)
	assert.Equal(t, document.ReviewDecisionReject, request.Decision)
	assert.Equal(t, "Unreadable", request.Reason)
	assert.Equal(t, authCtx.UserID, request.TenantInfo.UserID)
}

func TestBulkDeleteIDs_NeedsAtLeastOneDocument(t *testing.T) {
	t.Parallel()

	_, err := bulkDeleteIDs(nil)

	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr), "%v", err)
	assert.Equal(t, "ids", validationErr.Field)
}

func TestDocumentDeliveryFacts_AmountIsTheExactDecimal(t *testing.T) {
	t.Parallel()

	resolver := &DocumentDeliveryFactsResolver{&Deps{Core: allowAll()}}

	amount, err := resolver.Amount(t.Context(), &document.DeliveryFacts{
		Amount: decimal.NewNullDecimal(decimal.RequireFromString("12.345")),
	})
	require.NoError(t, err)
	require.NotNil(t, amount)
	assert.Equal(t, "12.345", *amount)

	none, err := resolver.Amount(t.Context(), &document.DeliveryFacts{})
	require.NoError(t, err)
	assert.Nil(t, none)
}
