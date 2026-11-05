package accesstest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/documenthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type documents struct {
	documentports.DocumentRepository

	doc     *document.Document
	viewers []pulid.ID
	deleted []pulid.ID
}

func (d *documents) GetByID(
	context.Context,
	documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return d.doc, nil
}

func (d *documents) GetByIDs(
	context.Context,
	documentports.GetDocumentsByIDsRequest,
) ([]*document.Document, error) {
	return []*document.Document{d.doc}, nil
}

func (d *documents) GetByResourceID(
	_ context.Context,
	req *documentports.GetDocumentsByResourceRequest,
) ([]*document.Document, error) {
	d.viewers = append(d.viewers, req.Viewer)

	return []*document.Document{}, nil
}

func (d *documents) List(
	_ context.Context,
	req *documentports.ListDocumentsRequest,
) (*pagination.ListResult[*document.Document], error) {
	d.viewers = append(d.viewers, req.Viewer)

	return &pagination.ListResult[*document.Document]{}, nil
}

func (d *documents) BulkDelete(
	_ context.Context,
	req documentports.BulkDeleteDocumentRequest,
) error {
	d.deleted = append(d.deleted, req.IDs...)

	return nil
}

type missingCache struct{}

func (missingCache) GetByID(
	context.Context,
	documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return nil, customerports.ErrCacheMiss
}

type grantedEngine struct {
	permissionports.PermissionEngine

	denied map[string]bool
}

func (e *grantedEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{Allowed: !e.denied[req.Resource]}, nil
}

type searchIndex struct {
	documentports.DocumentContentService

	hits []*document.Document
}

func (s *searchIndex) SearchDocuments(
	context.Context,
	pagination.TenantInfo,
	string,
	string,
	string,
) ([]*document.Document, error) {
	return s.hits, nil
}

type fixture struct {
	router *gin.Engine
	repo   *documents
	reader pulid.ID
}

func newFixture(
	t *testing.T,
	doc *document.Document,
	engine *grantedEngine,
	reader pulid.ID,
) *fixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := &documents{doc: doc}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    zap.NewNop(),
		AppConfig: &appconfig.Config{Debug: true},
	})
	handler := documenthandler.New(documenthandler.Params{
		Service: documentservice.New(documentservice.Params{
			Logger:               zap.NewNop(),
			Repo:                 repo,
			CacheRepo:            missingCache{},
			Permissions:          engine,
			DocumentIntelligence: &searchIndex{hits: []*document.Document{doc}},
		}),
		ErrorHandler: errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: engine,
				ErrorHandler:     errorHandler,
			},
		),
		StorageConfig: &storageconfig.Config{},
		Logger:        zap.NewNop(),
	})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		authctx.SetAuthContext(c, reader, doc.BusinessUnitID, doc.OrganizationID)
		c.Next()
	})
	handler.RegisterRoutes(router.Group(""))

	return &fixture{router: router, repo: repo, reader: reader}
}

func (f *fixture) do(method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(recorder, request)

	return recorder
}

func conversationFile(owner pulid.ID) *document.Document {
	return &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ResourceType:   document.ConversationResourceType,
		ResourceID:     pulid.MustNew("athr_").String(),
		UploadedByID:   owner,
		OriginalName:   "rate-confirmation.pdf",
	}
}

func TestEveryDocumentRoute_RefusesAnotherPersonsConversationFile(t *testing.T) {
	t.Parallel()

	doc := conversationFile(pulid.MustNew("usr_"))
	fixture := newFixture(t, doc, &grantedEngine{}, pulid.MustNew("usr_"))

	checked := 0
	for _, route := range fixture.router.Routes() {
		if !strings.Contains(route.Path, ":documentID") {
			continue
		}
		path := strings.NewReplacer(
			":documentID", doc.ID.String(),
			":pageNumber", "1",
		).Replace(route.Path)

		recorder := fixture.do(route.Method, path, "{}")

		assert.Equal(t, http.StatusNotFound, recorder.Code, "%s %s", route.Method, route.Path)
		checked++
	}
	assert.GreaterOrEqual(t, checked, 19, "every route naming a document is walked")
}

func TestDocumentRoute_LetsThePersonWhoAttachedTheFileRead(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	doc := conversationFile(owner)
	fixture := newFixture(t, doc, &grantedEngine{}, owner)

	recorder := fixture.do(http.MethodGet, "/documents/"+doc.ID.String()+"/", "")

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Contains(t, recorder.Body.String(), "rate-confirmation.pdf")
}

