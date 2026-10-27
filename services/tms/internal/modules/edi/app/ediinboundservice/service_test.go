package ediinboundservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks/ediportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/lading"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type inboundFixture struct {
	service            *Service
	file               *edi.EDIInboundFile
	partner            *edi.EDIPartner
	inboundFileRepo    *ediportsmock.MockEDIInboundFileRepository
	partnerRepo        *ediportsmock.MockEDIPartnerRepository
	messageRepo        *ediportsmock.MockEDIMessageRepository
	documentTypeRepo   *ediportsmock.MockEDIDocumentTypeRepository
	projects           *ediportsmock.MockLadingProjectRepository
	flowRepo           *ediportsmock.MockFlowRepository
	orgs               *tenantportsmock.MockOrganizationRepository
	transferRepo       *ediportsmock.MockEDILoadTenderTransferRepository
	recipientRepo      *ediportsmock.MockEDITenderRecipientRepository
	mappingProfileRepo *ediportsmock.MockEDIMappingProfileRepository
	controlNumberRepo  *ediportsmock.MockEDIControlNumberRepository
}

func newInboundFixture(t *testing.T, rawContent string) *inboundFixture {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	fixture := &inboundFixture{
		file: &edi.EDIInboundFile{
			ID:                     pulid.MustNew("ediinf_"),
			OrganizationID:         orgID,
			BusinessUnitID:         buID,
			CommunicationProfileID: pulid.MustNew("edicp_"),
			EDIPartnerID:           partnerID,
			Method:                 edi.ConnectionMethodSFTP,
			FileName:               "inbound.x12",
			RawContent:             rawContent,
			Status:                 edi.InboundFileStatusReceived,
		},
		partner: &edi.EDIPartner{
			ID:                partnerID,
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			Kind:              edi.PartnerKindExternal,
			Code:              "PARTNER",
			EnabledForInbound: true,
		},
		inboundFileRepo:    ediportsmock.NewMockEDIInboundFileRepository(t),
		partnerRepo:        ediportsmock.NewMockEDIPartnerRepository(t),
		messageRepo:        ediportsmock.NewMockEDIMessageRepository(t),
		documentTypeRepo:   ediportsmock.NewMockEDIDocumentTypeRepository(t),
		projects:           ediportsmock.NewMockLadingProjectRepository(t),
		flowRepo:           ediportsmock.NewMockFlowRepository(t),
		orgs:               tenantportsmock.NewMockOrganizationRepository(t),
		transferRepo:       ediportsmock.NewMockEDILoadTenderTransferRepository(t),
		recipientRepo:      ediportsmock.NewMockEDITenderRecipientRepository(t),
		mappingProfileRepo: ediportsmock.NewMockEDIMappingProfileRepository(t),
		controlNumberRepo:  ediportsmock.NewMockEDIControlNumberRepository(t),
	}
	runtime := edilading.NewRuntime(edilading.RuntimeParams{
		Logger:   zap.NewNop(),
		Projects: fixture.projects,
		Mappings: fixture.mappingProfileRepo,
		Orgs:     fixture.orgs,
	})
	flows := edilading.NewFlows(edilading.FlowsParams{
		Logger:  zap.NewNop(),
		Runtime: runtime,
		Flows:   fixture.flowRepo,
	})
	ediSvc := ediservice.New(ediservice.Params{
		Logger:              zap.NewNop(),
		PartnerRepo:         fixture.partnerRepo,
		MappingProfileRepo:  fixture.mappingProfileRepo,
		TenderRecipientRepo: fixture.recipientRepo,
		ControlNumberRepo:   fixture.controlNumberRepo,
		MessageRepo:         fixture.messageRepo,
		DocumentTypeRepo:    fixture.documentTypeRepo,
		FlowRepo:            fixture.flowRepo,
		Lading:              runtime,
		Flows:               flows,
	})
	fixture.service = &Service{
		l:                   zap.NewNop(),
		metrics:             metrics.NewEDI(nil, zap.NewNop(), false),
		inboundFileRepo:     fixture.inboundFileRepo,
		partnerRepo:         fixture.partnerRepo,
		messageRepo:         fixture.messageRepo,
		documentTypeRepo:    fixture.documentTypeRepo,
		transferRepo:        fixture.transferRepo,
		tenderRecipientRepo: fixture.recipientRepo,
		ediService:          ediSvc,
		lading:              runtime,
		flows:               flows,
	}
	return fixture
}

