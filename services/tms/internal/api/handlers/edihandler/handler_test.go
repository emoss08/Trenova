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
	service := ediservice.New(ediservice.Params{
		Logger:       logger,
		DocumentRepo: repo,
		Validator:    ediservice.NewValidator(),
	})

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
