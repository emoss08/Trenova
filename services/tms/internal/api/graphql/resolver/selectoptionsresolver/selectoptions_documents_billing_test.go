package selectoptionsresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptbatchports"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/app/documenttypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/testutil/mocks/bankreceiptbatchportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttypeportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func documentResolver(
	repo documentports.DocumentRepository,
	engine *decidingPermissionEngine,
) *QueryResolver {
	return &QueryResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: engine},
		DocumentService: documentservice.NewTestService(
			zap.NewNop(), repo, nil, nil, nil, nil, nil, nil, nil, nil,
		),
	}}
}

func TestSelectOptions_DocumentShowsOnlyItsReferenceView(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")
	repo := documentportsmock.NewMockDocumentRepository(t)
	repo.EXPECT().
		SelectOptions(mock.Anything, mock.MatchedBy(func(req *documentports.DocumentSelectOptionsRequest) bool {
			return req.ResourceID == "shp_1" &&
				req.ResourceType == "shipment" &&
				len(req.IDs) == 0 &&
				req.SelectQueryRequest.Query == "bol"
		})).
		Return(&pagination.ListResult[*document.Document]{
			Items: []*document.Document{{
				ID:           docID,
				OriginalName: "BOL 4471.pdf",
				FileName:     "a1b2c3.pdf",
				FileType:     "application/pdf",
				FileSize:     2048,
				Status:       document.StatusActive,
				ResourceID:   "shp_1",
				ResourceType: "shipment",
				Description:  "signed by the receiver",
				CreatedAt:    1780415883,
				DocumentType: &documenttype.DocumentType{Name: "Bill of lading"},
			}},
			Total:      1,
			TotalExact: true,
		}, nil).
		Once()
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	query := "bol"
	result, err := documentResolver(
		repo,
		&decidingPermissionEngine{allowed: true},
	).SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceDocument,
		Query:    &query,
		Filters:  map[string]any{"resourceId": "shp_1", "resourceType": "shipment"},
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	node := result.Edges[0].Node
	assert.Equal(t, docID.String(), node.ID)
	assert.Equal(t, "BOL 4471.pdf", node.Label)
	require.NotNil(t, node.Description)
	assert.Equal(t, "Bill of lading", *node.Description)
	assert.Equal(
		t,
		map[string]any{"fileType": "application/pdf", "fileSize": int64(2048)},
		node.Meta,
	)
}

func TestSelectOptions_DocumentLooksUpSelectedIDsInOneReadInTheirOrder(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("doc_")
	second := pulid.MustNew("doc_")
	gone := pulid.MustNew("doc_")
	repo := documentportsmock.NewMockDocumentRepository(t)
	repo.EXPECT().
		SelectOptions(mock.Anything, mock.MatchedBy(func(req *documentports.DocumentSelectOptionsRequest) bool {
			return assert.ObjectsAreEqual([]pulid.ID{second, first, gone}, req.IDs) &&
				req.SelectQueryRequest.Query == "" &&
				req.SelectQueryRequest.Pagination.Offset == 0
		})).
		Return(&pagination.ListResult[*document.Document]{
			Items: []*document.Document{
				{ID: first, OriginalName: "first.pdf", CreatedAt: 1},
				{ID: second, OriginalName: "second.pdf", CreatedAt: 2},
			},
			Total:      2,
			TotalExact: true,
		}, nil).
		Once()
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	result, err := documentResolver(
		repo,
		&decidingPermissionEngine{allowed: true},
	).SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceDocument,
		Ids:      []string{second.String(), first.String(), gone.String()},
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 2)
	assert.Equal(t, "second.pdf", result.Edges[0].Node.Label)
	assert.Equal(t, "first.pdf", result.Edges[1].Node.Label)
}

func TestSelectOptions_DocumentRefusesSomeoneWhoCannotReadDocuments(t *testing.T) {
	t.Parallel()

	repo := documentportsmock.NewMockDocumentRepository(t)
	engine := &decidingPermissionEngine{allowed: false}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	_, err := documentResolver(repo, engine).SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceDocument,
	})

	require.Error(t, err)
	require.Len(t, engine.checked, 1)
	assert.Equal(t, permission.ResourceDocument.String(), engine.checked[0].Resource)
}

