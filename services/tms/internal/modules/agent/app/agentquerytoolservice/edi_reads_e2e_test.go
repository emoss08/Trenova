package agentquerytoolservice

import (
	"context"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ediFakes records what the screen's resolvers asked of the EDI
// repositories a test's reads reach.
type ediFakes struct {
	mu        sync.Mutex
	org       pulid.ID
	partners  *ediports.ListEDIPartnersRequest
	files     *ediports.ListEDIInboundFilesRequest
	messages  *ediports.ListEDIMessagesRequest
	transfers *ediports.ListEDITransfersRequest
	outbound  bool
	invoices  *ediports.ListEDICarrierInvoicesRequest
	counts    ediports.GetEDIMessageStatusCountsRequest
	identity  *carrierports.FindCarrierByIdentityRequest
}

type fakeEDIPartnerRepo struct {
	ediports.EDIPartnerRepository
	*ediFakes
}

type fakeEDIInboundFileRepo struct {
	ediports.EDIInboundFileRepository
	*ediFakes
}

type fakeEDIMessageRepo struct {
	ediports.EDIMessageRepository
	*ediFakes
}

type fakeEDITransferRepo struct {
	ediports.EDILoadTenderTransferRepository
	*ediFakes
}

type fakeEDICarrierInvoiceRepo struct {
	ediports.EDICarrierInvoiceRepository
	*ediFakes
}

type fakeEDICarrierRepo struct {
	carrierports.CarrierRepository
	*ediFakes
}

var ediPartnerFixture = pulid.MustNew("edip_")

func (f *fakeEDIPartnerRepo) ListCursor(
	_ context.Context,
	req *ediports.ListEDIPartnersRequest,
) (*pagination.CursorListResult[*edi.EDIPartner], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.partners = req

	return withEdges(&pagination.CursorListResult[*edi.EDIPartner]{
		Items: []*edi.EDIPartner{{
			ID: ediPartnerFixture, Code: "ACME", Name: "Acme Foods",
			Kind: edi.PartnerKindExternal, Status: "Active", Country: "US",
			ContactEmail: "edi@acme.example", EnabledForInbound: true,
		}},
		HasNextPage: true,
	}), nil
}

func (f *fakeEDIInboundFileRepo) ListInboundFilesCursor(
	_ context.Context,
	req *ediports.ListEDIInboundFilesRequest,
) (*pagination.CursorListResult[*edi.EDIInboundFile], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files = req

	return withEdges(&pagination.CursorListResult[*edi.EDIInboundFile]{
		Items: []*edi.EDIInboundFile{inboundFileFixture("ISA*00~")},
	}), nil
}

func (f *fakeEDIInboundFileRepo) GetInboundFileByID(
	_ context.Context,
	req ediports.GetEDIInboundFileByIDRequest,
) (*edi.EDIInboundFile, error) {
	file := inboundFileFixture("ISA*00*RAW~")
	file.ID = req.ID

	return file, nil
}

func inboundFileFixture(raw string) *edi.EDIInboundFile {
	messages := make([]*edi.EDIMessage, 0, 52)
	for range 52 {
		messages = append(messages, &edi.EDIMessage{
			ID: pulid.MustNew("edimsg_"), TransactionSet: edi.TransactionSet204,
			Direction: edi.DocumentDirectionInbound, RawX12: "ST*204*0001~",
		})
	}

	return &edi.EDIInboundFile{
		ID:            pulid.MustNew("ediinf_"),
		FileName:      "204_20260921.x12",
		Method:        edi.ConnectionMethodSFTP,
		Status:        edi.InboundFileStatusQuarantined,
		FailureReason: "ISA receiver does not match",
		RawContent:    raw,
		ReceivedAt:    1_790_000_000,
		CommunicationProfile: &edi.EDICommunicationProfile{
			Name:             "Acme SFTP",
			EncryptedSecrets: map[string]string{"password": "hunter2-sealed"},
		},
		Messages: messages,
	}
}

func (f *fakeEDIMessageRepo) ListMessagesCursor(
	_ context.Context,
	req *ediports.ListEDIMessagesRequest,
) (*pagination.CursorListResult[*edi.EDIMessage], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = req

	return withEdges(&pagination.CursorListResult[*edi.EDIMessage]{
		Items: []*edi.EDIMessage{messageFixture()},
	}), nil
}

func (f *fakeEDIMessageRepo) GetMessageByID(
	_ context.Context,
	req ediports.GetEDIMessageByIDRequest,
) (*edi.EDIMessage, error) {
	message := messageFixture()
	message.ID = req.ID
	for range 55 {
		message.ValidationErrors = append(message.ValidationErrors,
			&edi.EDIMessageValidationError{
				Severity: "Error", Code: "E101", SegmentID: "B10", ElementPosition: 2,
				Message: "Reference is missing",
			})
	}

	return message, nil
}

func messageFixture() *edi.EDIMessage {
	return &edi.EDIMessage{
		ID:                       pulid.MustNew("edimsg_"),
		EDIPartnerID:             ediPartnerFixture,
		Direction:                edi.DocumentDirectionOutbound,
		TransactionSet:           edi.TransactionSet214,
		Status:                   edi.MessageStatusGenerated,
		RawX12:                   "ISA*00*SECRET~ST*214*0001~",
		DeliveryStatus:           edi.MessageDeliveryStatusDeadLettered,
		DeliveryLastError:        "connection refused",
		AckStatus:                edi.MessageAcknowledgmentStatusPending,
		TransactionControlNumber: "0001",
		GeneratedAt:              timeutils.NowUnix() - 7200,
		PayloadSnapshot:          edi.DocumentPayload{TransactionSet: edi.TransactionSet214},
	}
}

func (f *fakeEDIMessageRepo) GetAckStatusCounts(
	_ context.Context,
	req ediports.GetEDIMessageStatusCountsRequest,
) (map[edi.MessageAcknowledgmentStatus]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counts = req

	return map[edi.MessageAcknowledgmentStatus]int{
		edi.MessageAcknowledgmentStatusPending:  2,
		edi.MessageAcknowledgmentStatusAccepted: 5,
		edi.MessageAcknowledgmentStatusRejected: 1,
	}, nil
}

func (f *fakeEDIMessageRepo) GetAckStatusCountsByPartner(
	context.Context,
	ediports.GetEDIMessageStatusCountsRequest,
) ([]*ediports.EDIPartnerAckStatusCount, error) {
	return []*ediports.EDIPartnerAckStatusCount{
		{PartnerID: ediPartnerFixture, PartnerName: "Acme", PartnerCode: "ACME",
			AckStatus: edi.MessageAcknowledgmentStatusAccepted, Count: 5},
		{PartnerID: ediPartnerFixture, PartnerName: "Acme", PartnerCode: "ACME",
			AckStatus: edi.MessageAcknowledgmentStatusPending, Count: 2},
	}, nil
}

func (f *fakeEDIMessageRepo) GetOverdueAckCount(
	context.Context,
	ediports.GetEDIOverdueAckCountRequest,
) (int, error) {
	return 3, nil
}

func (f *fakeEDIMessageRepo) ListOverdueAcks(
	context.Context,
	*ediports.ListOverdueEDIAcksRequest,
) (*ediports.OverdueEDIAcks, error) {
	message := messageFixture()

	return &ediports.OverdueEDIAcks{
		Messages: []*edi.EDIMessage{message},
		Partners: []*ediports.OverdueEDIAckPartner{{
			PartnerID: ediPartnerFixture, PartnerName: "Acme", PartnerCode: "ACME",
			Count: 3, OldestGeneratedAt: message.GeneratedAt,
		}},
	}, nil
}

func (f *fakeEDIMessageRepo) ListRecentDeadLettered(
	context.Context,
	*ediports.ListRecentEDIMessageFailuresRequest,
) ([]*edi.EDIMessage, error) {
	return []*edi.EDIMessage{messageFixture()}, nil
}

func (f *ediFakes) listTransfers(
	req *ediports.ListEDITransfersRequest,
	outbound bool,
) *pagination.CursorListResult[*edi.EDITransfer] {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.transfers, f.outbound = req, outbound

	return withEdges(&pagination.CursorListResult[*edi.EDITransfer]{
		Items: []*edi.EDITransfer{tenderFixture(f.org)},
	})
}

func (f *fakeEDITransferRepo) ListInboundCursor(
	_ context.Context,
	req *ediports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return f.listTransfers(req, false), nil
}

func (f *fakeEDITransferRepo) ListOutboundCursor(
	_ context.Context,
	req *ediports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return f.listTransfers(req, true), nil
}

func (f *fakeEDITransferRepo) GetTransferByID(
	_ context.Context,
	req ediports.GetEDITransferByIDRequest,
) (*edi.EDITransfer, error) {
	tender := tenderFixture(pulid.MustNew("org_"))
	tender.ID = req.ID
	tender.Status = edi.TransferStatusApproved

	return tender, nil
}

// tenderFixture is a load tender another organization sent to org.
func tenderFixture(org pulid.ID) *edi.EDITransfer {
	start, end := int64(1_790_000_000), int64(1_790_007_200)

	return &edi.EDITransfer{
		ID:                   pulid.MustNew("edilt_"),
		SourceOrganizationID: pulid.MustNew("org_"),
		TargetOrganizationID: org,
		Status:               edi.TransferStatusPendingApproval,
		TargetPartner:        &edi.EDIPartner{ID: ediPartnerFixture, Code: "ACME", Name: "Acme"},
		TenderPayload: edi.LoadTenderPayload{
			BOL:               "BOL-5",
			CustomerLabel:     "Acme Foods",
			TotalChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("1850.00")),
			Moves: []edi.LoadTenderMove{{Stops: []edi.LoadTenderStop{
				{
					Type: "Pickup", LocationName: "Acme DC", LocationCity: "Dallas",
					ScheduledWindowStart: start, ScheduledWindowEnd: &end,
				},
				{Type: "Delivery", LocationName: "Store 12", LocationCity: "Tulsa"},
			}}},
			Commodities: []edi.LoadTenderCommodity{{CommodityLabel: "Produce", Weight: 40000}},
		},
	}
}

