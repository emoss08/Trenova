//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicarrierinvoicerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediinboundfilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimappingprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimessagerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnerrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferrepository"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice/edicarrierinvoicedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile/ediinboundfiledb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer/editransferdb"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementcostservice"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/ediservicetest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ediDocumentType is the outbound 214 the migrations seed for every tenant.
const ediDocumentType = "edidt_x12_214_outbound"

// ediTenant is one tenant's EDI rows: its partners, the files they sent, the
// messages exchanged, the tenders received and the carriers' invoices.
type ediTenant struct {
	ctx       context.Context
	partners  []string
	files     []string
	messages  []string
	transfers []string
	invoices  []string
}

func writeEDIRows(
	t *testing.T,
	admin *keel.DB,
	tenant *agentreadtest.Tenant,
	label string,
) *ediTenant {
	t.Helper()

	ctx := t.Context()
	org, bu := tenant.Actor.OrganizationID, tenant.Actor.BusinessUnitID
	ctx = tenant.Ctx(ctx)
	must := func(err error) {
		t.Helper()
		require.NoError(t, err)
	}
	now := timeutils.NowUnix()
	out := &ediTenant{ctx: readscope.With(dbscope.WithTenant(t.Context(), tenant.Scope()), nil)}

	var partners []*edi.EDIPartner
	for idx := range 3 {
		partner := &edi.EDIPartner{
			ID: pulid.MustNew("edip_"), OrganizationID: org, BusinessUnitID: bu,
			Kind: edi.PartnerKindExternal, Status: "Active", Country: "US",
			Code:              fmt.Sprintf("%s-P%d", label, idx+1),
			Name:              fmt.Sprintf("Partner %s %d", label, idx+1),
			ContactEmail:      "edi@" + label + ".example",
			EnabledForInbound: true, Settings: map[string]any{},
		}
		must(edidb.EDIPartners.Create(ctx, admin, partner))
		partners = append(partners, partner)
		out.partners = append(out.partners, partner.ID.String())
	}

	profile := &edi.EDICommunicationProfile{
		ID: pulid.MustNew("edicp_"), OrganizationID: org, BusinessUnitID: bu,
		EDIPartnerID: partners[0].ID, Method: edi.ConnectionMethodSFTP, Status: "Active",
		Name: "SFTP " + label, Config: map[string]any{"host": "sftp." + label + ".example"},
		EncryptedSecrets: map[string]string{},
	}
	must(edidb.EDICommunicationProfiles.Create(ctx, admin, profile))

	var files []*ediinboundfile.EDIInboundFile
	for idx := range 3 {
		file := &ediinboundfile.EDIInboundFile{
			ID: pulid.MustNew("ediinf_"), OrganizationID: org, BusinessUnitID: bu,
			CommunicationProfileID: profile.ID, EDIPartnerID: partners[0].ID,
			Method: edi.ConnectionMethodSFTP, RemotePath: "/in",
			FileName:  fmt.Sprintf("%s_204_%d.x12", label, idx+1),
			Checksum:  pulid.MustNew("sum_").String(),
			SizeBytes: 12, RawContent: "ISA*00*RAW~",
			Status:     ediinboundfile.InboundFileStatusProcessed,
			ReceivedAt: now - int64(idx),
		}
		if idx == 0 {
			file.Status = ediinboundfile.InboundFileStatusQuarantined
		}
		must(ediinboundfiledb.EDIInboundFiles.Create(ctx, admin, file))
		files = append(files, file)
		out.files = append(out.files, file.ID.String())
	}

	var messages []*edimessage.EDIMessage
	for idx := range 3 {
		message := &edimessage.EDIMessage{
			ID: pulid.MustNew("edimsg_"), OrganizationID: org, BusinessUnitID: bu,
			EDIPartnerID: partners[0].ID, DocumentTypeID: ediDocumentType,
			Direction: edidocument.DocumentDirectionOutbound, Standard: edidocument.EDIStandardX12,
			TransactionSet: edidocument.TransactionSet214, X12Version: "004010",
			Status:                   edimessage.MessageStatusGenerated,
			InterchangeControlNumber: fmt.Sprintf("%09d", idx+1),
			GroupControlNumber:       "1", TransactionControlNumber: "0001",
			RawX12: "ISA*00*SECRET~",
			PayloadSnapshot: edipayload.DocumentPayload{
				TransactionSet: edidocument.TransactionSet214,
			},
			DeliveryStatus: edimessage.MessageDeliveryStatusSent,
			AckStatus:      edimessage.MessageAcknowledgmentStatusPending,
			GeneratedAt:    now - int64(idx),
		}
		if idx == 0 {
			message.DeliveryStatus = edimessage.MessageDeliveryStatusDeadLettered
		}
		must(edimessagedb.EDIMessages.Create(ctx, admin, message))
		messages = append(messages, message)
		out.messages = append(out.messages, message.ID.String())
	}

	for idx := range 3 {
		transfer := &editransfer.EDITransfer{
			ID:                   pulid.MustNew("edilt_"),
			SourceOrganizationID: org, SourceBusinessUnitID: bu,
			TargetOrganizationID: org, TargetBusinessUnitID: bu,
			SourcePartnerID: partners[1].ID, TargetPartnerID: partners[0].ID,
			InboundMessageID: messages[0].ID,
			Status:           editransfer.TransferStatusPendingApproval,
			TenderPayload: editransfer.LoadTenderPayload{
				BOL: fmt.Sprintf("%s-BOL-%d", label, idx+1),
				Moves: []editransfer.LoadTenderMove{{Stops: []editransfer.LoadTenderStop{
					{Type: "Pickup", LocationName: "DC " + label},
				}}},
			},
			MappingSnapshot: []editransfer.MappingResolution{},
			SubmittedAt:     now - int64(idx),
		}
		must(editransferdb.EDITransfers.Create(ctx, admin, transfer))
		out.transfers = append(out.transfers, transfer.ID.String())
	}

	for idx := range 3 {
		invoice := &edicarrierinvoice.CarrierInvoice{
			ID: pulid.MustNew("edici_"), OrganizationID: org, BusinessUnitID: bu,
			EDIPartnerID: partners[0].ID, InboundMessageID: messages[0].ID,
			InvoiceNumber:        fmt.Sprintf("%s-INV-%d", label, idx+1),
			InvoiceNumberKey:     fmt.Sprintf("%s-INV-%d", label, idx+1),
			ReconciliationStatus: edicarrierinvoice.CarrierInvoiceReconciliationStatusUnmatched,
			LineCharges:          []edipayload.FreightInvoiceCharge{},
			ReferenceNumbers:     map[string]string{},
		}
		must(edicarrierinvoicedb.CarrierInvoices.Create(ctx, admin, invoice))
		out.invoices = append(out.invoices, invoice.ID.String())
	}

	return out
}