func TestSelectOptions_DocumentTypeKeepsToTheAskedCategory(t *testing.T) {
	t.Parallel()

	repo := documenttypeportsmock.NewMockDocumentTypeRepository(t)
	repo.EXPECT().
		SelectOptions(mock.Anything, mock.MatchedBy(func(req *documenttypeports.DocumentTypeSelectOptionsRequest) bool {
			return req.Category == documenttype.DocumentCategory("Shipment") &&
				req.SelectQuery != nil
		})).
		Return(&pagination.ListResult[*documenttype.DocumentType]{}, nil).
		Once()
	resolver := &QueryResolver{&Deps{
		Core: &resolvercore.Core{},
		DocumentTypeService: documenttypeservice.New(documenttypeservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	_, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceDocumentType,
		Filters:  map[string]any{"documentCategory": "Shipment"},
	})

	require.NoError(t, err)
}

func bankReceiptResolver(
	svc bankreceiptbatchports.BankReceiptBatchService,
	engine *decidingPermissionEngine,
) *QueryResolver {
	return &QueryResolver{&Deps{
		Core:                    &resolvercore.Core{PermissionEngine: engine},
		BankReceiptBatchService: svc,
	}}
}

func TestSelectOptions_BankReceiptSourceNeedsBankReceiptRead(t *testing.T) {
	t.Parallel()

	svc := bankreceiptbatchportsmock.NewMockBankReceiptBatchService(t)
	svc.EXPECT().
		DistinctSources(mock.Anything, mock.MatchedBy(func(req *pagination.SelectQueryRequest) bool {
			return req.Query == "ch"
		})).
		Return(&pagination.ListResult[*bankreceiptbatchports.BankReceiptBatchSourceOption]{
			Items: []*bankreceiptbatchports.BankReceiptBatchSourceOption{
				{Value: "Chase", Label: "Chase"},
			},
			Total:      1,
			TotalExact: true,
		}, nil).
		Once()
	engine := &decidingPermissionEngine{allowed: true}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	query := "ch"
	result, err := bankReceiptResolver(svc, engine).SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceBankReceiptSource,
		Query:    &query,
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	assert.Equal(t, "Chase", result.Edges[0].Node.ID)
	assert.Equal(t, "Chase", result.Edges[0].Node.Label)
	require.Len(t, engine.checked, 1)
	assert.Equal(t, permission.ResourceBankReceipt.String(), engine.checked[0].Resource)
	assert.Equal(t, permission.OpRead, engine.checked[0].Operation)
}

func TestSelectOptions_BankReceiptSourceNamesASavedSourceWithoutReading(t *testing.T) {
	t.Parallel()

	svc := bankreceiptbatchportsmock.NewMockBankReceiptBatchService(t)
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	result, err := bankReceiptResolver(svc, &decidingPermissionEngine{allowed: true}).SelectOptions(
		ctx,
		gqlmodel.SelectOptionsInput{
			Resource: gqlmodel.SelectOptionResourceBankReceiptSource,
			Ids:      []string{"Wells Fargo"},
		},
	)
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	assert.Equal(t, "Wells Fargo", result.Edges[0].Node.ID)
	assert.Equal(t, "Wells Fargo", result.Edges[0].Node.Label)
}

func TestSelectOptions_BankReceiptSourceRefusesSomeoneWhoCannotReadReceipts(t *testing.T) {
	t.Parallel()

	svc := bankreceiptbatchportsmock.NewMockBankReceiptBatchService(t)
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	_, err := bankReceiptResolver(svc, &decidingPermissionEngine{allowed: false}).SelectOptions(
		ctx,
		gqlmodel.SelectOptionsInput{Resource: gqlmodel.SelectOptionResourceBankReceiptSource},
	)

	require.Error(t, err)
}
