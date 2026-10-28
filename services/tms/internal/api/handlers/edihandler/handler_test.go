package edihandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/edihandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/edimessageportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/editenderportsmock"
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
		Logger:             logger,
		DocumentTypeRepo:   edipartnerportsmock.NewMockEDIDocumentTypeRepository(t),
		TransactionSetRepo: edipartnerportsmock.NewMockEDITransactionSetRepository(t),
		ControlNumberRepo:  edipartnerportsmock.NewMockEDIControlNumberRepository(t),
		MessageRepo:        edimessageportsmock.NewMockEDIMessageRepository(t),
		Validator:          ediservice.NewValidator(),
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

func TestEDIHandler_DocumentTypeSelectOptionsRoute(t *testing.T) {
	t.Parallel()

	repo := edipartnerportsmock.NewMockEDIDocumentTypeRepository(t)
	repo.On(
		"SelectDocumentTypeOptions",
		mock.Anything,
		mock.MatchedBy(func(req *edipartnerports.EDIDocumentTypeSelectOptionsRequest) bool {
			return req.SelectQueryRequest.Query == "204" &&
				req.TransactionSet == edidocument.TransactionSet204 &&
				req.Direction == edidocument.DocumentDirectionOutbound
		}),
	).Return(&pagination.ListResult[*edidocument.EDIDocumentType]{
		Items: []*edidocument.EDIDocumentType{{
			ID:             pulid.MustNew("edidt_"),
			Code:           "204",
			Name:           "Motor Carrier Load Tender",
			Standard:       edidocument.EDIStandardX12,
			TransactionSet: edidocument.TransactionSet204,
			Direction:      edidocument.DocumentDirectionOutbound,
			DefaultVersion: edidocument.DefaultX12204Version,
			Status:         edidocument.DocumentStatusActive,
		}},
		Total:      1,
		TotalExact: true,
	}, nil).Once()
	handler := setupEDIHandler(t, func(params *ediservice.Params) {
		params.DocumentTypeRepo = repo
	})

	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/catalog/document-types/select-options/",
		map[string]string{"query": "204", "transactionSet": "204", "direction": "Outbound"},
		nil,
		http.StatusOK,
	)
}

func TestEDIHandler_TransactionSetCatalogRoutes(t *testing.T) {
	t.Parallel()

	tender := &edidocument.EDITransactionSet{
		ID:             pulid.ID("edits_x12_204"),
		Standard:       edidocument.EDIStandardX12,
		Code:           edidocument.TransactionSet204,
		Name:           "Motor Carrier Load Tender",
		Description:    "Outbound load tender document.",
		DefaultVersion: edidocument.DefaultX12204Version,
		Status:         edidocument.DocumentStatusActive,
	}
	repo := edipartnerportsmock.NewMockEDITransactionSetRepository(t)
	repo.On(
		"SelectTransactionSetOptions",
		mock.Anything,
		mock.MatchedBy(func(req *edipartnerports.EDITransactionSetSelectOptionsRequest) bool {
			return req.SelectQueryRequest.Query == "204" &&
				req.Standard == edidocument.EDIStandardX12 &&
				req.Status == edidocument.DocumentStatusActive &&
				len(req.IDs) == 0
		}),
	).Return(&pagination.ListResult[*edidocument.EDITransactionSet]{
		Items:      []*edidocument.EDITransactionSet{tender},
		Total:      1,
		TotalExact: true,
	}, nil).Once()
	repo.On(
		"SelectTransactionSetOptions",
		mock.Anything,
		mock.MatchedBy(func(req *edipartnerports.EDITransactionSetSelectOptionsRequest) bool {
			return len(req.IDs) == 1 &&
				req.IDs[0] == tender.ID &&
				req.SelectQueryRequest.Query == "" &&
				req.SelectQueryRequest.Pagination.Limit == 1
		}),
	).Return(&pagination.ListResult[*edidocument.EDITransactionSet]{
		Items:      []*edidocument.EDITransactionSet{tender},
		Total:      1,
		TotalExact: true,
	}, nil).Once()
	repo.On(
		"SelectTransactionSetOptions",
		mock.Anything,
		mock.MatchedBy(func(req *edipartnerports.EDITransactionSetSelectOptionsRequest) bool {
			return len(req.IDs) == 1 && req.IDs[0] == pulid.ID("edits_x12_000")
		}),
	).Return(&pagination.ListResult[*edidocument.EDITransactionSet]{}, nil).Once()

	handler := setupEDIHandler(t, func(params *ediservice.Params) {
		params.TransactionSetRepo = repo
	})

	listRecorder := runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/catalog/transaction-sets/select-options/",
		map[string]string{"query": "204", "standard": "X12", "status": "Active"},
		nil,
		http.StatusOK,
	)
	var listResp pagination.Response[[]*edidocument.EDITransactionSet]
	require.NoError(t, listRecorder.ResponseJSON(&listResp))
	require.Len(t, listResp.Results, 1)
	assert.Equal(t, 1, listResp.Count)
	assert.Equal(t, tender.ID, listResp.Results[0].ID)
	assert.Equal(t, edidocument.TransactionSet204, listResp.Results[0].Code)

	oneRecorder := runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/catalog/transaction-sets/select-options/edits_x12_204",
		nil,
		nil,
		http.StatusOK,
	)
	var oneResp edidocument.EDITransactionSet
	require.NoError(t, oneRecorder.ResponseJSON(&oneResp))
	assert.Equal(t, tender.ID, oneResp.ID)
	assert.Equal(t, "Motor Carrier Load Tender", oneResp.Name)

	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/catalog/transaction-sets/select-options/edits_x12_000",
		nil,
		nil,
		http.StatusNotFound,
	)
}

