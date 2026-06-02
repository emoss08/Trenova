package ediservice

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestShipmentStatusCapabilityEnabledDefaultsTrueForDirectExternalPartner(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	enabled, err := svc.shipmentStatusCapabilityEnabled(
		t.Context(),
		pagination.TenantInfo{},
		&edi.EDIPartner{Kind: edi.PartnerKindExternal},
	)

	require.NoError(t, err)
	require.True(t, enabled)
}

func TestShipmentStatusCapabilityEnabledDefaultsTrueForMissingConnection(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	connectionRepo := mocks.NewMockEDIConnectionRepository(t)
	connectionID := pulid.MustNew("edic_")
	connectionRepo.EXPECT().
		GetConnectionByID(mock.Anything, repositories.GetEDIConnectionByIDRequest{
			ID:         connectionID,
			TenantInfo: tenantInfo,
		}).
		Return(nil, sql.ErrNoRows).
		Once()

	svc := &Service{connectionRepo: connectionRepo}
	enabled, err := svc.shipmentStatusCapabilityEnabled(
		t.Context(),
		tenantInfo,
		&edi.EDIPartner{Kind: edi.PartnerKindExternal, EDIConnectionID: connectionID},
	)

	require.NoError(t, err)
	require.True(t, enabled)
}

func TestResolveServiceFailure214CandidateSkippedReasons(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	customerID := pulid.MustNew("cus_")
	partnerID := pulid.MustNew("edip_")
	trigger := services.ServiceFailureEDITriggerReviewed

	tests := []struct {
		name       string
		source     *shipment.Shipment
		partners   []*edi.EDIPartner
		profiles   []*edi.EDIPartnerDocumentProfile
		wantReason string
		wantAction services.ServiceFailureEDIAction
	}{
		{
			name:       "no customer",
			source:     &shipment.Shipment{},
			wantReason: "shipment customer is not linked to an EDI partner",
			wantAction: services.ServiceFailureEDIActionSkipped,
		},
		{
			name:       "no partner",
			source:     &shipment.Shipment{CustomerID: customerID},
			wantReason: "no outbound EDI partner for shipment customer",
			wantAction: services.ServiceFailureEDIActionSkipped,
		},
		{
			name:   "inactive or outbound disabled partner",
			source: &shipment.Shipment{CustomerID: customerID},
			partners: []*edi.EDIPartner{{
				ID:                 partnerID,
				Status:             domaintypes.StatusInactive,
				EnabledForOutbound: true,
			}},
			wantReason: "EDI partner is inactive or outbound disabled",
			wantAction: services.ServiceFailureEDIActionSkipped,
		},
		{
			name:   "inactive profile",
			source: &shipment.Shipment{CustomerID: customerID},
			partners: []*edi.EDIPartner{{
				ID:                 partnerID,
				Status:             domaintypes.StatusActive,
				EnabledForOutbound: true,
			}},
			profiles: []*edi.EDIPartnerDocumentProfile{{
				ID:     pulid.MustNew("edidp_"),
				Status: edi.DocumentStatusInactive,
			}},
			wantReason: "service failure 214 partner document profile inactive",
			wantAction: services.ServiceFailureEDIActionSkipped,
		},
		{
			name:   "trigger disabled",
			source: &shipment.Shipment{CustomerID: customerID},
			partners: []*edi.EDIPartner{{
				ID:                 partnerID,
				Status:             domaintypes.StatusActive,
				EnabledForOutbound: true,
			}},
			profiles: []*edi.EDIPartnerDocumentProfile{{
				ID:     pulid.MustNew("edidp_"),
				Status: edi.DocumentStatusActive,
				PartnerSettings: map[string]any{
					"serviceFailure214": map[string]any{"enabled": true},
				},
			}},
			wantReason: "service failure 214 trigger disabled",
			wantAction: services.ServiceFailureEDIActionSkipped,
		},
		{
			name:   "ambiguous mandatory profiles block",
			source: &shipment.Shipment{CustomerID: customerID},
			partners: []*edi.EDIPartner{{
				ID:                 partnerID,
				Status:             domaintypes.StatusActive,
				EnabledForOutbound: true,
			}},
			profiles: []*edi.EDIPartnerDocumentProfile{
				activeServiceFailure214Profile(true),
				activeServiceFailure214Profile(true),
			},
			wantReason: "ambiguous service failure 214 partner document profile",
			wantAction: services.ServiceFailureEDIActionBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partnerRepo := mocks.NewMockEDIPartnerRepository(t)
			documentProfileRepo := mocks.NewMockEDIPartnerDocumentProfileRepository(t)
			svc := &Service{
				partnerRepo:         partnerRepo,
				documentProfileRepo: documentProfileRepo,
			}
			if !tt.source.CustomerID.IsNil() {
				partnerRepo.EXPECT().
					List(mock.Anything, mock.AnythingOfType("*repositories.ListEDIPartnersRequest")).
					Return(&pagination.ListResult[*edi.EDIPartner]{
						Items: tt.partners,
						Total: len(tt.partners),
					}, nil).
					Once()
			}
			if len(tt.partners) > 0 && tt.partners[0].Status == domaintypes.StatusActive {
				documentProfileRepo.EXPECT().
					ListPartnerDocumentProfiles(
						mock.Anything,
						mock.AnythingOfType("*repositories.ListEDIPartnerDocumentProfilesRequest"),
					).
					Return(&pagination.ListResult[*edi.EDIPartnerDocumentProfile]{
						Items: tt.profiles,
						Total: len(tt.profiles),
					}, nil).
					Once()
			}

			candidate, result, err := svc.resolveServiceFailure214Candidate(
				t.Context(),
				&services.ServiceFailure214LifecycleRequest{
					TenantInfo: tenantInfo,
					Trigger:    trigger,
				},
				tt.source,
			)

			require.NoError(t, err)
			require.Nil(t, candidate)
			require.NotNil(t, result)
			require.Equal(t, tt.wantReason, result.SkippedReason)
			require.Equal(t, tt.wantAction, result.Action)
		})
	}
}

