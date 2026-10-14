package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInboundFiles struct {
	files  []*edi.EDIInboundFile
	listed *ediports.ListEDIInboundFilesRequest
}

func (f *fakeInboundFiles) ListInboundFiles(
	_ context.Context,
	req *ediports.ListEDIInboundFilesRequest,
) (*pagination.ListResult[*edi.EDIInboundFile], error) {
	f.listed = req

	return &pagination.ListResult[*edi.EDIInboundFile]{Items: f.files}, nil
}

func (f *fakeInboundFiles) GetInboundFile(
	context.Context,
	ediports.GetEDIInboundFileByIDRequest,
) (*edi.EDIInboundFile, error) {
	return f.files[0], nil
}

func inboundFile(raw string) *edi.EDIInboundFile {
	return &edi.EDIInboundFile{
		ID:            pulid.MustNew("eif_"),
		FileName:      "204_20260921.x12",
		Status:        edi.InboundFileStatusQuarantined,
		FailureReason: "ISA receiver does not match",
		RawContent:    raw,
		CommunicationProfile: &edi.EDICommunicationProfile{
			Name:             "Acme SFTP",
			EncryptedSecrets: map[string]string{"password": "hunter2-sealed"},
		},
		Messages: []*edi.EDIMessage{{
			ID:             pulid.MustNew("emsg_"),
			TransactionSet: "204",
			RawX12:         "ST*204*0001~",
		}},
	}
}

func TestGetEDIInboundFile_ReturnsRawX12OnlyWhenAskedAndAtRestricted(t *testing.T) {
	t.Parallel()

	raw := strings.Repeat("€", maxRawX12Bytes)
	file := inboundFile(raw)
	tool := newGetEDIInboundFileTool(
		&fakeInboundFiles{files: []*edi.EDIInboundFile{file}},
		&fakePermissions{},
	)
	id := file.ID.String()

	plain, err := tool.Query(t.Context(),
		agentParams(map[string]any{"inboundFileId": id}, permission.SensitivityRestricted))
	require.NoError(t, err)
	assert.Empty(t, plain.(*ediInboundFileView).Raw, "raw X12 is opt-in")

	internal, err := tool.Query(t.Context(),
		agentParams(map[string]any{"inboundFileId": id, "includeRaw": true}, ""))
	require.NoError(t, err)
	withheld := internal.(*ediInboundFileView)
	assert.Empty(t, withheld.Raw)
	assert.Equal(t, []string{"rawX12"}, withheld.Withheld)

	restricted, err := tool.Query(t.Context(), agentParams(
		map[string]any{"inboundFileId": id, "includeRaw": true},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)
	view := restricted.(*ediInboundFileView)
	assert.LessOrEqual(t, len(view.Raw), maxRawX12Bytes)
	assert.True(t, utf8.ValidString(view.Raw), "the cap never splits a character")
	assert.True(t, view.RawTruncated)
	assert.Equal(t, []agent.RecordRef{inboundFileRef(file.ID)}, view.TaintedRecords())
}

func TestGetEDIInboundFile_NeverCarriesTransportSecretsOrMessageBodies(t *testing.T) {
	t.Parallel()

	file := inboundFile("ISA*00*~")
	tool := newGetEDIInboundFileTool(
		&fakeInboundFiles{files: []*edi.EDIInboundFile{file}},
		&fakePermissions{},
	)

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{"inboundFileId": file.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "hunter2")
	assert.NotContains(t, string(encoded), "ST*204", "a message's raw body stays out")
	assert.Contains(t, string(encoded), "Acme SFTP")
	assert.Contains(t, string(encoded), `"title":"204_20260921.x12"`,
		"the Desk names the file's card by its file name")
}

func TestListEDIInboundFiles_MarksEveryFileItReturns(t *testing.T) {
	t.Parallel()

	first, second := inboundFile(""), inboundFile("")
	fake := &fakeInboundFiles{files: []*edi.EDIInboundFile{first, second}}
	tool := newListEDIInboundFilesTool(fake)

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"status": "Quarantined",
		"limit":  float64(1),
	}, ""))
	require.NoError(t, err)

	assert.Equal(t, edi.InboundFileStatusQuarantined, fake.listed.Status)
	outcome := result.(*gatedOutcome)
	assert.True(t, outcome.HasMore)
	assert.Equal(t, []agent.RecordRef{inboundFileRef(first.ID)}, outcome.TaintedRecords(),
		"only the file on the page is marked")
}