func TestEDIHandler_DocumentTypeOptionAcceptsCatalogID(t *testing.T) {
	t.Parallel()

	repo := edipartnerportsmock.NewMockEDIDocumentTypeRepository(t)
	repo.EXPECT().
		ListDocumentTypes(mock.Anything, edipartnerports.ListEDIDocumentTypesRequest{}).
		Return([]*edidocument.EDIDocumentType{{
			ID:             pulid.ID("edidt_x12_204_outbound"),
			Code:           "X12-204-OUT",
			Name:           "X12 204 Motor Carrier Load Tender",
			Standard:       edidocument.EDIStandardX12,
			TransactionSet: edidocument.TransactionSet204,
			Direction:      edidocument.DocumentDirectionOutbound,
			DefaultVersion: edidocument.DefaultX12204Version,
			Status:         edidocument.DocumentStatusActive,
		}}, nil).
		Once()

	handler := setupEDIHandler(t, func(params *ediservice.Params) {
		params.DocumentTypeRepo = repo
	})

	recorder := runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/catalog/document-types/select-options/edidt_x12_204_outbound",
		nil,
		nil,
		http.StatusOK,
	)
	var resp edidocument.EDIDocumentType
	require.NoError(t, recorder.ResponseJSON(&resp))
	assert.Equal(t, pulid.ID("edidt_x12_204_outbound"), resp.ID)
}

func TestEDIHandler_ProfileSelectOptionsRoutes(t *testing.T) {
	t.Parallel()

	partnerID := pulid.MustNew("edip_")
	communicationProfileID := pulid.MustNew("edicp_")
	mappingProfileID := pulid.MustNew("edimp_")
	repo := edipartnerportsmock.NewMockEDIDocumentTypeRepository(t)
	profileRepo := edipartnerportsmock.NewMockEDICommunicationProfileRepository(t)
	partnerRepo := edipartnerportsmock.NewMockEDIPartnerRepository(t)
	mappingProfileRepo := edipartnerportsmock.NewMockEDIMappingProfileRepository(t)

	profileRepo.EXPECT().
		SelectProfileOptions(mock.Anything, mock.MatchedBy(func(req *edipartnerports.EDICommunicationProfileSelectOptionsRequest) bool {
			return req.SelectQueryRequest.Query == "sftp" &&
				req.Status == "Active" &&
				req.Method == edi.ConnectionMethodSFTP &&
				req.PartnerID == partnerID
		})).
		Return(&pagination.ListResult[*edi.EDICommunicationProfile]{
			Items: []*edi.EDICommunicationProfile{{
				ID:           communicationProfileID,
				EDIPartnerID: partnerID,
				Method:       edi.ConnectionMethodSFTP,
				Status:       "Active",
				Name:         "Carrier SFTP",
			}},
			Total:      1,
			TotalExact: true,
		}, nil).
		Once()
	mappingProfileRepo.EXPECT().
		SelectMappingProfileOptions(mock.Anything, mock.MatchedBy(func(req *edipartnerports.EDIMappingProfileSelectOptionsRequest) bool {
			return req.SelectQueryRequest.Query == "carrier" && req.PartnerID == partnerID
		})).
		Return(&pagination.ListResult[*edi.EDIMappingProfile]{
			Items: []*edi.EDIMappingProfile{{
				ID:           mappingProfileID,
				EDIPartnerID: partnerID,
				Name:         "Carrier Mapping",
			}},
			Total:      1,
			TotalExact: true,
		}, nil).
		Once()

	handler := setupEDIHandler(t, func(params *ediservice.Params) {
		params.DocumentTypeRepo = repo
		params.ProfileRepo = profileRepo
		params.PartnerRepo = partnerRepo
		params.MappingProfileRepo = mappingProfileRepo
	})

	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/communication-profiles/select-options/",
		map[string]string{
			"query":     "sftp",
			"status":    "Active",
			"method":    "SFTP",
			"partnerId": partnerID.String(),
		},
		nil,
		http.StatusOK,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/mapping-profiles/select-options/",
		map[string]string{"query": "carrier", "partnerId": partnerID.String()},
		nil,
		http.StatusOK,
	)
}

