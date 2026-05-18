package edihandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/edihandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupEDIHandler(
	t *testing.T,
	repo *mocks.MockEDIDocumentRepository,
	applyParams ...func(*ediservice.Params),
) *edihandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})
	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})
	params := ediservice.Params{
		Logger:       logger,
		DocumentRepo: repo,
		Validator:    ediservice.NewValidator(),
	}
	for _, apply := range applyParams {
		apply(&params)
	}
	service := ediservice.New(params)

	return edihandler.New(edihandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestEDIHandler_PartnerSettingRoutes(t *testing.T) {
	t.Parallel()

	schemaID := pulid.MustNew("edips_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.On("ListPartnerSettingSchemas", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*edi.EDIPartnerSettingSchema]{
			Items: []*edi.EDIPartnerSettingSchema{{
				ID:             schemaID,
				Standard:       edi.EDIStandardX12,
				TransactionSet: edi.TransactionSet204,
				Direction:      edi.DocumentDirectionOutbound,
				X12Version:     edi.DefaultX12204Version,
				SchemaVersion:  1,
				Name:           "X12 204 Outbound Partner Settings",
				Status:         edi.PartnerSettingStatusActive,
			}},
			Total: 1,
		}, nil).
		Once()
	repo.On(
		"GetPartnerSettingSchema",
		mock.Anything,
		mock.MatchedBy(func(req repositories.GetEDIPartnerSettingSchemaRequest) bool {
			return req.ID == schemaID
		}),
	).Return(&edi.EDIPartnerSettingSchema{
		ID:            schemaID,
		SchemaVersion: 1,
		Name:          "X12 204 Outbound Partner Settings",
		Status:        edi.PartnerSettingStatusActive,
	}, nil).Once()
	repo.On(
		"ListPartnerSettingFields",
		mock.Anything,
		mock.MatchedBy(func(req *repositories.ListEDIPartnerSettingFieldsRequest) bool {
			return req.SchemaID == schemaID
		}),
	).Return(&pagination.ListResult[*edi.EDIPartnerSettingField]{
		Items: []*edi.EDIPartnerSettingField{{
			SchemaID: schemaID,
			Path:     "carrier.scac",
			Label:    "Carrier SCAC",
			DataType: edi.PartnerSettingDataTypeString,
			Required: true,
			Status:   edi.PartnerSettingStatusActive,
		}},
		Total: 1,
	}, nil).Once()
	repo.On(
		"SearchPartnerSettingFields",
		mock.Anything,
		mock.MatchedBy(func(req *repositories.ListEDIPartnerSettingFieldsRequest) bool {
			return req.Filter.Query == "referenceQualifiers" && req.PathPrefix == "carrier."
		}),
	).Return(&pagination.ListResult[*edi.EDIPartnerSettingField]{
		Items: []*edi.EDIPartnerSettingField{},
		Total: 0,
	}, nil).Once()
	repo.On(
		"SearchPartnerSettingFields",
		mock.Anything,
		mock.MatchedBy(func(req *repositories.ListEDIPartnerSettingFieldsRequest) bool {
			return req.Filter.Query == "contact"
		}),
	).Return(&pagination.ListResult[*edi.EDIPartnerSettingField]{
		Items: []*edi.EDIPartnerSettingField{{
			SchemaID: schemaID,
			Path:     "contact.phone",
			Label:    "Contact Phone",
			DataType: edi.PartnerSettingDataTypeString,
			Status:   edi.PartnerSettingStatusActive,
		}},
		Total: 1,
	}, nil).Once()

	handler := setupEDIHandler(t, repo)

	runEDIRequest(t, handler, http.MethodGet, "/api/v1/edi/partner-settings/schemas/", nil, nil, http.StatusOK)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/partner-settings/schemas/"+schemaID.String()+"/",
		nil,
		nil,
		http.StatusOK,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/partner-settings/schemas/"+schemaID.String()+"/fields/",
		nil,
		nil,
		http.StatusOK,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/partner-settings/fields/",
		map[string]string{"query": "referenceQualifiers", "pathPrefix": "carrier."},
		nil,
		http.StatusOK,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/partner-settings/fields/",
		map[string]string{"query": "contact"},
		nil,
		http.StatusOK,
	)
}

