//go:build integration

package ediconnectiondraft

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicommunicationprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediconnectiondraftrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicontrolnumberrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edidocumenttyperepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediladingrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimappingprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnerrepository"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument/edidocumentdb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject/ladingprojectdb"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func realService(t *testing.T) (context.Context, *keel.DB, *Service, pagination.TenantInfo) {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var orgID, buID string
	require.NoError(t, db.Pool().QueryRow(ctx, `SELECT id, business_unit_id FROM organizations
		WHERE NOT EXISTS (SELECT 1 FROM edi_partners ep WHERE ep.organization_id = organizations.id)
		ORDER BY id LIMIT 1`).Scan(&orgID, &buID))
	scope := editest.TenantOf(
		ctx,
		pagination.TenantInfo{OrgID: pulid.ID(orgID), BuID: pulid.ID(buID)},
	)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	documentTypes := edidocumenttyperepository.New(
		edidocumenttyperepository.Params{DB: conn, Logger: logger},
	)
	partners := edipartnerrepository.New(edipartnerrepository.Params{DB: conn, Logger: logger})
	validator := ediservice.NewValidator()
	setup := ediservice.New(ediservice.Params{
		Logger:      logger,
		PartnerRepo: partners,
		ProfileRepo: edicommunicationprofilerepository.New(
			edicommunicationprofilerepository.Params{DB: conn, Logger: logger},
		),
		MappingProfileRepo: edimappingprofilerepository.New(
			edimappingprofilerepository.Params{DB: conn, Logger: logger},
		),
		ControlNumberRepo: edicontrolnumberrepository.New(
			edicontrolnumberrepository.Params{DB: conn, Logger: logger},
		),
		DocumentTypeRepo: documentTypes,
		Validator:        validator,
		DB:               conn,
		Savepoints:       conn,
	})
	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{Name: "Trenova Logistics", ScacCode: "TRNV", Timezone: "America/Chicago"}, nil).
		Maybe()
	ladingRepos := ediladingrepository.Params{DB: conn, Logger: logger}
	runtime := edilading.NewRuntime(edilading.RuntimeParams{
		Logger:   logger,
		Projects: ediladingrepository.NewProjects(ladingRepos),
		Orgs:     orgs,
	})
	projects := edilading.NewProjects(edilading.ProjectsParams{
		Logger:   logger,
		Runtime:  runtime,
		Projects: ediladingrepository.NewProjects(ladingRepos),
		Replays:  ediladingrepository.NewReplays(ladingRepos),
		Partners: partners,
	})
	service := New(Params{
		Repo: ediconnectiondraftrepository.New(
			ediconnectiondraftrepository.Params{DB: conn, Logger: logger},
		),
		DocumentTypes: documentTypes,
		Setup:         setup,
		Validator:     validator,
		Lading:        runtime,
		Projects:      projects,
		DB:            conn,
		Logger:        logger,
	})
	return scope.Ctx, db, service, scope.Info
}

