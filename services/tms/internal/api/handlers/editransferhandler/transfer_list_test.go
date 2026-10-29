package editransferhandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/editransferhandler"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/testutil/edihandlertest"
	"github.com/emoss08/trenova/internal/testutil/ediservicetest"
	"github.com/emoss08/trenova/internal/testutil/mocks/editenderportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func setupTransferHandler(
	t *testing.T,
	transfers *editenderportsmock.MockEDILoadTenderTransferRepository,
) *editransferhandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	edis := ediservicetest.New(ediservicetest.Deps{
		Logger:       logger,
		TransferRepo: transfers,
	})
	mw := edihandlertest.NewMiddleware(logger)

	return editransferhandler.New(editransferhandler.Params{
		Setup:                edis.Setup,
		Transfers:            edis.Transfers,
		TenderChanges:        edis.TenderChanges,
		TransferChanges:      edis.TransferChanges,
		Tenders:              edis.Tenders,
		ErrorHandler:         mw.ErrorHandler,
		PermissionMiddleware: mw.PermissionMiddleware,
	})
}

func TestEDIHandler_TransferListDirectionRoutes(t *testing.T) {
	t.Parallel()

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

	handler := setupTransferHandler(t, transferRepo)

	edihandlertest.Run(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/transfers/",
		map[string]string{"direction": "inbound"},
		nil,
		http.StatusOK,
	)
	edihandlertest.Run(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/transfers/",
		map[string]string{"direction": "outbound"},
		nil,
		http.StatusOK,
	)
	edihandlertest.Run(
		t,
		handler,
		http.MethodGet,
		"/api/v1/edi/transfers/",
		map[string]string{"direction": "sideways"},
		nil,
		http.StatusBadRequest,
	)
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

	edihandlertest.Run(
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

	edihandlertest.Run(
		t,
		setupTransferHandler(t, editenderportsmock.NewMockEDILoadTenderTransferRepository(t)),
		http.MethodGet,
		"/api/v1/edi/transfers/",
		map[string]string{"direction": "inbound", "inboundFileId": "not-an-id"},
		nil,
		http.StatusBadRequest,
	)
}
