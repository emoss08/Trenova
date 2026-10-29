package editenderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/app/edialert"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/edipayloadbuild"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendershipment"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/editenderportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentserviceportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestService_SubmitLoadTenderRequiresReciprocalPartnerInSameBusinessUnit(t *testing.T) {
	ctx := t.Context()
	buID := pulid.MustNew("bu_")
	sourceOrgID := pulid.MustNew("org_")
	targetOrgID := pulid.MustNew("org_")
	userID := pulid.MustNew("usr_")
	sourcePartnerID := pulid.MustNew("edip_")
	targetPartnerID := pulid.MustNew("edip_")
	sourceShipmentID := pulid.MustNew("sp_")

	sourcePartner := &edi.EDIPartner{
		ID:                     sourcePartnerID,
		BusinessUnitID:         buID,
		OrganizationID:         sourceOrgID,
		Kind:                   edi.PartnerKindInternal,
		InternalOrganizationID: targetOrgID,
		EnabledForOutbound:     true,
	}
	targetPartner := &edi.EDIPartner{
		ID:                 targetPartnerID,
		BusinessUnitID:     buID,
		OrganizationID:     targetOrgID,
		Kind:               edi.PartnerKindInternal,
		EnabledForInbound:  true,
		EnabledForOutbound: true,
	}

	partnerRepo := edipartnerportsmock.NewMockEDIPartnerRepository(t)
	partnerRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req edipartnerports.GetEDIPartnerByIDRequest) bool {
			return req.ID == sourcePartnerID &&
				req.TenantInfo.OrgID == sourceOrgID &&
				req.TenantInfo.BuID == buID
		})).
		Return(sourcePartner, nil)
	partnerRepo.EXPECT().
		GetReciprocalInternalPartner(mock.Anything, mock.MatchedBy(func(req edipartnerports.GetReciprocalInternalPartnerRequest) bool {
			return req.SourceOrganizationID == sourceOrgID &&
				req.TargetOrganizationID == targetOrgID &&
				req.BusinessUnitID == buID
		})).
		Return(targetPartner, nil)
	mappingProfileRepo := edipartnerportsmock.NewMockEDIMappingProfileRepository(t)
	mappingProfileRepo.EXPECT().
		GetMappingItems(mock.Anything, mock.MatchedBy(func(req edipartnerports.GetMappingItemsRequest) bool {
			return req.PartnerID == targetPartnerID &&
				req.TenantInfo.OrgID == targetOrgID &&
				req.TenantInfo.BuID == buID
		})).
		Return([]*edi.EDIMappingProfileItem{}, nil)

	connectionRepo := edipartnerportsmock.NewMockEDIConnectionRepository(t)
	connectionRepo.EXPECT().
		GetActiveConnectionForPartner(mock.Anything, mock.MatchedBy(func(req edipartnerports.GetActiveEDIConnectionForPartnerRequest) bool {
			return req.PartnerID == sourcePartnerID &&
				req.TenantInfo.OrgID == sourceOrgID &&
				req.TenantInfo.BuID == buID &&
				req.Method == edi.ConnectionMethodInternal
		})).
		Return(&edi.EDIConnection{
			ID:                   pulid.MustNew("edic_"),
			BusinessUnitID:       buID,
			SourceOrganizationID: sourceOrgID,
			TargetOrganizationID: targetOrgID,
			SourcePartnerID:      sourcePartnerID,
			TargetPartnerID:      targetPartnerID,
			Method:               edi.ConnectionMethodInternal,
			Status:               edi.ConnectionStatusActive,
			Capabilities: edi.ConnectionCapabilities{
				LoadTenderOutbound: true,
				LoadTenderInbound:  true,
			},
		}, nil)

	profileRepo := edipartnerportsmock.NewMockEDICommunicationProfileRepository(t)
	profileRepo.EXPECT().
		GetActiveProfileByPartner(mock.Anything, mock.MatchedBy(func(req edipartnerports.GetActiveEDICommunicationProfileByPartnerRequest) bool {
			return req.PartnerID == sourcePartnerID &&
				req.TenantInfo.OrgID == sourceOrgID &&
				req.TenantInfo.BuID == buID &&
				req.Method == edi.ConnectionMethodInternal
		})).
		Return(&edi.EDICommunicationProfile{ID: pulid.MustNew("edicp_")}, nil)
	profileRepo.EXPECT().
		GetActiveProfileByPartner(mock.Anything, mock.MatchedBy(func(req edipartnerports.GetActiveEDICommunicationProfileByPartnerRequest) bool {
			return req.PartnerID == targetPartnerID &&
				req.TenantInfo.OrgID == targetOrgID &&
				req.TenantInfo.BuID == buID &&
				req.Method == edi.ConnectionMethodInternal
		})).
		Return(&edi.EDICommunicationProfile{ID: pulid.MustNew("edicp_")}, nil)

	var createdTransfer *editransfer.EDITransfer
	transferRepo := editenderportsmock.NewMockEDILoadTenderTransferRepository(t)
	transferRepo.EXPECT().
		CreateTransfer(mock.Anything, mock.AnythingOfType("*editransfer.EDITransfer")).
		RunAndReturn(func(_ context.Context, entity *editransfer.EDITransfer) (*editransfer.EDITransfer, error) {
			createdTransfer = entity
			return entity, nil
		})

	shipmentSvc := shipmentserviceportsmock.NewMockShipmentReader(t)
	shipmentSvc.EXPECT().
		Get(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
			return req.ID == sourceShipmentID &&
				req.TenantInfo.OrgID == sourceOrgID &&
				req.TenantInfo.BuID == buID &&
				req.ExpandShipmentDetails
		})).
		Return(&shipment.Shipment{
			ID:                sourceShipmentID,
			BusinessUnitID:    buID,
			OrganizationID:    sourceOrgID,
			Status:            shipmentvocab.StatusNew,
			ServiceTypeID:     pulid.MustNew("st_"),
			CustomerID:        pulid.MustNew("cus_"),
			FormulaTemplateID: pulid.MustNew("ft_"),
		}, nil)

	service := &Service{
		l:              zap.NewNop(),
		partnerRepo:    partnerRepo,
		connectionRepo: connectionRepo,
		profileRepo:    profileRepo,
		transferRepo:   transferRepo,
		shipmentSvc:    shipmentSvc,
		tenderMapping: editendermapping.New(editendermapping.Params{
			MappingProfileRepo: mappingProfileRepo,
		}),
		outbound: edioutbound.New(edioutbound.Params{
			Logger:       zap.NewNop(),
			PartnerRepo:  partnerRepo,
			ShipmentSvc:  shipmentSvc,
			TransferRepo: transferRepo,
			Alert: edialert.New(edialert.Params{
				Logger: zap.NewNop(),
			}),
			Delivery: edidelivery.New(edidelivery.Params{
				Logger:      zap.NewNop(),
				PartnerRepo: partnerRepo,
				ProfileRepo: profileRepo,
				Alert: edialert.New(edialert.Params{
					Logger: zap.NewNop(),
				}),
				Recipient: editenderrecipient.New(editenderrecipient.Params{
					ProfileRepo: profileRepo,
				}),
				Secret: ediprofilesecret.New(ediprofilesecret.Params{}),
			}),
			Payload: edipayloadbuild.New(edipayloadbuild.Params{
				ShipmentSvc: shipmentSvc,
			}),
			Recipient: editenderrecipient.New(editenderrecipient.Params{
				ProfileRepo: profileRepo,
			}),
		}),
		recipient: editenderrecipient.New(editenderrecipient.Params{
			ProfileRepo: profileRepo,
		}),
		shipments: editendershipment.New(editendershipment.Params{}),
	}

	transfer, err := service.SubmitLoadTender(ctx, &SubmitLoadTenderRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  sourceOrgID,
			BuID:   buID,
			UserID: userID,
		},
		SourceShipmentID: sourceShipmentID,
		EDIPartnerID:     sourcePartnerID,
	}, &permissionports.RequestActor{
		UserID:         userID,
		BusinessUnitID: buID,
		OrganizationID: sourceOrgID,
	})

	require.NoError(t, err)
	require.NotNil(t, transfer)
	require.Equal(t, buID, transfer.TargetBusinessUnitID)
	require.NotNil(t, createdTransfer)
	require.Equal(t, buID, createdTransfer.SourceBusinessUnitID)
	require.Equal(t, buID, createdTransfer.TargetBusinessUnitID)
}