// Every EDI read takes, through the screen's resolvers under row-level
// security, the tenant's own rows and none of another's: each list pages
// through them and narrows by a typed filter, each get reads one of the
// tenant's records and refuses another's, and at a role ceiling of Public
// nothing but the ids is shown.
func TestConvertedReads_EDIReadsUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	logger := zap.NewNop()
	partners := edipartnerrepository.New(edipartnerrepository.Params{DB: conn, Logger: logger})
	files := ediinboundfilerepository.New(ediinboundfilerepository.Params{DB: conn, Logger: logger})
	messages := edimessagerepository.New(edimessagerepository.Params{DB: conn, Logger: logger})
	transfers := editransferrepository.New(editransferrepository.Params{DB: conn, Logger: logger})
	invoices := edicarrierinvoicerepository.New(
		edicarrierinvoicerepository.Params{DB: conn, Logger: logger})
	edis := ediservicetest.New(ediservicetest.Deps{
		Logger: logger, PartnerRepo: partners, InboundFileRepo: files, MessageRepo: messages,
		TransferRepo: transfers,
		MappingProfileRepo: edimappingprofilerepository.New(
			edimappingprofilerepository.Params{DB: conn, Logger: logger}),
	})
	resolvers := agentread.ResolverParams{
		EDISetupService:    edis.Setup,
		EDITransferService: edis.Transfers,
		EDIOutboundService: edis.Outbound,
		EDIMessageService:  edis.Messages,
		EDIOverviewService: edis.Overview,
		EDIInboundService: ediinboundservice.New(ediinboundservice.Params{
			Logger:          logger,
			InboundFileRepo: files,
			InboundApply:    edis.InboundApply,
			Outbound:        edis.Outbound,
			Alerts:          edis.Alerts,
		}),
		CarrierSettlementCostService: carriersettlementcostservice.New(
			carriersettlementcostservice.Params{Logger: logger, EDIInvoiceRepo: invoices},
		),
	}
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(
		t,
		agentreadtest.Options{Engine: engine, Resolvers: resolvers},
	)
	public := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: &publicRole{}, Resolvers: resolvers,
	})

	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")
	ours := writeEDIRows(t, admin, mine, "A")
	other := writeEDIRows(t, admin, theirs, "B")
	actor := mine.Actor

	cases := []struct {
		tool        string
		ours, other []string
		narrow      map[string]any
		narrowed    []string
		hidden      string
	}{
		{
			tool: "edi_list_partners", ours: ours.partners, other: other.partners,
			narrow: map[string]any{"query": "A-P2"}, narrowed: ours.partners[1:2],
			hidden: "code",
		},
		{
			tool: "edi_list_inbound_files", ours: ours.files, other: other.files,
			narrow:   map[string]any{"status": []any{"Quarantined"}},
			narrowed: ours.files[:1], hidden: "fileName",
		},
		{
			tool: "edi_list_messages", ours: ours.messages, other: other.messages,
			narrow:   map[string]any{"deliveryStatus": []any{"DeadLettered"}},
			narrowed: ours.messages[:1], hidden: "transactionSet",
		},
		{
			tool: "edi_list_transfers", ours: ours.transfers, other: other.transfers,
			narrow:   map[string]any{"status": []any{"PendingApproval"}},
			narrowed: ours.transfers, hidden: "status",
		},
		{
			tool: "edi_list_carrier_invoices", ours: ours.invoices, other: other.invoices,
			narrow:   map[string]any{"status": "Matched"},
			narrowed: nil, hidden: "invoiceNumber",
		},
	}
	for _, c := range cases {
		t.Run(c.tool, func(t *testing.T) {
			converted, err := agentquerytoolservice.ConvertedListTool(c.tool, reader, engine)
			require.NoError(t, err)

			ids, pages := walk(t, ours.ctx, converted, actor, map[string]any{})
			assert.Greater(t, pages, 1, "the walk must cross a page")
			assert.ElementsMatch(t, c.ours, ids)
			for _, id := range c.other {
				assert.NotContains(t, ids, id)
			}

			narrowed, _ := walk(t, ours.ctx, converted, actor, c.narrow)
			assert.ElementsMatch(t, c.narrowed, narrowed)

			hidden, err := agentquerytoolservice.ConvertedListTool(c.tool, public, engine)
			require.NoError(t, err)
			page := callTool(t, ours.ctx, hidden, actor, map[string]any{})
			assert.NotEmpty(t, page["withheldByAccess"])
			for _, item := range page["items"].([]any) {
				row := item.(map[string]any)
				assert.NotEmpty(t, row["id"])
				assert.NotContains(t, row, c.hidden)
			}
		})
	}

	gets := []struct {
		tool, param  string
		mine, theirs string
		withheld     string
	}{
		{"edi_get_inbound_file", "inboundFileId", ours.files[0], other.files[0], "file.fileName"},
		{"edi_get_message", "messageId", ours.messages[0], other.messages[0], "message.status"},
		{
			"edi_get_transfer",
			"transferId",
			ours.transfers[0],
			other.transfers[0],
			"transfer.status",
		},
	}
	for _, g := range gets {
		t.Run(g.tool, func(t *testing.T) {
			converted, err := agentquerytoolservice.ConvertedGetTool(g.tool, reader, engine)
			require.NoError(t, err)
			record := callTool(t, ours.ctx, converted, actor, map[string]any{g.param: g.mine})
			withheld, _ := record["withheldByAccess"].([]any)
			for _, path := range withheld {
				assert.Equal(t, "transfer.charges", path,
					"at Internal only a tender's rate is withheld")
			}
			assert.NotContains(t, fmt.Sprint(record), "SECRET")

			_, err = converted.Query(ours.ctx, &agenttoolports.QueryToolParams{
				OrganizationID: actor.OrganizationID,
				BusinessUnitID: actor.BusinessUnitID,
				Actor:          actor,
				Timezone:       "America/Chicago",
				Params:         map[string]any{g.param: g.theirs},
				RunID:          pulid.MustNew("ar_"),
			})
			require.Error(t, err, "another tenant's record is not found")

			hidden, err := agentquerytoolservice.ConvertedGetTool(g.tool, public, engine)
			require.NoError(t, err)
			record = callTool(t, ours.ctx, hidden, actor, map[string]any{g.param: g.mine})
			assert.Contains(t, record["withheldByAccess"], g.withheld)
		})
	}
}
