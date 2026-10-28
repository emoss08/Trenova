package ediservice

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ladingportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
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
	connectionRepo := edipartnerportsmock.NewMockEDIConnectionRepository(t)
	connectionID := pulid.MustNew("edic_")
	connectionRepo.EXPECT().
		GetConnectionByID(mock.Anything, edipartnerports.GetEDIConnectionByIDRequest{
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
	trigger := servicefailureports.ServiceFailureEDITriggerReviewed
	active := func() *edi.EDIPartner {
		return &edi.EDIPartner{
			ID:                 pulid.MustNew("edip_"),
			Status:             domaintypes.StatusActive,
			EnabledForOutbound: true,
		}
	}
	sends := []lading.Flow{{Name: "status", Clauses: []lading.Clause{{
		Kind:   lading.ClauseSend,
		Set:    "214",
		Map:    "statusOut",
		Events: []string{"ServiceFailure"},
	}}}}

	tests := []struct {
		name       string
		source     *shipment.Shipment
		partners   []*edi.EDIPartner
		flows      [][]lading.Flow
		wantReason string
		wantAction servicefailureports.ServiceFailureEDIAction
	}{
		{
			name:       "no customer",
			source:     &shipment.Shipment{},
			wantReason: "shipment customer is not linked to an EDI partner",
			wantAction: ediports.ServiceFailureEDIActionSkipped,
		},
		{
			name:       "no partner",
			source:     &shipment.Shipment{CustomerID: customerID},
			wantReason: "no outbound EDI partner for shipment customer",
			wantAction: ediports.ServiceFailureEDIActionSkipped,
		},
		{
			name:   "inactive or outbound disabled partner",
			source: &shipment.Shipment{CustomerID: customerID},
			partners: []*edi.EDIPartner{{
				ID:                 pulid.MustNew("edip_"),
				Status:             domaintypes.StatusInactive,
				EnabledForOutbound: true,
			}},
			wantReason: "EDI partner is inactive or outbound disabled",
			wantAction: ediports.ServiceFailureEDIActionSkipped,
		},
		{
			name:       "no Lading project",
			source:     &shipment.Shipment{CustomerID: customerID},
			partners:   []*edi.EDIPartner{active()},
			flows:      [][]lading.Flow{nil},
			wantReason: "the partner has no active Lading version",
			wantAction: ediports.ServiceFailureEDIActionSkipped,
		},
		{
			name:       "flow sends nothing on a service failure",
			source:     &shipment.Shipment{CustomerID: customerID},
			partners:   []*edi.EDIPartner{active()},
			flows:      [][]lading.Flow{{}},
			wantReason: "the partner's flow sends no 214 on a service failure",
			wantAction: ediports.ServiceFailureEDIActionSkipped,
		},
		{
			name:       "two partners send one blocks",
			source:     &shipment.Shipment{CustomerID: customerID},
			partners:   []*edi.EDIPartner{active(), active()},
			flows:      [][]lading.Flow{sends, sends},
			wantReason: "more than one of the customer's EDI partners sends a 214 on a service failure",
			wantAction: ediports.ServiceFailureEDIActionBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			partnerRepo := edipartnerportsmock.NewMockEDIPartnerRepository(t)
			projects := ladingportsmock.NewMockLadingProjectRepository(t)
			orgs := tenantportsmock.NewMockOrganizationRepository(t)
			orgs.EXPECT().GetByID(mock.Anything, mock.Anything).
				Return(&tenant.Organization{Name: "Trenova", Timezone: "America/Chicago"}, nil).
				Maybe()
			svc := &Service{
				partnerRepo: partnerRepo,
				lading: edilading.NewRuntime(edilading.RuntimeParams{
					Logger:   zap.NewNop(),
					Projects: projects,
					Orgs:     orgs,
				}),
			}
			if !tt.source.CustomerID.IsNil() {
				partnerRepo.EXPECT().
					List(mock.Anything, mock.AnythingOfType("*edipartnerports.ListEDIPartnersRequest")).
					Return(&pagination.ListResult[*edi.EDIPartner]{
						Items: tt.partners,
						Total: len(tt.partners),
					}, nil).
					Once()
			}
			for index, flows := range tt.flows {
				partnerID := tt.partners[index].ID
				call := projects.EXPECT().GetRunnableVersions(
					mock.Anything,
					ladingports.GetRunnableLadingVersionRequest{
						TenantInfo: tenantInfo,
						PartnerID:  partnerID,
					},
				)
				if flows == nil {
					call.Return(nil, errortypes.NewNotFoundError("no project")).Once()
					continue
				}
				call.Return(&ladingports.RunnableLadingVersions{
					Project: &edi.LadingProject{ID: pulid.MustNew("ldp_"), EDIPartnerID: partnerID},
					Active: &edi.LadingVersion{
						ID:       pulid.MustNew("ldv_"),
						Status:   edi.LadingVersionStatusActive,
						Manifest: lading.Check{Flows: flows},
					},
				}, nil).Once()
			}

			candidate, result, err := svc.resolveServiceFailure214Candidate(
				t.Context(),
				&ediports.ServiceFailure214LifecycleRequest{
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
