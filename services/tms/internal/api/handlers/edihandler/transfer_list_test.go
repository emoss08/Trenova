package edihandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/edihandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/testutil/ediservicetest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/editenderportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func setupTransferHandler(
	t *testing.T,
	transfers *editenderportsmock.MockEDILoadTenderTransferRepository,
) *edihandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &appconfig.Config{Debug: true},
	})
	params := serviceParams(ediservicetest.New(ediservicetest.Deps{
		Logger:       logger,
		TransferRepo: transfers,
	}))
	params.ErrorHandler = errorHandler
	params.PermissionMiddleware = middleware.NewPermissionMiddleware(
		middleware.PermissionMiddlewareParams{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
			ErrorHandler:     errorHandler,
		},
	)
	return edihandler.New(params)
}

func TestEDIHandler_ListInboundTransfersFromOneInboundFile(t *testing.T) {
	t.Parallel()

	fileID := pulid.MustNew("ediinf_")
	transfers := editenderportsmock.NewMockEDILoadTenderTransferRepository(t)
	transfers.EXPECT().
		ListInbound(mock.Anything, mock.MatchedBy(func(req *editenderports.ListEDITransfersRequest) bool {
			return req.InboundFileID == fileID
		})).
		Return(&pagination.ListResult[*editransfer.EDITransfer]{}, nil).
		Once()

	runEDIRequest(
		t,
		setupTransferHandler(t, transfers),
		http.MethodGet,
		"/api/v1/edi/transfers/",
		map[string]string{"direction": "inbound", "inboundFileId": fileID.String()},
		nil,
		http.StatusOK,
	)
}

func TestEDIHandler_ListInboundTransfersRefusesAMalformedInboundFileID(t *testing.T) {
	t.Parallel()

	runEDIRequest(
		t,
		setupTransferHandler(t, editenderportsmock.NewMockEDILoadTenderTransferRepository(t)),
		http.MethodGet,
		"/api/v1/edi/transfers/",
		map[string]string{"direction": "inbound", "inboundFileId": "not-an-id"},
		nil,
		http.StatusBadRequest,
	)
}
