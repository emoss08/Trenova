package agentquerytoolservice

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInboundFiles struct {
	files  []*edi.EDIInboundFile
	listed *repositories.ListEDIInboundFilesRequest
}

func (f *fakeInboundFiles) ListInboundFiles(
	_ context.Context,
	req *repositories.ListEDIInboundFilesRequest,
) (*pagination.ListResult[*edi.EDIInboundFile], error) {
	f.listed = req

	return &pagination.ListResult[*edi.EDIInboundFile]{Items: f.files}, nil
}

func (f *fakeInboundFiles) GetInboundFile(
	context.Context,
	repositories.GetEDIInboundFileByIDRequest,
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
	request   *repositories.ListEDITransfersRequest
}

func (f *fakeTransfers) ListInboundTransfersCursor(
	_ context.Context,
	req *repositories.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	f.direction, f.request = transferInbound, req

	return &pagination.CursorListResult[*edi.EDITransfer]{
		Items:       f.transfers,
		HasNextPage: true,
	}, nil
}

func (f *fakeTransfers) ListOutboundTransfersCursor(
	_ context.Context,
	req *repositories.ListEDITransfersRequest,
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

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{"status": "PendingApproval"}, ""))
	require.NoError(t, err)

	assert.Equal(t, transferInbound, fake.direction)
	require.Len(t, fake.request.Filter.FieldFilters, 1)
	outcome := result.(ediTransferOutcome)
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
	repositories.GetEDIPartnerByIDRequest,
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
