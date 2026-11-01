package edisetuphandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/edisetuphandler"
	"github.com/emoss08/trenova/internal/modules/edi/app/edivalidation"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/testutil/edihandlertest"
	"github.com/emoss08/trenova/internal/testutil/ediservicetest"
	"github.com/emoss08/trenova/internal/testutil/mocks/edimessageportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func setupEDIHandler(
	t *testing.T,
	applyParams ...func(*ediservicetest.Deps),
) *edisetuphandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	params := ediservicetest.Deps{
		Logger:             logger,
		DocumentTypeRepo:   edipartnerportsmock.NewMockEDIDocumentTypeRepository(t),
		TransactionSetRepo: edipartnerportsmock.NewMockEDITransactionSetRepository(t),
		ControlNumberRepo:  edipartnerportsmock.NewMockEDIControlNumberRepository(t),
		MessageRepo:        edimessageportsmock.NewMockEDIMessageRepository(t),
		Validator:          edivalidation.NewValidator(),
	}
	for _, apply := range applyParams {
		apply(&params)
	}
	edis := ediservicetest.New(params)
	mw := edihandlertest.NewMiddleware(logger)

	return edisetuphandler.New(edisetuphandler.Params{
		Setup:                edis.Setup,
		Overview:             edis.Overview,
		ErrorHandler:         mw.ErrorHandler,
		PermissionMiddleware: mw.PermissionMiddleware,
	})
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
	handler := setupEDIHandler(t, func(params *ediservicetest.Deps) {
		params.DocumentTypeRepo = repo
		params.PartnerRepo = partnerRepo
	})

	edihandlertest.Run(
		t,
		handler,
		http.MethodPost,
		"/api/v1/edi/partners/",
		nil,
		map[string]any{"code": "EXT", "name": "External Partner"},
		http.StatusCreated,
	)
	edihandlertest.Run(
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