// expectActiveVersion gives the partner Trenova's carrier starter as its active Lading
// version. It acknowledges nothing, so a test sees no outbound acknowledgment, unless the test
// names the acknowledgment.
func (f *inboundFixture) expectActiveVersion(t *testing.T, acknowledge ...string) *lading.Starter {
	t.Helper()

	ack := lading.AcknowledgeNone
	if len(acknowledge) > 0 {
		ack = acknowledge[0]
	}
	client := lading.Default()
	starter, err := client.Starter(t.Context(), &lading.StarterRequest{
		Name:        "partner",
		Label:       "Partner",
		Role:        lading.RoleCarrier,
		Them:        &[2]string{"ZZ", "PARTNER"},
		Us:          &[2]string{"ZZ", "TRENOVA"},
		Usage:       "production",
		Acknowledge: ack,
	})
	require.NoError(t, err)
	org := lading.Org{Name: "Trenova", SCAC: "TRNV", Zone: "America/Chicago"}
	encoded, err := lading.Encode(
		&lading.Project{Files: starter.Files, Samples: starter.Samples},
		&lading.Context{Org: org},
	)
	require.NoError(t, err)
	check, err := client.Check(t.Context(), encoded)
	require.NoError(t, err)
	require.Zero(t, check.Errors)

	f.orgs.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{Name: org.Name, ScacCode: org.SCAC, Timezone: org.Zone}, nil).
		Maybe()
	f.projects.EXPECT().GetRunnableVersions(mock.Anything, mock.Anything).
		Return(&ediports.RunnableLadingVersions{
			Project: &edi.LadingProject{
				ID:           pulid.MustNew("ldp_"),
				EDIPartnerID: f.partner.ID,
				Settings:     map[string]any{},
			},
			Active: &edi.LadingVersion{
				ID:       pulid.MustNew("ldv_"),
				Number:   1,
				Status:   edi.LadingVersionStatusActive,
				Files:    starter.Files,
				Manifest: *check,
				Version:  1,
			},
		}, nil).
		Maybe()
	return starter
}

// expectFlows lets the partner's flow open, step and journal instances.
func (f *inboundFixture) expectFlows() {
	f.flowRepo.EXPECT().OpenInstance(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, instance *edi.FlowInstance) (*edi.FlowInstance, error) {
			instance.ID = pulid.MustNew("edfi_")
			return instance, nil
		}).
		Maybe()
	f.flowRepo.EXPECT().SaveInstance(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, instance *edi.FlowInstance) (*edi.FlowInstance, error) {
			return instance, nil
		}).
		Maybe()
	f.flowRepo.EXPECT().AppendEvent(mock.Anything, mock.Anything).Return(nil).Maybe()
	f.messageRepo.EXPECT().LinkMessageFlow(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *ediports.LinkEDIMessageFlowRequest) (*edi.EDIMessage, error) {
			return &edi.EDIMessage{ID: req.ID, FlowInstanceID: req.FlowInstanceID}, nil
		}).
		Maybe()
}

func (f *inboundFixture) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: f.file.OrganizationID,
		BuID:  f.file.BusinessUnitID,
	}
}

func (f *inboundFixture) expectFileLoadAndUpdates(t *testing.T) *edi.EDIInboundFile {
	t.Helper()
	var lastUpdated edi.EDIInboundFile
	f.inboundFileRepo.EXPECT().
		GetInboundFileByID(mock.Anything, ediports.GetEDIInboundFileByIDRequest{
			ID:         f.file.ID,
			TenantInfo: f.tenantInfo(),
		}).
		Return(f.file, nil).
		Once()
	f.inboundFileRepo.EXPECT().
		UpdateInboundFile(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, updated *edi.EDIInboundFile) (*edi.EDIInboundFile, error) {
			lastUpdated = *updated
			return updated, nil
		})
	return &lastUpdated
}

func (f *inboundFixture) expectPartnerLoad() {
	f.partnerRepo.EXPECT().
		GetByID(mock.Anything, ediports.GetEDIPartnerByIDRequest{
			ID:         f.partner.ID,
			TenantInfo: f.tenantInfo(),
		}).
		Return(f.partner, nil).
		Once()
}

func (f *inboundFixture) expectInboundDocumentType(set edi.TransactionSet) {
	f.documentTypeRepo.EXPECT().
		ListDocumentTypes(mock.Anything, ediports.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandardX12,
			TransactionSet: set,
			Direction:      edi.DocumentDirectionInbound,
		}).
		Return([]*edi.EDIDocumentType{{
			ID:             pulid.MustNew("edidt_"),
			TransactionSet: set,
			Direction:      edi.DocumentDirectionInbound,
			DefaultVersion: "004010",
		}}, nil).
		Once()
}

func (f *inboundFixture) expectMessageCreate() {
	f.messageRepo.EXPECT().
		CreateMessageWithDiagnostics(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req ediports.CreateEDIMessageWithDiagnosticsRequest) (*edi.EDIMessage, error) {
			message := *req.Message
			message.ID = pulid.MustNew("edimsg_")
			return &message, nil
		}).
		Once()
}

