//go:build integration

package ediconnectiondraft

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicommunicationprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediconnectiondraftrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicontrolnumberrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edidocumenttyperepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimappingprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnerdocumentprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnerrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnersettingrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editemplaterepository"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type seededTenant struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func realService(t *testing.T) (context.Context, *bun.DB, *Service, pagination.TenantInfo) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	registry := seeder.NewRegistry()
	seeds.Register(registry)
	_, err := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	).Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var org seededTenant
	require.NoError(t, db.NewSelect().
		Table("organizations").
		Column("id", "business_unit_id").
		Where("NOT EXISTS (SELECT 1 FROM edi_partners AS ep WHERE ep.organization_id = organizations.id)").
		Limit(1).
		Scan(ctx, &org))

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	templates := editemplaterepository.New(editemplaterepository.Params{DB: conn, Logger: logger})
	documentTypes := edidocumenttyperepository.New(edidocumenttyperepository.Params{DB: conn, Logger: logger})
	validator := ediservice.NewValidator()
	setup := ediservice.New(ediservice.Params{
		Logger:              logger,
		PartnerRepo:         edipartnerrepository.New(edipartnerrepository.Params{DB: conn, Logger: logger}),
		ProfileRepo:         edicommunicationprofilerepository.New(edicommunicationprofilerepository.Params{DB: conn, Logger: logger}),
		DocumentProfileRepo: edipartnerdocumentprofilerepository.New(edipartnerdocumentprofilerepository.Params{DB: conn, Logger: logger}),
		MappingProfileRepo:  edimappingprofilerepository.New(edimappingprofilerepository.Params{DB: conn, Logger: logger}),
		ControlNumberRepo:   edicontrolnumberrepository.New(edicontrolnumberrepository.Params{DB: conn, Logger: logger}),
		PartnerSettingRepo:  edipartnersettingrepository.New(edipartnersettingrepository.Params{DB: conn, Logger: logger}),
		TemplateRepo:        templates,
		DocumentTypeRepo:    documentTypes,
		Validator:           validator,
		DB:                  conn,
	})
	service := New(Params{
		Repo:          ediconnectiondraftrepository.New(ediconnectiondraftrepository.Params{DB: conn, Logger: logger}),
		Templates:     templates,
		DocumentTypes: documentTypes,
		Setup:         setup,
		Validator:     validator,
		DB:            conn,
		Logger:        logger,
	})
	return ctx, db, service, pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
}

func TestApplyConnectionDraft_WritesTheWholeConnectionAndARerunIsAChangeSet(t *testing.T) {
	ctx, db, service, tenant := realService(t)

	req := acmeRequest(t)
	req.TenantInfo = tenant
	draft, err := service.PlanConnectionDraft(ctx, req)
	require.NoError(t, err)
	require.True(t, draft.Ready, draft.Blockers)
	assert.Equal(t, ediports.PartnerMatchNew, draft.Partner.Match)

	actor := approver(req)
	applied, err := service.ApplyConnectionDraft(ctx, &ediports.ApplyConnectionDraftRequest{
		Draft:                  req,
		ExpectedDocumentDigest: draft.DocumentDigest,
		Actor:                  actor,
	})
	require.NoError(t, err)
	require.True(t, applied.Created)

	partner := new(edi.EDIPartner)
	require.NoError(t, db.NewSelect().Model(partner).Where("id = ?", applied.PartnerID).Scan(ctx))
	assert.Equal(t, "ACMERETAIL", partner.Code)
	assert.Equal(t, domaintypes.StatusInactive, partner.Status)
	assert.True(t, partner.EnabledForInbound)
	assert.True(t, partner.EnabledForOutbound)

	var transports []*edi.EDICommunicationProfile
	require.NoError(t, db.NewSelect().Model(&transports).Where("edi_partner_id = ?", partner.ID).Scan(ctx))
	require.Len(t, transports, 1)
	assert.Equal(t, edi.ConnectionMethodAS2, transports[0].Method)
	assert.Equal(t, domaintypes.StatusInactive, transports[0].Status)
	assert.Empty(t, transports[0].EncryptedSecrets)
	assert.Equal(t, "ACMERETAILAS2", transports[0].Config["partnerAS2Id"])

	var profiles []*edi.EDIPartnerDocumentProfile
	require.NoError(t, db.NewSelect().Model(&profiles).Where("edi_partner_id = ?", partner.ID).Scan(ctx))
	require.Len(t, profiles, 4)
	for _, profile := range profiles {
		assert.Equal(t, edi.DocumentStatusInactive, profile.Status)
		assert.True(t, profile.TemplateID.IsNotNil())
		if profile.Direction == edi.DocumentDirectionOutbound {
			assert.Equal(t, "007912345", profile.Envelope.InterchangeReceiverID)
			assert.True(t, profile.Acknowledgment.Expected)
		}
	}

	var sequences []*edi.EDIControlNumberSequence
	require.NoError(t, db.NewSelect().Model(&sequences).Where("edi_partner_id = ?", partner.ID).Scan(ctx))
	require.Len(t, sequences, 3)
	for _, sequence := range sequences {
		assert.Equal(t, int64(1000), sequence.NextValue)
	}

	revised := acmeRequest(t)
	revised.TenantInfo = tenant
	for index := range revised.Values {
		if revised.Values[index].Field == fieldPartnerContactName {
			revised.Values[index] = value(fieldPartnerContactName, "Whitfield", 1, "EDI Coordinator: Dana Whitfield")
		}
	}
	rerun, err := service.PlanConnectionDraft(ctx, revised)
	require.NoError(t, err)
	assert.Equal(t, ediports.PartnerMatchExisting, rerun.Partner.Match)
	assert.Equal(t, partner.ID, rerun.Partner.PartnerID)
	assert.Equal(t, ediports.ChangeKindChange, rerun.Partner.Change)
	assert.Equal(t, ediports.ChangeKindSame, rerun.Transport.Change)
	for _, profile := range rerun.DocumentProfiles {
		assert.Equal(t, ediports.ChangeKindSame, profile.Change, "%s %s", profile.TransactionSet, profile.Direction)
	}
	assert.Equal(t, 1, rerun.Changes)

	_, err = service.ApplyConnectionDraft(ctx, &ediports.ApplyConnectionDraftRequest{
		Draft:                  revised,
		ExpectedDocumentDigest: rerun.DocumentDigest,
		Actor:                  actor,
	})
	require.NoError(t, err)

	count, err := db.NewSelect().Model((*edi.EDIPartner)(nil)).
		Where("organization_id = ?", tenant.OrgID).
		Where("LOWER(code) = ?", "acmeretail").
		Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count, "a re-run never creates a second partner")
	require.NoError(t, db.NewSelect().Model(partner).Where("id = ?", partner.ID).Scan(ctx))
	assert.Equal(t, "Whitfield", partner.ContactName)
	assert.Equal(t, domaintypes.StatusInactive, partner.Status)

	again, err := service.PlanConnectionDraft(ctx, revised)
	require.NoError(t, err)
	assert.False(t, again.Ready)
	assert.Zero(t, again.Changes)
}