func (f *fakeEDICarrierInvoiceRepo) ListCarrierInvoices(
	_ context.Context,
	req *ediports.ListEDICarrierInvoicesRequest,
) (*pagination.ListResult[*edi.CarrierInvoice], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.invoices = req
	total := decimal.NewNullDecimal(decimal.RequireFromString("1200.00"))
	items := []*edi.CarrierInvoice{
		{
			ID: pulid.MustNew("edici_"), InvoiceNumber: "INV-1", TotalAmount: total,
			ReconciliationStatus: edi.CarrierInvoiceReconciliationStatusMappingRequired,
			ReferenceNumbers:     map[string]string{"SCAC": "ACME"},
		},
		{
			ID: pulid.MustNew("edici_"), InvoiceNumber: "INV-2", TotalAmount: total,
			CarrierID:            pulid.MustNew("car_"),
			ReconciliationStatus: edi.CarrierInvoiceReconciliationStatusUnmatched,
		},
	}
	if limit := req.Filter.Pagination.Limit; limit > 0 && limit < len(items) {
		items = items[:limit]
	}

	return &pagination.ListResult[*edi.CarrierInvoice]{
		Items:      items,
		Total:      len(items),
		TotalExact: true,
	}, nil
}

func (f *fakeEDICarrierRepo) FindByIdentity(
	_ context.Context,
	req *carrierports.FindCarrierByIdentityRequest,
) (*carrier.Carrier, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.identity = req

	return &carrier.Carrier{ID: pulid.MustNew("car_"), Name: "Acme Carriers", SCAC: "ACME"}, nil
}