func TestPreviewServiceFailure214ForLifecycleReadyForGeneration(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	shipmentID := pulid.MustNew("sp_")
	customerID := pulid.MustNew("cus_")
	failure := &servicefailure.ServiceFailure{
		ID:          pulid.MustNew("sf_"),
		ShipmentID:  shipmentID,
		Number:      "SF-1005",
		Type:        servicefailure.TypeLateDelivery,
		Status:      servicefailure.StatusReviewed,
		DetectedAt:  1,
		LateMinutes: 1,
		ReasonCode:  &servicefailure.ReasonCode{DefaultReasonCode: "NS"},
	}
	messageRepo := mocks.NewMockEDIMessageRepository(t)
	shipmentSvc := mocks.NewMockShipmentService(t)
	partnerRepo := mocks.NewMockEDIPartnerRepository(t)
	documentProfileRepo := mocks.NewMockEDIPartnerDocumentProfileRepository(t)
	svc := &Service{
		l:                   zap.NewNop(),
		messageRepo:         messageRepo,
		shipmentSvc:         shipmentSvc,
		partnerRepo:         partnerRepo,
		documentProfileRepo: documentProfileRepo,
	}

	messageRepo.EXPECT().
		GetServiceFailure214LifecycleMessage(mock.Anything, mock.Anything).
		Return(nil, sql.ErrNoRows).
		Once()
	shipmentSvc.EXPECT().
		Get(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(&shipment.Shipment{ID: shipmentID, CustomerID: customerID}, nil).
		Once()
	partnerRepo.EXPECT().
		List(mock.Anything, mock.AnythingOfType("*repositories.ListEDIPartnersRequest")).
		Return(&pagination.ListResult[*edi.EDIPartner]{
			Items: []*edi.EDIPartner{{
				ID:                 pulid.MustNew("edip_"),
				Status:             domaintypes.StatusActive,
				EnabledForOutbound: true,
			}},
			Total: 1,
		}, nil).
		Once()
	documentProfileRepo.EXPECT().
		ListPartnerDocumentProfiles(
			mock.Anything,
			mock.AnythingOfType("*repositories.ListEDIPartnerDocumentProfilesRequest"),
		).
		Return(&pagination.ListResult[*edi.EDIPartnerDocumentProfile]{
			Items: []*edi.EDIPartnerDocumentProfile{activeServiceFailure214Profile(false)},
			Total: 1,
		}, nil).
		Once()

	result, err := svc.PreviewServiceFailure214ForLifecycle(t.Context(), &services.ServiceFailure214LifecycleRequest{
		TenantInfo:       tenantInfo,
		ServiceFailureID: failure.ID,
		ShipmentID:       shipmentID,
		Trigger:          services.ServiceFailureEDITriggerReviewed,
		ServiceFailure:   failure,
	})

	require.NoError(t, err)
	require.Equal(t, services.ServiceFailureEDIActionSkipped, result.Action)
	require.Equal(t, "ready_for_generation", result.SkippedReason)
}

func activeServiceFailure214Profile(mandatory bool) *edi.EDIPartnerDocumentProfile {
	return &edi.EDIPartnerDocumentProfile{
		ID:     pulid.MustNew("edidp_"),
		Status: edi.DocumentStatusActive,
		PartnerSettings: map[string]any{
			"serviceFailure214": map[string]any{
				"enabled":             true,
				"sendOnReviewed":      !mandatory,
				"mandatoryOnReviewed": mandatory,
			},
		},
	}
}