type fakeTransfers struct {
	transfers []*edi.EDITransfer
	direction string
	request   *ediports.ListEDITransfersRequest
}

func (f *fakeTransfers) ListInboundTransfersCursor(
	_ context.Context,
	req *ediports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	f.direction, f.request = transferInbound, req

	return &pagination.CursorListResult[*edi.EDITransfer]{
		Items:       f.transfers,
		HasNextPage: true,
	}, nil
}

func (f *fakeTransfers) ListOutboundTransfersCursor(
	_ context.Context,
	req *ediports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	f.direction, f.request = transferOutbound, req

	return &pagination.CursorListResult[*edi.EDITransfer]{Items: f.transfers}, nil
}

func TestListEDITransfers_FiltersAndHandsBackACursor(t *testing.T) {
	t.Parallel()

	transfer := &edi.EDITransfer{
		ID:        pulid.MustNew("edilt_"),
		Status:    edi.TransferStatusPendingApproval,
		CreatedAt: 1_700_000_000,
		TenderPayload: edi.LoadTenderPayload{
			BOL:   "BOL-1",
			Moves: []edi.LoadTenderMove{{Stops: make([]edi.LoadTenderStop, 2)}},
		},
	}
	fake := &fakeTransfers{transfers: []*edi.EDITransfer{transfer}}
	tool := newListEDITransfersTool(fake)

	fileID := pulid.MustNew("ediinf_")
	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"status":        "PendingApproval",
		"inboundFileId": fileID.String(),
	}, ""))
	require.NoError(t, err)

	assert.Equal(t, transferInbound, fake.direction)
	assert.Equal(t, fileID, fake.request.InboundFileID, "a file's own tenders")
	require.Len(t, fake.request.Filter.FieldFilters, 1)
	outcome := result.(ediCursorOutcome)
	rows := outcome.Items.([]ediTransferRow)
	require.Len(t, rows, 1)
	assert.Equal(t, 2, rows[0].StopCount)
	assert.True(t, outcome.HasMore)
	assert.NotEmpty(t, outcome.NextCursor)
	assert.Len(t, outcome.TaintedRecords(), 1)

	_, err = tool.Query(t.Context(), agentParams(map[string]any{"after": "not-a-cursor"}, ""))
	require.Error(t, err)
}

type fakePartners struct {
	partner *edi.EDIPartner
}

func (f *fakePartners) GetPartner(
	context.Context,
	ediports.GetEDIPartnerByIDRequest,
) (*edi.EDIPartner, error) {
	return f.partner, nil
}

func (f *fakePartners) GetPartnerReadiness(
	context.Context,
	*ediservice.GetEDIPartnerReadinessRequest,
) ([]*ediservice.EDIPartnerReadiness, error) {
	return []*ediservice.EDIPartnerReadiness{{
		PartnerID: f.partner.ID,
		Items: []ediservice.EDIPartnerReadinessItem{
			{Key: "mappings", Label: "Entity mappings defined"},
		},
	}}, nil
}

func TestGetEDIPartner_GivesReadinessWithoutTransportCredentials(t *testing.T) {
	t.Parallel()

	partner := &edi.EDIPartner{
		ID:                 pulid.MustNew("ep_"),
		Code:               "ACME",
		Name:               "Acme Logistics",
		ContactEmail:       "edi@acme.example",
		DefaultTransportID: pulid.MustNew("ecp_"),
		Settings:           map[string]any{"apiToken": "tok_live_secret"},
		DefaultTransport: &edi.EDICommunicationProfile{
			EncryptedSecrets: map[string]string{"password": "hunter2-sealed"},
		},
	}
	tool := newGetEDIPartnerTool(&fakePartners{partner: partner}, &fakePermissions{})

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{"partnerId": partner.ID.String()}, ""))
	require.NoError(t, err)

	view := result.(ediPartnerView)
	assert.True(t, view.HasTransport)
	assert.False(t, view.Ready)
	require.Len(t, view.Readiness, 1)
	assert.Empty(t, view.ContactEmail)
	assert.Contains(t, view.Withheld, "contactEmail")

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "hunter2")
	assert.NotContains(t, string(encoded), "tok_live_secret")
}