func ediReader(
	t *testing.T,
	fakes *ediFakes,
	engine permissionports.PermissionEngine,
) agentports.GraphQLReader {
	t.Helper()

	logger := zap.NewNop()
	files := &fakeEDIInboundFileRepo{ediFakes: fakes}
	edis := ediservice.New(ediservice.Params{
		Logger:          logger,
		PartnerRepo:     &fakeEDIPartnerRepo{ediFakes: fakes},
		MessageRepo:     &fakeEDIMessageRepo{ediFakes: fakes},
		TransferRepo:    &fakeEDITransferRepo{ediFakes: fakes},
		InboundFileRepo: files,
	})

	return agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: engine,
		Resolvers: agentread.ResolverParams{
			EDIService: edis,
			EDIInboundService: ediinboundservice.New(ediinboundservice.Params{
				Logger: logger, InboundFileRepo: files, EDIService: edis,
			}),
			CarrierSettlementService: carriersettlementservice.New(carriersettlementservice.Params{
				Logger:         logger,
				EDIInvoiceRepo: &fakeEDICarrierInvoiceRepo{ediFakes: fakes},
				CarrierRepo:    &fakeEDICarrierRepo{ediFakes: fakes},
			}),
		},
	})
}

func ediTool(t *testing.T, name string, reader agentports.GraphQLReader) agentports.AgentQueryTool {
	t.Helper()

	engine := &mocks.DenyResourcesPermissionEngine{}
	for _, spec := range ediReadSpecs() {
		if spec.name == name {
			tool, err := newGraphQLReadTool(&spec, reader, engine)
			require.NoError(t, err)
			return tool
		}
	}
	for _, spec := range ediGetSpecs() {
		if spec.name == name {
			tool, err := newGraphQLGetTool(&spec, reader, engine)
			require.NoError(t, err)
			return tool
		}
	}
	require.FailNow(t, "no spec "+name)

	return nil
}