func TestEDIHandler_ValidatePartnerSettings(t *testing.T) {
	t.Parallel()

	schemaID := pulid.MustNew("edips_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.On("GetActivePartnerSettingSchema", mock.Anything, mock.Anything).
		Return(&edi.EDIPartnerSettingSchema{ID: schemaID, SchemaVersion: 1}, nil).
		Once()
	repo.On(
		"ListPartnerSettingFields",
		mock.Anything,
		mock.MatchedBy(func(req *repositories.ListEDIPartnerSettingFieldsRequest) bool {
			return req.SchemaID == schemaID
		}),
	).Return(&pagination.ListResult[*edi.EDIPartnerSettingField]{
		Items: []*edi.EDIPartnerSettingField{{
			SchemaID:  schemaID,
			Path:      "carrier.scac",
			Label:     "Carrier SCAC",
			DataType:  edi.PartnerSettingDataTypeString,
			Required:  true,
			Nullable:  false,
			MinLength: 2,
			MaxLength: 4,
			Status:    edi.PartnerSettingStatusActive,
		}},
		Total: 1,
	}, nil).Once()

	handler := setupEDIHandler(t, repo)
	recorder := runEDIRequest(
		t,
		handler,
		http.MethodPost,
		"/api/v1/edi/partner-settings/validate/",
		nil,
		map[string]any{"settings": map[string]any{}},
		http.StatusOK,
	)

	var resp struct {
		Diagnostics []edix12.Diagnostic `json:"diagnostics"`
	}
	require.NoError(t, recorder.ResponseJSON(&resp))
	require.Len(t, resp.Diagnostics, 1)
	assert.Equal(t, "partner_setting_required", resp.Diagnostics[0].Code)
}

func TestEDIHandler_SelectOptionsRoutes(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("editpl_")
	profileID := pulid.MustNew("edipdp_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.On(
		"SelectDocumentTypeOptions",
		mock.Anything,
		mock.MatchedBy(func(req *repositories.EDIDocumentTypeSelectOptionsRequest) bool {
			return req.SelectQueryRequest.Query == "204" &&
				req.TransactionSet == edi.TransactionSet204 &&
				req.Direction == edi.DocumentDirectionOutbound
		}),
	).Return(&pagination.ListResult[*edi.EDIDocumentType]{
		Items: []*edi.EDIDocumentType{{
			ID:             pulid.MustNew("edidt_"),
			Code:           "204",
			Name:           "Motor Carrier Load Tender",
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
			DefaultVersion: edi.DefaultX12204Version,
			Status:         edi.DocumentStatusActive,
		}},
		Total: 1,
	}, nil).Once()
	repo.On(
		"SelectTemplateOptions",
		mock.Anything,
		mock.MatchedBy(func(req *repositories.EDITemplateSelectOptionsRequest) bool {
			return req.SelectQueryRequest.Query == "load" &&
				req.TransactionSet == edi.TransactionSet204 &&
				req.Direction == edi.DocumentDirectionOutbound
		}),
	).Return(&pagination.ListResult[*edi.EDITemplate]{
		Items: []*edi.EDITemplate{{
			ID:             templateID,
			Name:           "Outbound 204",
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
			Status:         edi.TemplateStatusDraft,
		}},
		Total: 1,
	}, nil).Once()
	repo.On(
		"SelectPartnerDocumentProfileOptions",
		mock.Anything,
		mock.MatchedBy(func(req *repositories.EDIPartnerDocumentProfileSelectOptionsRequest) bool {
			return req.SelectQueryRequest.Query == "profile" &&
				req.TransactionSet == edi.TransactionSet204 &&
				req.Direction == edi.DocumentDirectionOutbound
		}),
	).Return(&pagination.ListResult[*edi.EDIPartnerDocumentProfile]{
		Items: []*edi.EDIPartnerDocumentProfile{{
			ID:             profileID,
			Name:           "Outbound Profile",
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
		}},
		Total: 1,
	}, nil).Once()
	repo.On(
		"SelectSourceContextFieldOptions",
		mock.Anything,
		mock.MatchedBy(func(req *repositories.ListEDISourceContextFieldsRequest) bool {
			return req.Filter.Query == "bol" &&
				req.Status == edi.SourceContextFieldStatusActive &&
				req.TransactionSet == edi.TransactionSet204
		}),
	).Return(&pagination.ListResult[*edi.EDISourceContextField]{
		Items: []*edi.EDISourceContextField{{
			Path:        "shipment.bol",
			DisplayName: "BOL",
			DataType:    edi.SourceContextDataTypeString,
			Status:      edi.SourceContextFieldStatusActive,
		}},
		Total: 1,
	}, nil).Once()
	repo.On(
		"SelectPartnerSettingFieldOptions",
		mock.Anything,
		mock.MatchedBy(func(req *repositories.ListEDIPartnerSettingFieldsRequest) bool {
			return req.Filter.Query == "receiver" &&
				req.Status == edi.PartnerSettingStatusActive &&
				req.TransactionSet == edi.TransactionSet204
		}),
	).Return(&pagination.ListResult[*edi.EDIPartnerSettingField]{
		Items: []*edi.EDIPartnerSettingField{{
			Path:     "envelope.receiverId",
			Label:    "Receiver ID",
			DataType: edi.PartnerSettingDataTypeString,
			Status:   edi.PartnerSettingStatusActive,
		}},
		Total: 1,
	}, nil).Once()

	handler := setupEDIHandler(t, repo)

	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/document-types/select-options/",
		map[string]string{"query": "204", "transactionSet": "204", "direction": "Outbound"},
		nil,
		http.StatusOK,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/templates/select-options/",
		map[string]string{"query": "load", "transactionSet": "204", "direction": "Outbound"},
		nil,
		http.StatusOK,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/document-profiles/select-options/",
		map[string]string{"query": "profile", "transactionSet": "204", "direction": "Outbound"},
		nil,
		http.StatusOK,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/source-context/fields/select-options/",
		map[string]string{"query": "bol", "status": "Active", "transactionSet": "204"},
		nil,
		http.StatusOK,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/partner-settings/fields/select-options/",
		map[string]string{"query": "receiver", "status": "Active", "transactionSet": "204"},
		nil,
		http.StatusOK,
	)
}

