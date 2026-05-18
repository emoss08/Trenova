package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestService_CreatePartner_AllowsExternalWithoutInternalOrganization(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEDIPartnerRepository(t)
	svc := New(Params{
		Logger:      zap.NewNop(),
		PartnerRepo: repo,
		Validator:   NewValidator(),
	})
	partner := testExternalPartner()
	partner.EnabledForInbound = false
	partner.EnabledForOutbound = false

	repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *edi.EDIPartner) bool {
			return entity.Kind == edi.PartnerKindExternal &&
				entity.InternalOrganizationID.IsNil() &&
				!entity.EnabledForInbound &&
				!entity.EnabledForOutbound
		})).
		Return(partner, nil).
		Once()

	created, err := svc.CreatePartner(t.Context(), partner, nil)

	require.NoError(t, err)
	require.False(t, created.EnabledForInbound)
	require.False(t, created.EnabledForOutbound)
}

func TestService_CreatePartner_RejectsInternalWithoutInternalOrganization(t *testing.T) {
	t.Parallel()

	svc := New(Params{
		Logger:    zap.NewNop(),
		Validator: NewValidator(),
	})
	partner := testExternalPartner()
	partner.Kind = edi.PartnerKindInternal

	created, err := svc.CreatePartner(t.Context(), partner, nil)

	require.Nil(t, created)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Internal organization is required")
}

func TestService_UpdatePartner_AllowsExternalWithoutInternalOrganization(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEDIPartnerRepository(t)
	svc := New(Params{
		Logger:      zap.NewNop(),
		PartnerRepo: repo,
		Validator:   NewValidator(),
	})
	partner := testExternalPartner()
	partner.ID = pulid.MustNew("edip_")
	partner.EnabledForInbound = false
	partner.EnabledForOutbound = false

	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetEDIPartnerByIDRequest{
			ID: partner.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: partner.OrganizationID,
				BuID:  partner.BusinessUnitID,
			},
		}).
		Return(testExternalPartner(), nil).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(entity *edi.EDIPartner) bool {
			return entity.ID == partner.ID &&
				entity.InternalOrganizationID.IsNil() &&
				!entity.EnabledForInbound &&
				!entity.EnabledForOutbound
		})).
		Return(partner, nil).
		Once()

	updated, err := svc.UpdatePartner(t.Context(), partner, nil)

	require.NoError(t, err)
	require.False(t, updated.EnabledForInbound)
	require.False(t, updated.EnabledForOutbound)
}

func testExternalPartner() *edi.EDIPartner {
	return &edi.EDIPartner{
		BusinessUnitID:     pulid.MustNew("bu_"),
		OrganizationID:     pulid.MustNew("org_"),
		Kind:               edi.PartnerKindExternal,
		Status:             domaintypes.StatusActive,
		Code:               "EXT",
		Name:               "External Partner",
		Country:            "US",
		EnabledForInbound:  true,
		EnabledForOutbound: true,
		Settings:           map[string]any{},
	}
}
