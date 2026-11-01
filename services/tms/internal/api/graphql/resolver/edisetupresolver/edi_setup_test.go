package edisetupresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/edi/app/edivalidation"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/ediservicetest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/edimessageportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEDISetupOperations_CheckTheEDIPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("edip_").String()
	itemID := pulid.MustNew("edimi_").String()
	partnerInput := gqlmodel.EDIPartnerInput{Code: "ACME", Name: "Acme"}
	profileInput := gqlmodel.EDICommunicationProfileInput{
		Method: edi.ConnectionMethodSFTP,
		Name:   "Acme SFTP",
		Config: map[string]any{},
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"ediPartner", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).EDIPartner(ctx, id)
			return err
		}},
		{"pendingEdiConnections", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).PendingEDIConnections(ctx, nil)
			return err
		}},
		{
			"ediPartnerMappingProfile",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).EDIPartnerMappingProfile(ctx, id)
				return err
			},
		},
		{"inspectEdiCertificate", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).InspectEDICertificate(ctx, "pem")
			return err
		}},
		{"createEdiPartner", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateEDIPartner(ctx, partnerInput)
			return err
		}},
		{"updateEdiPartner", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateEDIPartner(ctx, id, partnerInput)
			return err
		}},
		{"createEdiConnection", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateEDIConnection(
				ctx,
				gqlmodel.CreateEDIConnectionInput{TargetOrganizationID: id},
			)
			return err
		}},
		{"acceptEdiConnection", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).AcceptEDIConnection(ctx, id)
			return err
		}},
		{"rejectEdiConnection", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).RejectEDIConnection(ctx, id, nil)
			return err
		}},
		{
			"createEdiCommunicationProfile",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateEDICommunicationProfile(ctx, profileInput)
				return err
			},
		},
		{
			"updateEdiCommunicationProfile",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateEDICommunicationProfile(
					ctx,
					id,
					profileInput,
				)
				return err
			},
		},
		{
			"testEdiCommunicationProfileConnection",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).TestEDICommunicationProfileConnection(ctx, id)
				return err
			},
		},
		{
			"pollEdiCommunicationProfile",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).PollEDICommunicationProfile(ctx, id)
				return err
			},
		},
		{
			"saveEdiMappingProfileItems",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).SaveEDIMappingProfileItems(ctx, id, nil)
				return err
			},
		},
		{
			"deleteEdiMappingProfileItem",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DeleteEDIMappingProfileItem(ctx, id, itemID)
				return err
			},
		},
		{
			"saveEdiPartnerMappingProfile",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).SaveEDIPartnerMappingProfile(ctx, id, nil)
				return err
			},
		},
		{
			"deleteEdiPartnerMappingItem",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DeleteEDIPartnerMappingItem(ctx, id, itemID)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(t, err, engine, permission.ResourceEDI, call.operation)
		})
	}
}

func TestCreateEDIPartner_DefaultsNewPartnersToEnabledAndKeepsExplicitFalse(t *testing.T) {
	t.Parallel()

	partnerRepo := edipartnerportsmock.NewMockEDIPartnerRepository(t)
	partnerRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *edi.EDIPartner) bool {
			return entity.Kind == edi.PartnerKindExternal &&
				entity.Status == domaintypes.StatusActive &&
				entity.Country == "US" &&
				entity.EnabledForInbound &&
				entity.EnabledForOutbound
		})).
		RunAndReturn(func(_ context.Context, entity *edi.EDIPartner) (*edi.EDIPartner, error) {
			entity.ID = pulid.MustNew("edip_")
			return entity, nil
		}).
		Once()
	partnerRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *edi.EDIPartner) bool {
			return entity.Code == "EXT2" &&
				!entity.EnabledForInbound &&
				!entity.EnabledForOutbound
		})).
		RunAndReturn(func(_ context.Context, entity *edi.EDIPartner) (*edi.EDIPartner, error) {
			entity.ID = pulid.MustNew("edip_")
			return entity, nil
		}).
		Once()
	services := ediservicetest.New(ediservicetest.Deps{
		Logger:             zap.NewNop(),
		PartnerRepo:        partnerRepo,
		DocumentTypeRepo:   edipartnerportsmock.NewMockEDIDocumentTypeRepository(t),
		TransactionSetRepo: edipartnerportsmock.NewMockEDITransactionSetRepository(t),
		ControlNumberRepo:  edipartnerportsmock.NewMockEDIControlNumberRepository(t),
		MessageRepo:        edimessageportsmock.NewMockEDIMessageRepository(t),
		Validator:          edivalidation.NewValidator(),
	})
	resolver := &MutationResolver{&Deps{
		Core:            &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		EdiSetupService: services.Setup,
	}}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)

	created, err := resolver.CreateEDIPartner(
		ctx,
		gqlmodel.EDIPartnerInput{Code: "EXT", Name: "External Partner"},
	)
	require.NoError(t, err)
	assert.Equal(t, authCtx.OrganizationID, created.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, created.BusinessUnitID)

	disabled := false
	_, err = resolver.CreateEDIPartner(ctx, gqlmodel.EDIPartnerInput{
		Code:               "EXT2",
		Name:               "External Partner 2",
		EnabledForInbound:  &disabled,
		EnabledForOutbound: &disabled,
	})
	require.NoError(t, err)
}

