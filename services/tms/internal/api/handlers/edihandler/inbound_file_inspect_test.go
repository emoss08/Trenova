package edihandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/edihandler"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/testutil/edihandlertest"
	"github.com/emoss08/trenova/internal/testutil/ediservicetest"
	"github.com/emoss08/trenova/internal/testutil/mocks/edimessageportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const inspectedInterchange = "ISA*00*          *00*          *ZZ*SIMPARTNER     *ZZ*TRENOVA        " +
	"*261014*1513*^*00401*000000042*0*T*>~GS*SM*SIMPARTNER*TRENOVA*20261014*1513*42*X*004010~" +
	"ST*204*0001~B2**SIML**SIM000042**PP~B2A*00~SE*4*0001~GE*1*42~IEA*1*000000042~"

func setupInboundFileHandler(
	t *testing.T,
	files *edimessageportsmock.MockEDIInboundFileRepository,
) *edihandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	edis := ediservicetest.New(ediservicetest.Deps{Logger: logger})
	mw := edihandlertest.NewMiddleware(logger)

	return edihandler.New(edihandler.Params{
		Delivery: edis.Delivery,
		Outbound: edis.Outbound,
		Messages: edis.Messages,
		InboundService: ediinboundservice.New(
			ediinboundservice.Params{Logger: logger, InboundFileRepo: files},
		),
		ErrorHandler:         mw.ErrorHandler,
		PermissionMiddleware: mw.PermissionMiddleware,
	})
}

func TestEDIHandler_InspectInboundFileLaysOutTheFileItHolds(t *testing.T) {
	t.Parallel()

	fileID := pulid.MustNew("ediinf_")
	files := edimessageportsmock.NewMockEDIInboundFileRepository(t)
	files.EXPECT().
		GetInboundFileByID(mock.Anything, mock.MatchedBy(func(req edimessageports.GetEDIInboundFileByIDRequest) bool {
			return req.ID == fileID && !req.IncludeMessages
		})).
		Return(&ediinboundfile.EDIInboundFile{ID: fileID, RawContent: inspectedInterchange}, nil).
		Once()

	ctx := edihandlertest.Run(
		t,
		setupInboundFileHandler(t, files),
		http.MethodGet,
		"/api/v1/edi/inbound-files/"+fileID.String()+"/inspect/",
		nil,
		nil,
		http.StatusOK,
	)

	var result edix12inspect.FileInspection
	require.NoError(t, ctx.ResponseJSON(&result))
	assert.True(t, result.Interchange)
	require.Len(t, result.Transactions, 1)
	assert.Equal(t, "204", result.Transactions[0].TransactionSet)
	assert.Equal(t, "000000042", result.Envelope.ISAControlNumber)

	var wire map[string]any
	require.NoError(t, ctx.ResponseJSON(&wire))
	for _, heavy := range []string{"rawX12", "formatted", "segments"} {
		assert.NotContains(
			t,
			wire,
			heavy,
			"the file is already on the card; only its overview is sent",
		)
	}
}

func TestEDIHandler_InspectInboundFileRefusesAPurgedFile(t *testing.T) {
	t.Parallel()

	fileID := pulid.MustNew("ediinf_")
	purgedAt := int64(1_760_000_000)
	files := edimessageportsmock.NewMockEDIInboundFileRepository(t)
	files.EXPECT().
		GetInboundFileByID(mock.Anything, mock.Anything).
		Return(&ediinboundfile.EDIInboundFile{ID: fileID, RawPurgedAt: &purgedAt}, nil).
		Once()

	edihandlertest.Run(
		t,
		setupInboundFileHandler(t, files),
		http.MethodGet,
		"/api/v1/edi/inbound-files/"+fileID.String()+"/inspect/",
		nil,
		nil,
		http.StatusBadRequest,
	)
}

func TestEDIHandler_InspectInboundFileRefusesAnIDThatIsNotOne(t *testing.T) {
	t.Parallel()

	edihandlertest.Run(
		t,
		setupInboundFileHandler(t, edimessageportsmock.NewMockEDIInboundFileRepository(t)),
		http.MethodGet,
		"/api/v1/edi/inbound-files/not-an-id/inspect/",
		nil,
		nil,
		http.StatusBadRequest,
	)
}