func TestEDIHandler_TransferListDirectionRoutes(t *testing.T) {
	t.Parallel()

	repo := edipartnerportsmock.NewMockEDIDocumentTypeRepository(t)
	transferRepo := editenderportsmock.NewMockEDILoadTenderTransferRepository(t)
	transferRepo.EXPECT().
		ListInbound(mock.Anything, mock.MatchedBy(func(req *editenderports.ListEDITransfersRequest) bool {
			return req.Filter != nil
		})).
		Return(&pagination.ListResult[*editransfer.EDITransfer]{
			Items:      []*editransfer.EDITransfer{},
			Total:      0,
			TotalExact: true,
		}, nil).
		Once()
	transferRepo.EXPECT().
		ListOutbound(mock.Anything, mock.MatchedBy(func(req *editenderports.ListEDITransfersRequest) bool {
			return req.Filter != nil
		})).
		Return(&pagination.ListResult[*editransfer.EDITransfer]{
			Items:      []*editransfer.EDITransfer{},
			Total:      0,
			TotalExact: true,
		}, nil).
		Once()

	handler := setupEDIHandler(t, func(params *ediservice.Params) {
		params.DocumentTypeRepo = repo
		params.TransferRepo = transferRepo
	})

	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/transfers/",
		map[string]string{"direction": "inbound"},
		nil,
		http.StatusOK,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/transfers/",
		map[string]string{"direction": "outbound"},
		nil,
		http.StatusOK,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/transfers/",
		map[string]string{"direction": "sideways"},
		nil,
		http.StatusBadRequest,
	)
}

func TestEDIHandler_CreatePartnerDefaultsAndExplicitFalseFlags(t *testing.T) {
	t.Parallel()

	repo := edipartnerportsmock.NewMockEDIDocumentTypeRepository(t)
	partnerRepo := edipartnerportsmock.NewMockEDIPartnerRepository(t)
	partnerRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *edi.EDIPartner) bool {
			return entity.Kind == edi.PartnerKindExternal &&
				entity.Status == "Active" &&
				entity.Country == "US" &&
				entity.EnabledForInbound &&
				entity.EnabledForOutbound
		})).
		RunAndReturn(func(_ context.Context, entity *edi.EDIPartner) (*edi.EDIPartner, error) {
			entity.ID = pulid.MustNew("edip_")
			return entity, nil
		}).
		Once()
	partnerRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *edi.EDIPartner) bool {
			return entity.Kind == edi.PartnerKindExternal &&
				entity.InternalOrganizationID.IsNil() &&
				!entity.EnabledForInbound &&
				!entity.EnabledForOutbound
		})).
		RunAndReturn(func(_ context.Context, entity *edi.EDIPartner) (*edi.EDIPartner, error) {
			entity.ID = pulid.MustNew("edip_")
			return entity, nil
		}).
		Once()
	handler := setupEDIHandler(t, func(params *ediservice.Params) {
		params.DocumentTypeRepo = repo
		params.PartnerRepo = partnerRepo
	})

	runEDIRequest(
		t,
		handler,
		http.MethodPost,
		"/api/v1/edi/partners/",
		nil,
		map[string]any{"code": "EXT", "name": "External Partner"},
		http.StatusCreated,
	)
	runEDIRequest(
		t,
		handler,
		http.MethodPost,
		"/api/v1/edi/partners/",
		nil,
		map[string]any{
			"kind":               "External",
			"status":             "Active",
			"code":               "EXT2",
			"name":               "External Partner 2",
			"country":            "US",
			"enabledForInbound":  false,
			"enabledForOutbound": false,
		},
		http.StatusCreated,
	)
}