func ediCall(params map[string]any) (context.Context, *agentports.QueryToolParams) {
	call := keyedCall(params)
	call.DataAccessCeiling = permission.SensitivityInternal

	return readscope.With(context.Background(), nil), call
}

func listItems(t *testing.T, data any) []map[string]any {
	t.Helper()

	switch outcome := data.(type) {
	case graphQLListOutcome:
		return outcome.Items
	case *besidePage:
		return outcome.page.Items
	case *offsetPage:
		return outcome.items()
	case map[string]any:
		raw, _ := outcome[outcomeItems].([]map[string]any)
		if raw != nil {
			return raw
		}
		list, _ := outcome[outcomeItems].([]any)
		out := make([]map[string]any, 0, len(list))
		for _, row := range list {
			out = append(out, row.(map[string]any))
		}
		return out
	}
	require.FailNow(t, "not a list outcome")

	return nil
}

// The partner list narrows by a customer and a status through the screen's
// table filters, and never shows a contact's email at the default data
// access.
func TestEDIPartners_ThroughTheExecutor(t *testing.T) {
	t.Parallel()

	fakes := &ediFakes{}
	tool := ediTool(t, "edi_list_partners",
		ediReader(t, fakes, &mocks.DenyResourcesPermissionEngine{}))
	customer := pulid.MustNew("cus_")
	ctx, call := ediCall(map[string]any{
		"customerId": customer.String(), "status": []any{"Active"}, "query": "acme",
	})
	data, err := tool.Query(ctx, call)
	require.NoError(t, err)

	require.NotNil(t, fakes.partners)
	assert.Equal(t, "acme", fakes.partners.Filter.Query)
	fields := map[string]any{}
	for _, filter := range fakes.partners.Filter.FieldFilters {
		fields[filter.Field] = filter.Value
	}
	assert.Equal(t, customer.String(), fields["customerId"])
	items := listItems(t, data)
	require.Len(t, items, 1)
	assert.Equal(t, "ACME", items[0]["code"])
	assert.NotContains(t, items[0], "contactEmail")
	outcome := data.(graphQLListOutcome)
	assert.True(t, outcome.HasMore)
	assert.NotEmpty(t, outcome.NextCursor)

	_, err = tool.Query(ctx, keyedCall(map[string]any{"contactEmail": "edi@acme.example"}))
	require.ErrorContains(t, err, "hidden at your access")
}