type fakeEDIMessages struct {
	message *edi.EDIMessage
	asked   ediports.GetEDIMessageByIDRequest
}

func (f *fakeEDIMessages) GetMessage(
	_ context.Context,
	req ediports.GetEDIMessageByIDRequest,
) (*edi.EDIMessage, error) {
	f.asked = req

	return f.message, nil
}

func deadLetteredInvoice(raw string) *edi.EDIMessage {
	sent := int64(1_760_000_000)
	return &edi.EDIMessage{
		ID:                       pulid.MustNew("edimsg_"),
		EDIPartnerID:             pulid.MustNew("edip_"),
		InvoiceID:                pulid.MustNew("inv_"),
		ShipmentID:               pulid.MustNew("shp_"),
		Direction:                edi.DocumentDirectionOutbound,
		TransactionSet:           edi.TransactionSet210,
		X12Version:               "004010",
		Status:                   edi.MessageStatusGenerated,
		InterchangeControlNumber: "000000209",
		GroupControlNumber:       "209",
		TransactionControlNumber: "0001",
		SegmentCount:             13,
		RawX12:                   raw,
		DeliveryStatus:           edi.MessageDeliveryStatusDeadLettered,
		DeliveryAttempts:         5,
		DeliveryLastAttemptAt:    &sent,
		DeliveryLastError:        "SFTP upload refused: 553 file name not allowed",
		DeliveryRemotePath:       "sftp://edi.acme.example/inbound/210",
		AckStatus:                edi.MessageAcknowledgmentStatusPending,
		GeneratedAt:              sent - 3600,
		PayloadSnapshot: edi.DocumentPayload{
			TransactionSet: edi.TransactionSet210,
			FreightInvoice: &edi.FreightInvoicePayload{
				InvoiceNumber: "INV-2610-0042",
				BOL:           "BOL-88213",
			},
		},
		Partner: &edi.EDIPartner{
			Code: "ACME",
			Name: "Acme Foods",
		},
		ValidationErrors: []*edi.EDIMessageValidationError{{
			Severity:        edi.ValidationSeverityError,
			Code:            "N4_POSTAL",
			SegmentID:       "N4",
			ElementPosition: 3,
			Path:            "N4[0].03",
			Message:         "Postal code is required",
			SuggestedFix:    "Add the bill-to postal code",
		}},
	}
}