func TestPartnerFromInput_MapsTheInput(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	customerRaw := customerID.String()
	empty := ""
	kind := edi.PartnerKindInternal
	status := domaintypes.StatusInactive
	version := 3
	description := "Grocery shipper"

	entity, err := partnerFromInput(&gqlmodel.EDIPartnerInput{
		Kind:            &kind,
		Status:          &status,
		Code:            "ACME",
		Name:            "Acme",
		Description:     &description,
		CustomerID:      &customerRaw,
		EDIConnectionID: &empty,
		Settings:        map[string]any{"autoAccept": true},
		Version:         &version,
	}, false)
	require.NoError(t, err)

	assert.Equal(t, edi.PartnerKindInternal, entity.Kind)
	assert.Equal(t, domaintypes.StatusInactive, entity.Status)
	assert.Equal(t, customerID, entity.CustomerID)
	assert.True(t, entity.EDIConnectionID.IsNil())
	assert.Equal(t, "Grocery shipper", entity.Description)
	assert.Equal(t, map[string]any{"autoAccept": true}, entity.Settings)
	assert.Equal(t, int64(3), entity.Version)
	assert.False(t, entity.EnabledForInbound)
	assert.False(t, entity.EnabledForOutbound)
}

func TestPartnerFromInput_RefusesAMalformedID(t *testing.T) {
	t.Parallel()

	bad := "not-an-id"
	_, err := partnerFromInput(&gqlmodel.EDIPartnerInput{
		Code:       "ACME",
		Name:       "Acme",
		CustomerID: &bad,
	}, true)

	require.True(t, errortypes.IsError(err), "%v", err)
}

func TestCommunicationProfileRequestFromInput_CarriesSecretsApartFromConfig(t *testing.T) {
	t.Parallel()

	status := domaintypes.StatusActive
	partnerID := pulid.MustNew("edip_")
	partnerRaw := partnerID.String()
	req, err := communicationProfileRequestFromInput(&gqlmodel.EDICommunicationProfileInput{
		EDIPartnerID: &partnerRaw,
		Method:       edi.ConnectionMethodSFTP,
		Status:       &status,
		Name:         "Acme SFTP",
		Config:       map[string]any{"host": "sftp.acme.test"},
		Secrets:      map[string]any{"password": "hunter2", "privateKey": nil},
	})
	require.NoError(t, err)

	assert.Equal(t, partnerID, req.EDIPartnerID)
	assert.Equal(t, "Active", req.Status)
	assert.Equal(t, map[string]any{"host": "sftp.acme.test"}, req.Config)
	assert.Equal(t, map[string]string{"password": "hunter2", "privateKey": ""}, req.Secrets)
}

func TestCommunicationProfileRequestFromInput_RefusesASecretThatIsNotText(t *testing.T) {
	t.Parallel()

	_, err := communicationProfileRequestFromInput(&gqlmodel.EDICommunicationProfileInput{
		Method:  edi.ConnectionMethodSFTP,
		Name:    "Acme SFTP",
		Config:  map[string]any{},
		Secrets: map[string]any{"password": 42},
	})

	require.True(t, errortypes.IsError(err), "%v", err)
}

func TestConnectionRequestFromInput_MapsBothSidesOfTheConnection(t *testing.T) {
	t.Parallel()

	targetID := pulid.MustNew("org_")
	req, err := connectionRequestFromInput(&gqlmodel.CreateEDIConnectionInput{
		TargetOrganizationID: targetID.String(),
		Method:               edi.ConnectionMethodInternal,
		Capabilities: &gqlmodel.EDIConnectionCapabilitiesInput{
			LoadTenderOutbound: true,
			ShipmentStatus:     true,
		},
		SourcePartnerConfig: &gqlmodel.EDIConnectionPartnerConfigInput{
			Code:              "SRC",
			Name:              "Source",
			EnabledForInbound: true,
		},
		TargetPartnerConfig: &gqlmodel.EDIConnectionPartnerConfigInput{
			Code:               "TGT",
			Name:               "Target",
			EnabledForOutbound: true,
			Settings:           map[string]any{"autoAccept": false},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, targetID, req.TargetOrganizationID)
	assert.True(t, req.Capabilities.LoadTenderOutbound)
	assert.False(t, req.Capabilities.LoadTenderInbound)
	assert.Equal(t, "SRC", req.SourcePartnerConfig.Code)
	assert.True(t, req.SourcePartnerConfig.EnabledForInbound)
	assert.Equal(t, map[string]any{"autoAccept": false}, req.TargetPartnerConfig.Settings)
}

func TestPendingConnectionLimit(t *testing.T) {
	t.Parallel()

	tooMany, few, none := 500, 5, 0
	assert.Equal(t, defaultPendingConnections, pendingConnectionLimit(nil))
	assert.Equal(t, defaultPendingConnections, pendingConnectionLimit(&none))
	assert.Equal(t, 5, pendingConnectionLimit(&few))
	assert.Equal(t, maxPendingConnections, pendingConnectionLimit(&tooMany))
}