// Inbound files narrow by partner and status, the tender list reads Inbound
// unless a call names a direction and narrows by the file the tenders came
// in, and the message list bounds its creation window to the minute.
func TestEDILists_SendTheirTypedFiltersToTheScreensResolver(t *testing.T) {
	t.Parallel()

	fakes := &ediFakes{}
	reader := ediReader(t, fakes, &mocks.DenyResourcesPermissionEngine{})

	files := ediTool(t, "edi_list_inbound_files", reader)
	ctx, call := ediCall(map[string]any{
		"partnerId": ediPartnerFixture.String(), "status": []any{"Quarantined"},
	})
	data, err := files.Query(ctx, call)
	require.NoError(t, err)
	require.NotNil(t, fakes.files)
	assert.Len(t, fakes.files.Filter.FieldFilters, 2)
	items := listItems(t, data)
	require.Len(t, items, 1)
	assert.Equal(t, "ISA receiver does not match", items[0]["failureReason"])
	assert.NotContains(t, items[0], "rawContent")

	transfers := ediTool(t, "edi_list_transfers", reader)
	inboundFile := pulid.MustNew("ediinf_")
	fakes.org = call.OrganizationID
	ctx, call = ediCall(map[string]any{"inboundFileId": inboundFile.String()})
	fakes.org = call.OrganizationID
	data, err = transfers.Query(ctx, call)
	require.NoError(t, err)
	require.NotNil(t, fakes.transfers)
	assert.False(t, fakes.outbound, "a call that names no direction reads the inbound tenders")
	assert.Equal(t, inboundFile, fakes.transfers.InboundFileID)
	items = listItems(t, data)
	require.Len(t, items, 1)
	assert.Equal(t, "Inbound", items[0]["direction"])
	assert.Equal(t, "BOL-5", items[0]["bol"])
	assert.EqualValues(t, 2, items[0]["stopCount"])
	assert.NotContains(t, items[0], "tenderPayload")

	ctx, call = ediCall(map[string]any{"direction": "Outbound"})
	_, err = transfers.Query(ctx, call)
	require.NoError(t, err)
	assert.True(t, fakes.outbound)

	messages := ediTool(t, "edi_list_messages", reader)
	invoice := pulid.MustNew("inv_")
	ctx, call = ediCall(map[string]any{
		"invoiceId":     invoice.String(),
		"ackStatus":     []any{"Rejected"},
		"createdAtFrom": "2026-10-01T08:30:00-05:00",
	})
	data, err = messages.Query(ctx, call)
	require.NoError(t, err)
	require.NotNil(t, fakes.messages)
	fields := map[string]any{}
	for _, filter := range fakes.messages.Filter.FieldFilters {
		fields[filter.Field] = filter.Value
	}
	assert.Equal(t, invoice.String(), fields["invoiceId"])
	assert.EqualValues(t, 1_790_861_400, fields["createdAt"],
		"an instant is bounded to the second it names")
	items = listItems(t, data)
	require.Len(t, items, 1)
	assert.Equal(t, "connection refused", items[0]["deliveryLastError"])
	assert.NotContains(t, items[0], "rawX12")

	ctx, call = ediCall(map[string]any{"createdAtFrom": "2026-10-01"})
	_, err = messages.Query(ctx, call)
	require.ErrorContains(t, err, "date and time with its offset")
}

