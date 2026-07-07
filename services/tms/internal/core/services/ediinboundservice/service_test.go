package ediinboundservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type inboundFixture struct {
	service             *Service
	file                *edi.EDIInboundFile
	partner             *edi.EDIPartner
	inboundFileRepo     *mocks.MockEDIInboundFileRepository
	partnerRepo         *mocks.MockEDIPartnerRepository
	messageRepo         *mocks.MockEDIMessageRepository
	documentTypeRepo    *mocks.MockEDIDocumentTypeRepository
	documentProfileRepo *mocks.MockEDIPartnerDocumentProfileRepository
	transferRepo        *mocks.MockEDILoadTenderTransferRepository
	recipientRepo       *mocks.MockEDITenderRecipientRepository
	mappingProfileRepo  *mocks.MockEDIMappingProfileRepository
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
		inboundFileRepo:     mocks.NewMockEDIInboundFileRepository(t),
		partnerRepo:         mocks.NewMockEDIPartnerRepository(t),
		messageRepo:         mocks.NewMockEDIMessageRepository(t),
		documentTypeRepo:    mocks.NewMockEDIDocumentTypeRepository(t),
		documentProfileRepo: mocks.NewMockEDIPartnerDocumentProfileRepository(t),
		transferRepo:        mocks.NewMockEDILoadTenderTransferRepository(t),
		recipientRepo:       mocks.NewMockEDITenderRecipientRepository(t),
		mappingProfileRepo:  mocks.NewMockEDIMappingProfileRepository(t),
	}
	ediSvc := ediservice.New(ediservice.Params{
		Logger:              zap.NewNop(),
		MappingProfileRepo:  fixture.mappingProfileRepo,
		TenderRecipientRepo: fixture.recipientRepo,
	})
	fixture.service = &Service{
		l:                   zap.NewNop(),
		inboundFileRepo:     fixture.inboundFileRepo,
		partnerRepo:         fixture.partnerRepo,
		messageRepo:         fixture.messageRepo,
		documentTypeRepo:    fixture.documentTypeRepo,
		documentProfileRepo: fixture.documentProfileRepo,
		transferRepo:        fixture.transferRepo,
		tenderRecipientRepo: fixture.recipientRepo,
		ediService:          ediSvc,
	}
	return fixture
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
		GetInboundFileByID(mock.Anything, repositories.GetEDIInboundFileByIDRequest{
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
		GetByID(mock.Anything, repositories.GetEDIPartnerByIDRequest{
			ID:         f.partner.ID,
			TenantInfo: f.tenantInfo(),
		}).
		Return(f.partner, nil).
		Once()
}

func (f *inboundFixture) expectInboundDocumentType(set edi.TransactionSet) {
	f.documentTypeRepo.EXPECT().
		ListDocumentTypes(mock.Anything, repositories.ListEDIDocumentTypesRequest{
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
		RunAndReturn(func(_ context.Context, req repositories.CreateEDIMessageWithDiagnosticsRequest) (*edi.EDIMessage, error) {
			message := *req.Message
			message.ID = pulid.MustNew("edimsg_")
			return &message, nil
		}).
		Once()
}

func (f *inboundFixture) expectNoInboundAckProfile(set edi.TransactionSet) {
	f.documentProfileRepo.EXPECT().
		GetActivePartnerDocumentProfile(
			mock.Anything,
			repositories.GetActiveEDIPartnerDocumentProfileRequest{
				PartnerID:      f.partner.ID,
				TenantInfo:     f.tenantInfo(),
				TransactionSet: set,
				Direction:      edi.DocumentDirectionInbound,
			},
		).
		Return(nil, errortypes.NewNotFoundError("EDIPartnerDocumentProfile")).
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
			func(req repositories.GetEDIOutboundMessageForAckRequest) bool {
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
			func(req *repositories.UpdateEDIMessageAcknowledgmentRequest) bool {
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

	raw := renderBase204(t, sampleTenderPayload())
	fixture := newInboundFixture(t, raw)
	fixture.expectFileLoadAndUpdates(t)
	fixture.expectPartnerLoad()
	fixture.expectInboundDocumentType(edi.TransactionSet204)
	fixture.expectMessageCreate()
	fixture.expectNoInboundAckProfile(edi.TransactionSet204)

	fixture.mappingProfileRepo.EXPECT().
		GetMappingItems(mock.Anything, mock.Anything).
		Return([]*edi.EDIMappingProfileItem{}, nil).
		Once()
	fixture.transferRepo.EXPECT().
		CreateTransfer(mock.Anything, mock.MatchedBy(func(transfer *edi.EDITransfer) bool {
			return transfer.Status == edi.TransferStatusMappingRequired &&
				transfer.InboundMessageID.IsNotNil() &&
				transfer.SourceShipmentID.IsNil() &&
				transfer.TargetPartnerID == fixture.partner.ID &&
				transfer.TenderPayload.BOL == "BOL-778899"
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