func TestProcessInboundFile_Reconciles997Acknowledgment(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000101*0*P*>~" +
		"GS*FA*PARTNER*TRENOVA*20260107*1200*101*X*004010~" +
		"ST*997*0001~" +
		"AK1*SM*42~" +
		"AK2*204*0042~" +
		"AK5*A~" +
		"AK9*A*1*1*1~" +
		"SE*6*0001~" +
		"GE*1*101~" +
		"IEA*1*000000101~"
	fixture := newInboundFixture(t, raw)
	lastUpdate := fixture.expectFileLoadAndUpdates(t)
	fixture.expectPartnerLoad()
	fixture.expectActiveVersion(t)
	fixture.expectInboundDocumentType(edi.TransactionSet997)
	fixture.expectMessageCreate()

	originalMessage := &edi.EDIMessage{
		ID:             pulid.MustNew("edimsg_"),
		OrganizationID: fixture.file.OrganizationID,
		BusinessUnitID: fixture.file.BusinessUnitID,
		EDIPartnerID:   fixture.partner.ID,
		Direction:      edi.DocumentDirectionOutbound,
		AckStatus:      edi.MessageAcknowledgmentStatusPending,
	}
	fixture.messageRepo.EXPECT().
		GetOutboundMessageForAck(mock.Anything, mock.MatchedBy(
			func(req ediports.GetEDIOutboundMessageForAckRequest) bool {
				return req.PartnerID == fixture.partner.ID &&
					req.GroupControlNumber == "42" &&
					req.TransactionControlNumber == "0042" &&
					req.TransactionSet == edi.TransactionSet204
			},
		)).
		Return(originalMessage, nil).
		Once()
	fixture.messageRepo.EXPECT().
		UpdateMessageAcknowledgment(mock.Anything, mock.MatchedBy(
			func(req *ediports.UpdateEDIMessageAcknowledgmentRequest) bool {
				return req.ID == originalMessage.ID &&
					req.AckStatus == edi.MessageAcknowledgmentStatusAccepted &&
					req.AckMessageID.IsNotNil()
			},
		)).
		Return(originalMessage, nil).
		Once()

	file, err := fixture.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     fixture.file.ID,
		TenantInfo: fixture.tenantInfo(),
	})

	require.NoError(t, err)
	require.Equal(t, edi.InboundFileStatusProcessed, file.Status)
	require.Equal(t, "000000101", lastUpdate.InterchangeControlNumber)
}

func TestProcessInboundFile_Creates204TransferRequiringMapping(t *testing.T) {
	t.Parallel()

	fixture := newInboundFixture(t, "")
	starter := fixture.expectActiveVersion(t)
	fixture.file.RawContent = starterSample(t, starter, "samples/tender-original.x12")
	fixture.expectFileLoadAndUpdates(t)
	fixture.expectPartnerLoad()
	fixture.expectInboundDocumentType(edi.TransactionSet204)
	fixture.expectMessageCreate()
	fixture.expectFlows()

	fixture.mappingProfileRepo.EXPECT().
		GetMappingItems(mock.Anything, mock.Anything).
		Return([]*edi.EDIMappingProfileItem{}, nil)
	fixture.transferRepo.EXPECT().
		CreateTransfer(mock.Anything, mock.MatchedBy(func(transfer *edi.EDITransfer) bool {
			return transfer.Status == edi.TransferStatusMappingRequired &&
				transfer.InboundMessageID.IsNotNil() &&
				transfer.SourceShipmentID.IsNil() &&
				transfer.TargetPartnerID == fixture.partner.ID &&
				transfer.TenderPayload.BOL == "BOL-55"
		})).
		RunAndReturn(func(_ context.Context, transfer *edi.EDITransfer) (*edi.EDITransfer, error) {
			transfer.ID = pulid.MustNew("edilt_")
			return transfer, nil
		}).
		Once()

	file, err := fixture.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     fixture.file.ID,
		TenantInfo: fixture.tenantInfo(),
	})

	require.NoError(t, err)
	require.Equal(t, edi.InboundFileStatusPartiallyProcessed, file.Status)
	require.Contains(t, file.FailureReason, "requires mapping")
}

func TestProcessInboundFile_QuarantinesUnparseableContent(t *testing.T) {
	t.Parallel()

	fixture := newInboundFixture(t, "this is not x12")
	fixture.expectFileLoadAndUpdates(t)

	file, err := fixture.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     fixture.file.ID,
		TenantInfo: fixture.tenantInfo(),
	})

	require.NoError(t, err)
	require.Equal(t, edi.InboundFileStatusQuarantined, file.Status)
	require.NotEmpty(t, file.FailureReason)
}

func starterSample(t *testing.T, starter *lading.Starter, path string) string {
	t.Helper()
	for _, sample := range starter.Samples {
		if sample.Path == path {
			return sample.Content
		}
	}
	t.Fatalf("the starter has no sample %s", path)
	return ""
}