// A carrier invoice no carrier is linked to carries the carrier its SCAC
// points to, its amounts stay withheld below Restricted, and the list pages
// by the root's limit and offset.
func TestEDICarrierInvoices_ThroughTheExecutor(t *testing.T) {
	t.Parallel()

	fakes := &ediFakes{}
	tool := ediTool(t, "edi_list_carrier_invoices",
		ediReader(t, fakes, &mocks.DenyResourcesPermissionEngine{}))
	ctx, call := ediCall(map[string]any{"status": "MappingRequired", "limit": float64(1)})
	data, err := tool.Query(ctx, call)
	require.NoError(t, err)

	require.NotNil(t, fakes.invoices)
	assert.Equal(t, edi.CarrierInvoiceReconciliationStatusMappingRequired,
		fakes.invoices.ReconciliationStatus)
	require.NotNil(t, fakes.identity)
	assert.Equal(t, "ACME", fakes.identity.SCAC)
	outcome := data.(*offsetPage).record
	items := listItems(t, data)
	require.Len(t, items, 1)
	suggested, _ := items[0]["suggestedCarrier"].(map[string]any)
	assert.Equal(t, "Acme Carriers", suggested["name"])
	assert.NotContains(t, items[0], "totalAmount")
	assert.Contains(t, outcome[outcomeWithheld], "totalAmount")
	assert.Equal(t, true, outcome[outcomeHasMore])
	assert.NotEmpty(t, outcome[outcomeNextCursor])

	ctx, call = ediCall(map[string]any{"status": "Lost"})
	_, err = tool.Query(ctx, call)
	require.ErrorContains(t, err, "Reconciliation status must be")
}

// One file, one message and one tender read their details through the
// screen's resolvers: the parsed sets capped at fifty, every finding, the
// stops and what accepting a tender waits on, with the raw X12 and the rate
// withheld at the default data access and the raw read only on request.
func TestEDIGets_ThroughTheExecutor(t *testing.T) {
	t.Parallel()

	fakes := &ediFakes{}
	reader := ediReader(t, fakes, &mocks.DenyResourcesPermissionEngine{})

	file := ediTool(t, "edi_get_inbound_file", reader)
	ctx, call := ediCall(map[string]any{
		"inboundFileId": pulid.MustNew("ediinf_").String(), "includeRaw": true,
	})
	data, err := file.Query(ctx, call)
	require.NoError(t, err)
	record := data.(map[string]any)
	assert.EqualValues(t, 52, record["messageCount"])
	assert.Len(t, record["messages"], 50)
	assert.Equal(t, "Acme SFTP", record["communicationProfileName"])
	assert.NotContains(t, record, "raw")
	assert.Contains(t, record[outcomeWithheld], "raw")
	assert.NotContains(t, sonicString(t, record), "hunter2")

	message := ediTool(t, "edi_get_message", reader)
	ctx, call = ediCall(map[string]any{"messageId": pulid.MustNew("edimsg_").String()})
	data, err = message.Query(ctx, call)
	require.NoError(t, err)
	record = data.(map[string]any)
	assert.EqualValues(t, 55, record["diagnosticCount"])
	assert.Len(t, record["diagnostics"], 50)
	assert.Contains(t, record, "decoded")
	assert.NotContains(t, record, "raw")
	assert.NotContains(t, sonicString(t, record), "SECRET")

	transfer := ediTool(t, "edi_get_transfer", reader)
	ctx, call = ediCall(map[string]any{"transferId": pulid.MustNew("edilt_").String()})
	data, err = transfer.Query(ctx, call)
	require.NoError(t, err)
	record = data.(map[string]any)
	tender, _ := record["transfer"].(map[string]any)
	require.NotNil(t, tender)
	assert.Len(t, tender["stops"], 2)
	assert.Equal(t, "Outbound", tender["direction"],
		"a tender this organization did not receive was sent by it")
	assert.Equal(t, false, record["canAccept"])
	assert.NotContains(t, tender, "charges")
	assert.Contains(t, record[outcomeWithheld], "transfer.charges")
}

// A person whose role reaches Restricted reads the raw X12 when asked for
// it, cut to 16 KB, and a message's raw text is never read unless asked.
func TestEDIGets_ReadTheRawX12OnlyWhenAskedAndAtRestricted(t *testing.T) {
	t.Parallel()

	fakes := &ediFakes{}
	reader := ediReader(t, fakes, &restrictedRole{})
	message := ediTool(t, "edi_get_message", reader)

	ctx, call := ediCall(map[string]any{
		"messageId": pulid.MustNew("edimsg_").String(), "includeRaw": true,
	})
	call.DataAccessCeiling = permission.SensitivityRestricted
	data, err := message.Query(ctx, call)
	require.NoError(t, err)
	raw, _ := data.(map[string]any)["raw"].(map[string]any)
	require.NotNil(t, raw)
	assert.Equal(t, "ISA*00*SECRET~ST*214*0001~", raw["text"])

	ctx, call = ediCall(map[string]any{"messageId": pulid.MustNew("edimsg_").String()})
	call.DataAccessCeiling = permission.SensitivityRestricted
	data, err = message.Query(ctx, call)
	require.NoError(t, err)
	assert.NotContains(t, data.(map[string]any), "raw")
	withheld, _ := data.(map[string]any)[outcomeWithheld].([]string)
	assert.NotContains(t, withheld, "raw")
}

