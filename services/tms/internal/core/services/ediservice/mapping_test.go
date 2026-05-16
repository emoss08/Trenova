package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMappingResolutionIndex(t *testing.T) {
	sourceID := pulid.MustNew("cus_")
	targetID := pulid.MustNew("cus_")
	unresolvedID := pulid.MustNew("st_")

	index := resolutionIndex([]edi.MappingResolution{
		{
			EntityType: edi.MappingEntityTypeCustomer,
			SourceID:   sourceID,
			TargetID:   targetID,
			Resolved:   true,
		},
		{
			EntityType: edi.MappingEntityTypeServiceType,
			SourceID:   unresolvedID,
			Resolved:   false,
		},
	})

	mapped, ok := mappedID(index, edi.MappingEntityTypeCustomer, sourceID)
	require.True(t, ok)
	require.Equal(t, targetID, mapped)

	_, ok = mappedID(index, edi.MappingEntityTypeServiceType, unresolvedID)
	require.False(t, ok)
}

func TestRequiredEntityTypesAreStable(t *testing.T) {
	required := map[edi.MappingEntityType][]pulid.ID{
		edi.MappingEntityTypeLocation:        {pulid.MustNew("loc_")},
		edi.MappingEntityTypeCustomer:        {pulid.MustNew("cus_")},
		edi.MappingEntityTypeFormulaTemplate: {pulid.MustNew("ft_")},
	}

	require.Equal(t, []edi.MappingEntityType{
		edi.MappingEntityTypeCustomer,
		edi.MappingEntityTypeFormulaTemplate,
		edi.MappingEntityTypeLocation,
	}, requiredEntityTypes(required))
}

func TestBuildMappingPreviewUsesTenderSourceLabelsForUnresolvedRows(t *testing.T) {
	customerID := pulid.MustNew("cus_")
	locationID := pulid.MustNew("loc_")
	partnerID := pulid.MustNew("edip_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	partnerRepo := mocks.NewMockEDIPartnerRepository(t)
	partnerRepo.EXPECT().
		GetMappingItems(mock.Anything, mock.MatchedBy(func(req repositories.GetMappingItemsRequest) bool {
			return req.PartnerID == partnerID &&
				req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID
		})).
		Return([]*edi.EDIMappingProfileItem{}, nil)

	service := &Service{partnerRepo: partnerRepo}
	preview, err := service.buildMappingPreview(
		t.Context(),
		&edi.EDIPartner{
			ID:             partnerID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		edi.LoadTenderPayload{
			CustomerID:    customerID,
			CustomerLabel: "ACME - Acme Logistics",
			Moves: []edi.LoadTenderMove{
				{
					Stops: []edi.LoadTenderStop{
						{
							LocationID:    locationID,
							LocationLabel: "DAL - Dallas Terminal",
						},
					},
				},
			},
			RequiredMappingEntityIDs: map[edi.MappingEntityType][]pulid.ID{
				edi.MappingEntityTypeCustomer: {customerID},
				edi.MappingEntityTypeLocation: {locationID},
			},
		},
		nil,
	)

	require.NoError(t, err)
	require.Len(t, preview.Unresolved, 2)
	require.Equal(t, "ACME - Acme Logistics", preview.Unresolved[0].SourceLabel)
	require.Equal(t, "DAL - Dallas Terminal", preview.Unresolved[1].SourceLabel)
}

func TestBuildMappingPreviewKeepsSavedTargetLabelForResolvedRows(t *testing.T) {
	sourceID := pulid.MustNew("cus_")
	targetID := pulid.MustNew("cus_")
	partnerID := pulid.MustNew("edip_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	partnerRepo := mocks.NewMockEDIPartnerRepository(t)
	partnerRepo.EXPECT().
		GetMappingItems(mock.Anything, mock.Anything).
		Return([]*edi.EDIMappingProfileItem{
			{
				EntityType:  edi.MappingEntityTypeCustomer,
				SourceID:    sourceID,
				SourceLabel: "Saved source label",
				TargetID:    targetID,
				TargetLabel: "Receiving customer",
			},
		}, nil)

	service := &Service{partnerRepo: partnerRepo}
	preview, err := service.buildMappingPreview(
		t.Context(),
		&edi.EDIPartner{
			ID:             partnerID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		edi.LoadTenderPayload{
			CustomerID:    sourceID,
			CustomerLabel: "Tender source label",
			RequiredMappingEntityIDs: map[edi.MappingEntityType][]pulid.ID{
				edi.MappingEntityTypeCustomer: {sourceID},
			},
		},
		nil,
	)

	require.NoError(t, err)
	require.Len(t, preview.Resolved, 1)
	require.Equal(t, "Saved source label", preview.Resolved[0].SourceLabel)
	require.Equal(t, "Receiving customer", preview.Resolved[0].TargetLabel)
	require.Equal(t, targetID, preview.Resolved[0].TargetID)
}