func TestApplyConnectionDraft_WritesTheWholeConnectionAndARerunIsAChangeSet(t *testing.T) {
	ctx, db, service, tenant := realService(t)

	req := acmeRequest(t)
	req.TenantInfo = tenant
	draft, err := service.PlanConnectionDraft(ctx, req)
	require.NoError(t, err)
	require.True(t, draft.Ready, draft.Blockers)
	assert.Equal(t, ediconnectiondraftports.PartnerMatchNew, draft.Partner.Match)

	actor := approver(req)
	applied, err := service.ApplyConnectionDraft(
		ctx,
		&ediconnectiondraftports.ApplyConnectionDraftRequest{
			Draft:                  req,
			ExpectedDocumentDigest: draft.DocumentDigest,
			Actor:                  actor,
		},
	)
	require.NoError(t, err)
	require.True(t, applied.Created)

	ep := edidb.EDIPartners
	partner, err := ep.Get(ctx, db, applied.PartnerID)
	require.NoError(t, err)
	assert.Equal(t, "ACMERETAIL", partner.Code)
	assert.Equal(t, domaintypes.StatusInactive, partner.Status)
	assert.True(t, partner.EnabledForInbound)
	assert.True(t, partner.EnabledForOutbound)

	cp := edidb.EDICommunicationProfiles
	transports, err := cp.Where(cp.EDIPartnerID.Eq(partner.ID)).All(ctx, db)
	require.NoError(t, err)
	require.Len(t, transports, 1)
	assert.Equal(t, edi.ConnectionMethodAS2, transports[0].Method)
	assert.Equal(t, domaintypes.StatusInactive, transports[0].Status)
	assert.Empty(t, transports[0].EncryptedSecrets)
	assert.Equal(t, "ACMERETAILAS2", transports[0].Config["partnerAS2Id"])

	lp := ladingprojectdb.LadingProjects
	project, err := lp.Where(lp.EDIPartnerID.Eq(partner.ID)).One(ctx, db)
	require.NoError(t, err)
	assert.True(
		t,
		project.ActiveVersionID.IsNil(),
		"a drafted project runs nothing until it is certified",
	)
	lv := ladingprojectdb.LadingVersions
	versions, err := lv.Where(lv.ProjectID.Eq(project.ID)).All(ctx, db)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	drafted := versions[0]
	assert.Equal(t, ladingproject.LadingVersionStatusDraft, drafted.Status)
	assert.Equal(t, "Drafted from acme-retail-edi-guide.pdf", drafted.Notes)
	paths := make([]string, 0, len(drafted.Files))
	for _, file := range drafted.Files {
		paths = append(paths, file.Path)
	}
	assert.Contains(t, paths, "guides/214.lading")
	assert.Contains(t, paths, "guides/204.lading")
	require.NotNil(t, drafted.Manifest.Partner)
	require.NotNil(t, drafted.Manifest.Partner.Interchange)
	assert.Equal(t, "007912345", drafted.Manifest.Partner.Interchange.Them.ID)

	cn := edidocumentdb.EDIControlNumberSequences
	sequences, err := cn.Where(cn.EDIPartnerID.Eq(partner.ID)).All(ctx, db)
	require.NoError(t, err)
	require.Len(t, sequences, 3)
	for _, sequence := range sequences {
		assert.Equal(t, int64(1000), sequence.NextValue)
	}

	revised := acmeRequest(t)
	revised.TenantInfo = tenant
	for index := range revised.Values {
		if revised.Values[index].Field == fieldPartnerContactName {
			revised.Values[index] = value(
				fieldPartnerContactName,
				"Whitfield",
				1,
				"EDI Coordinator: Dana Whitfield",
			)
		}
	}
	rerun, err := service.PlanConnectionDraft(ctx, revised)
	require.NoError(t, err)
	assert.Equal(t, ediconnectiondraftports.PartnerMatchExisting, rerun.Partner.Match)
	assert.Equal(t, partner.ID, rerun.Partner.PartnerID)
	assert.Equal(t, ediconnectiondraftports.ChangeKindChange, rerun.Partner.Change)
	assert.Equal(t, ediconnectiondraftports.ChangeKindSame, rerun.Transport.Change)
	require.NotNil(t, rerun.Project)
	assert.Equal(t, project.ID, rerun.Project.ProjectID)
	assert.Equal(
		t,
		ediconnectiondraftports.ChangeKindSame,
		rerun.Project.Change,
		"the drafted project already says this",
	)
	assert.Equal(t, 1, rerun.Changes)

	_, err = service.ApplyConnectionDraft(ctx, &ediconnectiondraftports.ApplyConnectionDraftRequest{
		Draft:                  revised,
		ExpectedDocumentDigest: rerun.DocumentDigest,
		Actor:                  actor,
	})
	require.NoError(t, err)

	count, err := ep.Where(ep.Code.Lower().Eq("acmeretail")).Count(ctx, db)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count, "a re-run never creates a second partner")
	partner, err = ep.Get(ctx, db, partner.ID)
	require.NoError(t, err)
	assert.Equal(t, "Whitfield", partner.ContactName)
	assert.Equal(t, domaintypes.StatusInactive, partner.Status)

	again, err := service.PlanConnectionDraft(ctx, revised)
	require.NoError(t, err)
	assert.False(t, again.Ready)
	assert.Zero(t, again.Changes)
}