func TestEDIHandler_SelectedValueRoutesReturnAutocompleteFields(t *testing.T) {
	t.Parallel()

	partnerID := pulid.MustNew("edip_")
	documentTypeID := pulid.MustNew("edidt_")
	templateID := pulid.MustNew("editpl_")
	profileID := pulid.MustNew("edipdp_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	partnerRepo := mocks.NewMockEDIPartnerRepository(t)

	partnerRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req repositories.GetEDIPartnerByIDRequest) bool {
			return req.ID == partnerID && req.TenantInfo.OrgID.IsNotNil() && req.TenantInfo.BuID.IsNotNil()
		})).
		Return(&edi.EDIPartner{
			ID:   partnerID,
			Code: "CARR",
			Name: "Carrier Partner",
			Kind: edi.PartnerKindExternal,
		}, nil).
		Once()
	repo.EXPECT().
		GetTemplateByID(mock.Anything, mock.MatchedBy(func(req repositories.GetEDITemplateByIDRequest) bool {
			return req.ID == templateID && req.TenantInfo.OrgID.IsNotNil() && req.TenantInfo.BuID.IsNotNil()
		})).
		Return(&edi.EDITemplate{
			ID:          templateID,
			Name:        "Outbound 204",
			Description: "Load tender template",
			Status:      edi.TemplateStatusDraft,
		}, nil).
		Once()
	repo.EXPECT().
		GetPartnerDocumentProfileByID(
			mock.Anything,
			mock.MatchedBy(func(req repositories.GetEDIPartnerDocumentProfileByIDRequest) bool {
				return req.ID == profileID && req.TenantInfo.OrgID.IsNotNil() && req.TenantInfo.BuID.IsNotNil()
			}),
		).
		Return(&edi.EDIPartnerDocumentProfile{
			ID:   profileID,
			Name: "Outbound Profile",
			Partner: &edi.EDIPartner{
				ID:   partnerID,
				Code: "CARR",
				Name: "Carrier Partner",
			},
		}, nil).
		Once()
	repo.EXPECT().
		ListDocumentTypes(mock.Anything, repositories.ListEDIDocumentTypesRequest{}).
		Return([]*edi.EDIDocumentType{{
			ID:             documentTypeID,
			Code:           "204",
			Name:           "Motor Carrier Load Tender",
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
			DefaultVersion: edi.DefaultX12204Version,
			Status:         edi.DocumentStatusActive,
		}}, nil).
		Once()

	handler := setupEDIHandler(t, repo, func(params *ediservice.Params) {
		params.PartnerRepo = partnerRepo
	})

	partnerRecorder := runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/partners/"+partnerID.String()+"/",
		nil,
		nil,
		http.StatusOK,
	)
	var partnerResp edi.EDIPartner
	require.NoError(t, partnerRecorder.ResponseJSON(&partnerResp))
	assert.Equal(t, partnerID, partnerResp.ID)
	assert.Equal(t, "CARR", partnerResp.Code)
	assert.Equal(t, "Carrier Partner", partnerResp.Name)
	assert.Equal(t, edi.PartnerKindExternal, partnerResp.Kind)

	templateRecorder := runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/templates/select-options/"+templateID.String(),
		nil,
		nil,
		http.StatusOK,
	)
	var templateResp edi.EDITemplate
	require.NoError(t, templateRecorder.ResponseJSON(&templateResp))
	assert.Equal(t, templateID, templateResp.ID)
	assert.Equal(t, "Outbound 204", templateResp.Name)
	assert.Equal(t, "Load tender template", templateResp.Description)
	assert.Equal(t, edi.TemplateStatusDraft, templateResp.Status)

	profileRecorder := runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/document-profiles/select-options/"+profileID.String(),
		nil,
		nil,
		http.StatusOK,
	)
	var profileResp edi.EDIPartnerDocumentProfile
	require.NoError(t, profileRecorder.ResponseJSON(&profileResp))
	assert.Equal(t, profileID, profileResp.ID)
	assert.Equal(t, "Outbound Profile", profileResp.Name)
	require.NotNil(t, profileResp.Partner)
	assert.Equal(t, "CARR", profileResp.Partner.Code)
	assert.Equal(t, "Carrier Partner", profileResp.Partner.Name)

	documentTypeRecorder := runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/document-types/select-options/"+documentTypeID.String(),
		nil,
		nil,
		http.StatusOK,
	)
	var documentTypeResp edi.EDIDocumentType
	require.NoError(t, documentTypeRecorder.ResponseJSON(&documentTypeResp))
	assert.Equal(t, documentTypeID, documentTypeResp.ID)
	assert.Equal(t, "204", documentTypeResp.Code)
	assert.Equal(t, "Motor Carrier Load Tender", documentTypeResp.Name)
	assert.Equal(t, edi.TransactionSet204, documentTypeResp.TransactionSet)
	assert.Equal(t, edi.DocumentDirectionOutbound, documentTypeResp.Direction)
	assert.Equal(t, edi.DefaultX12204Version, documentTypeResp.DefaultVersion)
}

func runEDIRequest(
	t *testing.T,
	handler *edihandler.Handler,
	method string,
	path string,
	query map[string]string,
	body any,
	wantStatus int,
) *testutil.GinTestContext {
	t.Helper()

	ginCtx := testutil.NewGinTestContext().
		WithMethod(method).
		WithPath(path).
		WithDefaultAuthContext()
	if len(query) > 0 {
		ginCtx.WithQuery(query)
	}
	if body != nil {
		ginCtx.WithJSONBody(body)
	}
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, wantStatus, ginCtx.ResponseCode())
	return ginCtx
}