// Acknowledgment health reads the window and partner a call names, says
// what came back for each partner and how long each overdue message has
// waited, and reads a 997 as every set it answers.
func TestEDIAckHealth_ThroughTheExecutor(t *testing.T) {
	t.Parallel()

	fakes := &ediFakes{}
	tool := ediTool(t, "edi_get_ack_health",
		ediReader(t, fakes, &mocks.DenyResourcesPermissionEngine{}))
	ctx, call := ediCall(map[string]any{
		"partnerId": ediPartnerFixture.String(), "windowHours": float64(48),
		"transactionSet": "997",
	})
	data, err := tool.Query(ctx, call)
	require.NoError(t, err)

	assert.Equal(t, ediPartnerFixture, fakes.counts.PartnerID)
	assert.Empty(t, fakes.counts.TransactionSet, "a 997 counts every set it answers")
	assert.InDelta(t, timeutils.NowUnix()-48*3600, fakes.counts.Since, 5)
	record := data.(map[string]any)
	assert.EqualValues(t, 6, record["acknowledged"])
	assert.EqualValues(t, 2, record["awaiting"])
	assert.Equal(t, true, record["answersEverySet"])
	assert.Contains(t, record[outcomeNote], "acknowledgments themselves")
	partners, _ := record["partners"].([]any)
	require.Len(t, partners, 1)
	assert.Equal(t, "5 of 7 came back with a 997 or 999 (5 accepted); 2 still awaiting one",
		partners[0].(map[string]any)["answer"])
	groups, _ := record["overdueByPartner"].([]any)
	require.Len(t, groups, 1)
	group := groups[0].(map[string]any)
	assert.EqualValues(t, 2, group["notListed"])
	assert.Equal(t, "2h 00m", group["oldestWaiting"])
	assert.Empty(t, record[outcomeWithheld])

	ctx, call = ediCall(map[string]any{"overdueAfterSeconds": float64(5)})
	_, err = tool.Query(ctx, call)
	require.ErrorContains(t, err, "between 60 and 2592000")
}

// An agent or a person who may not read EDI is refused before any EDI
// record is read, as authorize refuses them.
func TestEDIReads_AReaderWhoCannotReadEDIIsRefused(t *testing.T) {
	t.Parallel()

	fakes := &ediFakes{}
	engine := &mocks.DenyResourcesPermissionEngine{Denied: []string{permission.ResourceEDI.String()}}
	reader := ediReader(t, fakes, engine)
	for name, params := range map[string]map[string]any{
		"edi_list_partners":  {},
		"edi_get_message":    {"messageId": pulid.MustNew("edimsg_").String()},
		"edi_get_ack_health": {},
	} {
		ctx, call := ediCall(params)
		_, err := ediTool(t, name, reader).Query(ctx, call)
		var refused *agentports.ReadRefusedError
		require.ErrorAs(t, err, &refused, name)
	}
	assert.Nil(t, fakes.partners)
}

func sonicString(t *testing.T, value any) string {
	t.Helper()

	encoded, err := sonic.MarshalString(value)
	require.NoError(t, err)

	return encoded
}

// restrictedRole is a person whose role reaches Restricted fields.
type restrictedRole struct {
	mocks.DenyResourcesPermissionEngine
}

func (restrictedRole) GetResourcePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
) (*permissionports.ResourcePermissionDetail, error) {
	return &permissionports.ResourcePermissionDetail{
		MaxSensitivity: permission.SensitivityRestricted,
	}, nil
}