func TestEDIHandler_SelectedValueRoutesReturnAutocompleteFields(t *testing.T) {
	t.Parallel()

	partnerID := pulid.MustNew("edip_")
	documentTypeID := pulid.MustNew("edidt_")
	communicationProfileID := pulid.MustNew("edicp_")
	mappingProfileID := pulid.MustNew("edimp_")
	repo := edipartnerportsmock.NewMockEDIDocumentTypeRepository(t)
	partnerRepo := edipartnerportsmock.NewMockEDIPartnerRepository(t)
	profileRepo := edipartnerportsmock.NewMockEDICommunicationProfileRepository(t)
	mappingProfileRepo := edipartnerportsmock.NewMockEDIMappingProfileRepository(t)

	partnerRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req edipartnerports.GetEDIPartnerByIDRequest) bool {
			return req.ID == partnerID && req.TenantInfo.OrgID.IsNotNil() &&
				req.TenantInfo.BuID.IsNotNil()
		})).
		Return(&edi.EDIPartner{
			ID:   partnerID,
			Code: "CARR",
			Name: "Carrier Partner",
			Kind: edi.PartnerKindExternal,
		}, nil).
		Once()
	repo.EXPECT().
		ListDocumentTypes(mock.Anything, edipartnerports.ListEDIDocumentTypesRequest{}).
		Return([]*edidocument.EDIDocumentType{{
			ID:             documentTypeID,
			Code:           "204",
			Name:           "Motor Carrier Load Tender",
			Standard:       edidocument.EDIStandardX12,
			TransactionSet: edidocument.TransactionSet204,
			Direction:      edidocument.DocumentDirectionOutbound,
			DefaultVersion: edidocument.DefaultX12204Version,
			Status:         edidocument.DocumentStatusActive,
		}}, nil).
		Once()
	profileRepo.EXPECT().
		GetProfileByID(mock.Anything, mock.MatchedBy(func(req edipartnerports.GetEDICommunicationProfileByIDRequest) bool {
			return req.ID == communicationProfileID && req.TenantInfo.OrgID.IsNotNil() &&
				req.TenantInfo.BuID.IsNotNil()
		})).
		Return(&edi.EDICommunicationProfile{
			ID:     communicationProfileID,
			Method: edi.ConnectionMethodSFTP,
			Status: "Active",
			Name:   "Carrier SFTP",
		}, nil).
		Once()
	mappingProfileRepo.EXPECT().
		GetMappingProfileByID(mock.Anything, mock.MatchedBy(func(req edipartnerports.GetMappingProfileByIDRequest) bool {
			return req.ProfileID == mappingProfileID && req.TenantInfo.OrgID.IsNotNil() &&
				req.TenantInfo.BuID.IsNotNil()
		})).
		Return(&edi.EDIMappingProfile{
			ID:   mappingProfileID,
			Name: "Carrier Mapping",
		}, nil).
		Once()

	handler := setupEDIHandler(t, func(params *ediservice.Params) {
		params.DocumentTypeRepo = repo
		params.PartnerRepo = partnerRepo
		params.MappingProfileRepo = mappingProfileRepo
		params.ProfileRepo = profileRepo
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

	documentTypeRecorder := runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/catalog/document-types/select-options/"+documentTypeID.String(),
		nil,
		nil,
		http.StatusOK,
	)
	var documentTypeResp edidocument.EDIDocumentType
	require.NoError(t, documentTypeRecorder.ResponseJSON(&documentTypeResp))
	assert.Equal(t, documentTypeID, documentTypeResp.ID)
	assert.Equal(t, "204", documentTypeResp.Code)
	assert.Equal(t, "Motor Carrier Load Tender", documentTypeResp.Name)
	assert.Equal(t, edidocument.TransactionSet204, documentTypeResp.TransactionSet)
	assert.Equal(t, edidocument.DocumentDirectionOutbound, documentTypeResp.Direction)
	assert.Equal(t, edidocument.DefaultX12204Version, documentTypeResp.DefaultVersion)

	communicationProfileRecorder := runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/communication-profiles/select-options/"+communicationProfileID.String(),
		nil,
		nil,
		http.StatusOK,
	)
	var communicationProfileResp edi.EDICommunicationProfile
	require.NoError(t, communicationProfileRecorder.ResponseJSON(&communicationProfileResp))
	assert.Equal(t, communicationProfileID, communicationProfileResp.ID)
	assert.Equal(t, "Carrier SFTP", communicationProfileResp.Name)

	mappingProfileRecorder := runEDIRequest(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/mapping-profiles/select-options/"+mappingProfileID.String(),
		nil,
		nil,
		http.StatusOK,
	)
	var mappingProfileResp edi.EDIMappingProfile
	require.NoError(t, mappingProfileRecorder.ResponseJSON(&mappingProfileResp))
	assert.Equal(t, mappingProfileID, mappingProfileResp.ID)
	assert.Equal(t, "Carrier Mapping", mappingProfileResp.Name)
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