func TestDocumentRoute_FailsClosedOnAKindTheCatalogDoesNotKnow(t *testing.T) {
	t.Parallel()

	doc := conversationFile(pulid.Nil)
	doc.ResourceType = "not_a_record_kind"
	fixture := newFixture(t, doc, &grantedEngine{}, pulid.MustNew("usr_"))

	recorder := fixture.do(http.MethodGet, "/documents/"+doc.ID.String()+"/", "")

	assert.Equal(t, http.StatusNotFound, recorder.Code, recorder.Body.String())
}

func TestDocumentRoute_NeedsReadOnTheRecordTheDocumentIsOn(t *testing.T) {
	t.Parallel()

	doc := conversationFile(pulid.Nil)
	doc.ResourceType = permission.ResourceShipment.String()
	fixture := newFixture(
		t,
		doc,
		&grantedEngine{denied: map[string]bool{permission.ResourceShipment.String(): true}},
		pulid.MustNew("usr_"),
	)

	recorder := fixture.do(http.MethodGet, "/documents/"+doc.ID.String()+"/versions/", "")

	assert.Equal(t, http.StatusNotFound, recorder.Code, recorder.Body.String())
}

func TestBulkDelete_RefusesAnotherPersonsConversationFile(t *testing.T) {
	t.Parallel()

	doc := conversationFile(pulid.MustNew("usr_"))
	fixture := newFixture(t, doc, &grantedEngine{}, pulid.MustNew("usr_"))

	recorder := fixture.do(
		http.MethodPost,
		"/documents/bulk-delete/",
		`{"ids":["`+doc.ID.String()+`"]}`,
	)

	assert.Equal(t, http.StatusNotFound, recorder.Code, recorder.Body.String())
	assert.Empty(t, fixture.repo.deleted)
}

func TestResourceRoutes_ListOnlyWhatTheReaderMaySee(t *testing.T) {
	t.Parallel()

	doc := conversationFile(pulid.MustNew("usr_"))
	reader := pulid.MustNew("usr_")
	fixture := newFixture(t, doc, &grantedEngine{}, reader)
	base := "/documents/resource/" + document.ConversationResourceType + "/" + doc.ResourceID + "/"

	listed := fixture.do(http.MethodGet, base, "")
	require.Equal(t, http.StatusOK, listed.Code, listed.Body.String())
	assert.Equal(t, []pulid.ID{reader}, fixture.repo.viewers)

	searched := fixture.do(http.MethodGet, base+"?query=rate", "")
	require.Equal(t, http.StatusOK, searched.Code, searched.Body.String())
	assert.NotContains(t, searched.Body.String(), doc.ID.String())
}

func TestResourceRoutes_FailClosedOnAKindTheCatalogDoesNotKnow(t *testing.T) {
	t.Parallel()

	doc := conversationFile(pulid.Nil)
	fixture := newFixture(t, doc, &grantedEngine{}, pulid.MustNew("usr_"))

	recorder := fixture.do(http.MethodGet, "/documents/resource/not_a_record_kind/x/", "")

	assert.Equal(t, http.StatusNotFound, recorder.Code, recorder.Body.String())
}

func TestList_NamesTheReaderToTheRepository(t *testing.T) {
	t.Parallel()

	reader := pulid.MustNew("usr_")
	fixture := newFixture(t, conversationFile(pulid.Nil), &grantedEngine{}, reader)

	recorder := fixture.do(http.MethodGet, "/documents/", "")

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Equal(t, []pulid.ID{reader}, fixture.repo.viewers)
}

func TestEveryRecordKindADocumentHangsOffIsInTheCatalog(t *testing.T) {
	t.Parallel()

	reader := pulid.MustNew("usr_")
	for _, kind := range []string{
		"shipment", "order", "worker", "tractor", "trailer", "customer", "carrier", "permit",
		"invoice", "invoice_adjustment", "inbound_message", "shipment_comment", "shipment_import",
		"fuel_purchase", "fuel_purchase_import", "factoring_noa", "factoring_remittance",
		"factoring_submission", "driver_expense", "worker_credential",
	} {
		doc := conversationFile(pulid.Nil)
		doc.ResourceType = kind
		fixture := newFixture(t, doc, &grantedEngine{}, reader)

		recorder := fixture.do(http.MethodGet, "/documents/"+doc.ID.String()+"/", "")

		assert.Equal(t, http.StatusOK, recorder.Code, kind)
	}
}