func TestGetEDIMessage_ReadsTheDocumentItsDeliveryAndItsFindings(t *testing.T) {
	t.Parallel()

	message := deadLetteredInvoice("ISA*00*~")
	reader := &fakeEDIMessages{message: message}
	tool := newGetEDIMessageTool(reader, &fakePermissions{})
	params := agentParams(
		map[string]any{"messageId": message.ID.String()},
		permission.SensitivityRestricted,
	)

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, message.ID, reader.asked.ID)
	assert.Equal(t, params.OrganizationID, reader.asked.TenantInfo.OrgID)
	view := result.(*ediMessageView)
	assert.Equal(t, "210 to Acme Foods", view.Title)
	assert.Equal(t, "Acme Foods", view.Partner)
	assert.Equal(t, "ACME", view.PartnerCode)
	assert.Equal(t, "DeadLettered", view.DeliveryStatus)
	assert.Equal(t, int64(5), view.DeliveryAttempts)
	assert.Equal(t, "SFTP upload refused: 553 file name not allowed", view.DeliveryError)
	assert.Equal(t, "sftp://edi.acme.example/inbound/210", view.RemotePath)
	assert.Equal(t, message.InvoiceID.String(), view.InvoiceID)
	assert.Equal(t, "INV-2610-0042", view.Decoded.FreightInvoice.InvoiceNumber)
	assert.Equal(t, 1, view.DiagnosticCount)
	assert.Equal(t, []ediDiagnosticRow{{
		Severity:     "Error",
		Code:         "N4_POSTAL",
		Segment:      "N4",
		Element:      3,
		Path:         "N4[0].03",
		Message:      "Postal code is required",
		SuggestedFix: "Add the bill-to postal code",
	}}, view.Diagnostics)
	assert.Empty(t, view.Raw, "raw X12 is opt-in")
	assert.Equal(t,
		[]agent.RecordRef{{EntityType: ediMessageEntity, ID: message.ID.String()}},
		view.TaintedRecords(),
	)

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"ackReceivedAt":"no acknowledgment received"`)
	assert.Contains(t, string(encoded), `"title":"210 to Acme Foods"`)
}

func TestGetEDIMessage_NamesAnInboundDocumentByWhoSentIt(t *testing.T) {
	t.Parallel()

	message := deadLetteredInvoice("")
	message.Direction = edi.DocumentDirectionInbound
	message.TransactionSet = edi.TransactionSet997
	message.DeliveryStatus = ""
	message.AckStatus = edi.MessageAcknowledgmentStatusNotExpected
	message.PayloadSnapshot = edi.DocumentPayload{}
	tool := newGetEDIMessageTool(&fakeEDIMessages{message: message}, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{"messageId": message.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)

	view := result.(*ediMessageView)
	assert.Equal(t, "997 from Acme Foods", view.Title)
	assert.Nil(t, view.Decoded, "no decoded copy is said as none, not as an empty one")
	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(encoded),
		`"ackReceivedAt":"inbound documents are not acknowledged to us"`)
	assert.NotContains(t, string(encoded), `"decoded"`)

	message.Partner = nil
	orphan, err := tool.Query(t.Context(), agentParams(
		map[string]any{"messageId": message.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)
	assert.Equal(t, "997 inbound", orphan.(*ediMessageView).Title)
}

func TestGetEDIMessage_ReturnsRawX12OnlyWhenAskedAndAtRestricted(t *testing.T) {
	t.Parallel()

	raw := strings.Repeat("€", maxRawX12Bytes)
	message := deadLetteredInvoice(raw)
	tool := newGetEDIMessageTool(&fakeEDIMessages{message: message}, &fakePermissions{})
	asked := map[string]any{"messageId": message.ID.String(), "includeRaw": true}

	internal, err := tool.Query(t.Context(), agentParams(asked, ""))
	require.NoError(t, err)
	withheld := internal.(*ediMessageView)
	assert.Empty(t, withheld.Raw)
	assert.Equal(t, []string{"rawX12"}, withheld.Withheld)

	restricted, err := tool.Query(t.Context(), agentParams(asked, permission.SensitivityRestricted))
	require.NoError(t, err)
	view := restricted.(*ediMessageView)
	assert.LessOrEqual(t, len(view.Raw), maxRawX12Bytes)
	assert.True(t, utf8.ValidString(view.Raw), "the cap never splits a character")
	assert.True(t, view.RawTruncated)
	assert.Equal(t, fmt.Sprintf(
		"The raw X12 is %d bytes; only the first %d are shown.", len(raw), len(view.Raw),
	), view.Note)
}

func TestGetEDIMessage_SaysThePurgedX12IsGoneAndWhatRemains(t *testing.T) {
	t.Parallel()

	purged := int64(1_760_000_000)
	message := deadLetteredInvoice("")
	message.RawPurgedAt = &purged
	tool := newGetEDIMessageTool(&fakeEDIMessages{message: message}, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{"messageId": message.ID.String(), "includeRaw": true},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)

	view := result.(*ediMessageView)
	assert.Empty(t, view.Raw)
	assert.True(t, view.RawPurged)
	assert.Equal(t, "The raw X12 was purged under the retention policy; the decoded "+
		"document and its findings remain.", view.Note)
}

func TestGetEDIMessage_CapsTheFindingsAndCountsTheRest(t *testing.T) {
	t.Parallel()

	message := deadLetteredInvoice("")
	finding := message.ValidationErrors[0]
	message.ValidationErrors = make([]*edi.EDIMessageValidationError, 0, maxMessageDiagnostics+7)
	for range maxMessageDiagnostics + 7 {
		message.ValidationErrors = append(message.ValidationErrors, finding)
	}
	tool := newGetEDIMessageTool(&fakeEDIMessages{message: message}, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{"messageId": message.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)

	view := result.(*ediMessageView)
	assert.Equal(t, maxMessageDiagnostics+7, view.DiagnosticCount)
	assert.Len(t, view.Diagnostics, maxMessageDiagnostics)
	assert.Equal(t, 7, view.DiagnosticsOmitted)
}
