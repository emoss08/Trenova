//go:build integration

package documenttemplaterepository_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/documenttemplaterepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type fixture struct {
	ctx         context.Context
	db          *bun.DB
	templates   repositories.DocumentTemplateRepository
	versions    repositories.DocumentTemplateVersionRepository
	assignments repositories.DocumentTemplateAssignmentRepository
	tenantInfo  pagination.TenantInfo
	userID      pulid.ID
}

func setup(t *testing.T) *fixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)

	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{
			SystemUserPassword: "integration-system-password",
		},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{
		Environment: common.EnvDevelopment,
		Force:       true,
	})
	require.NoError(t, err)

	var org struct {
		ID             pulid.ID `bun:"id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("organizations").
		Column("id", "business_unit_id").
		Order("created_at ASC").
		Limit(1).
		Scan(ctx, &org))

	var userID pulid.ID
	require.NoError(t, db.NewSelect().
		TableExpr("users").
		Column("id").
		Where("current_organization_id = ?", org.ID).
		Limit(1).
		Scan(ctx, &userID))

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()

	return &fixture{
		ctx: ctx,
		db:  db,
		templates: documenttemplaterepository.NewTemplateRepository(
			documenttemplaterepository.TemplateParams{DB: conn, Logger: logger},
		),
		versions: documenttemplaterepository.NewVersionRepository(
			documenttemplaterepository.VersionParams{DB: conn, Logger: logger},
		),
		assignments: documenttemplaterepository.NewAssignmentRepository(
			documenttemplaterepository.AssignmentParams{DB: conn, Logger: logger},
		),
		tenantInfo: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
		userID:     userID,
	}
}

// seededTemplate returns the draft the starter seed created for a kind.
func (f *fixture) seededTemplate(
	t *testing.T,
	kind documenttemplate.Kind,
) *documenttemplate.DocumentTemplate {
	t.Helper()

	template := new(documenttemplate.DocumentTemplate)
	require.NoError(t, f.db.NewSelect().
		Model(template).
		Relation("Versions").
		Where("dtpl.organization_id = ?", f.tenantInfo.OrgID).
		Where("dtpl.business_unit_id = ?", f.tenantInfo.BuID).
		Where("dtpl.kind = ?", kind).
		Limit(1).
		Scan(f.ctx))

	return template
}

func (f *fixture) customerID(t *testing.T) pulid.ID {
	t.Helper()

	var id pulid.ID
	require.NoError(t, f.db.NewSelect().
		TableExpr("customers").
		Column("id").
		Where("organization_id = ?", f.tenantInfo.OrgID).
		Where("business_unit_id = ?", f.tenantInfo.BuID).
		Limit(1).
		Scan(f.ctx, &id))

	return id
}

func (f *fixture) makeOrgDefault(t *testing.T, templateID pulid.ID) {
	t.Helper()

	_, err := f.db.NewUpdate().
		Model((*documenttemplate.DocumentTemplate)(nil)).
		Set("is_org_default = TRUE").
		Where("id = ?", templateID).
		Exec(f.ctx)
	require.NoError(t, err)
}

// Resolution is the load-bearing query: it decides what every customer of an
// organization receives. Each tier is asserted against real rows because the
// tie-break lives in SQL, not in Go.
func TestResolveWalksAllThreeTiers(t *testing.T) {
	f := setup(t)
	kind := documenttemplate.KindInvoicePDF

	// Tier three. The seed leaves every template a draft, so nothing is live and
	// the caller is told to render the built-in.
	row, err := f.templates.Resolve(f.ctx, &repositories.ResolveDocumentTemplateRequest{
		TenantInfo: f.tenantInfo,
		Kind:       kind,
	})
	require.NoError(t, err)
	require.Nil(t, row, "a draft-only organization must fall through to the built-in")

	orgTemplate := f.seededTemplate(t, kind)
	require.Len(t, orgTemplate.Versions, 1)
	f.makeOrgDefault(t, orgTemplate.ID)

	published, err := f.versions.Publish(
		f.ctx,
		&repositories.PublishDocumentTemplateVersionRequest{
			TenantInfo:    f.tenantInfo,
			VersionID:     orgTemplate.Versions[0].ID,
			PublishedByID: f.userID,
			PublishNotes:  "org default",
		},
	)
	require.NoError(t, err)
	require.Equal(t, documenttemplate.VersionStatusActive, published.Status)

	// Tier two.
	row, err = f.templates.Resolve(f.ctx, &repositories.ResolveDocumentTemplateRequest{
		TenantInfo: f.tenantInfo,
		Kind:       kind,
	})
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, documenttemplate.SourceOrgDefault, row.Source)
	require.Equal(t, orgTemplate.ID, row.Template.ID)
	require.NotEmpty(t, row.Version.BodyHTML, "resolution must carry renderable content")

	customerID := f.customerID(t)

	// Tier one: a second template of the same kind, assigned to one customer.
	// It is deliberately not the organization default, which is what makes the
	// two tiers distinguishable.
	assignedTemplate := f.publishSecondTemplate(t, kind)

	_, err = f.assignments.Assign(f.ctx, &documenttemplate.DocumentTemplateAssignment{
		OrganizationID: f.tenantInfo.OrgID,
		BusinessUnitID: f.tenantInfo.BuID,
		Kind:           kind,
		TemplateID:     assignedTemplate.ID,
		CustomerID:     customerID,
		AssignedByID:   &f.userID,
	})
	require.NoError(t, err)

	row, err = f.templates.Resolve(f.ctx, &repositories.ResolveDocumentTemplateRequest{
		TenantInfo: f.tenantInfo,
		Kind:       kind,
		CustomerID: &customerID,
	})
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, documenttemplate.SourceAssigned, row.Source)
	require.Equal(t, assignedTemplate.ID, row.Template.ID)

	// A customer without an assignment must still land on the organization
	// default, not on someone else's override.
	other := f.otherCustomerID(t, customerID)
	row, err = f.templates.Resolve(f.ctx, &repositories.ResolveDocumentTemplateRequest{
		TenantInfo: f.tenantInfo,
		Kind:       kind,
		CustomerID: &other,
	})
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, documenttemplate.SourceOrgDefault, row.Source)
	require.Equal(t, orgTemplate.ID, row.Template.ID)

	// Unassigning must return the customer to the organization default rather
	// than to the built-in.
	require.NoError(t, f.assignments.Unassign(
		f.ctx,
		&repositories.UnassignDocumentTemplateRequest{
			TenantInfo: f.tenantInfo,
			Kind:       kind,
			CustomerID: customerID,
		},
	))

	row, err = f.templates.Resolve(f.ctx, &repositories.ResolveDocumentTemplateRequest{
		TenantInfo: f.tenantInfo,
		Kind:       kind,
		CustomerID: &customerID,
	})
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, documenttemplate.SourceOrgDefault, row.Source)
}

func (f *fixture) publishSecondTemplate(
	t *testing.T,
	kind documenttemplate.Kind,
) *documenttemplate.DocumentTemplate {
	t.Helper()

	template, err := f.templates.Create(f.ctx, &documenttemplate.DocumentTemplate{
		OrganizationID: f.tenantInfo.OrgID,
		BusinessUnitID: f.tenantInfo.BuID,
		Kind:           kind,
		Code:           "ACME_INVOICE",
		Name:           "Acme Invoice",
	})
	require.NoError(t, err)

	version, err := f.versions.Create(
		f.ctx,
		&repositories.CreateDocumentTemplateVersionRequest{
			Version: &documenttemplate.DocumentTemplateVersion{
				OrganizationID: f.tenantInfo.OrgID,
				BusinessUnitID: f.tenantInfo.BuID,
				TemplateID:     template.ID,
				BodyHTML:       "<p>Acme invoice</p>",
				ContentHash:    "acmehash",
			},
		},
	)
	require.NoError(t, err)

	_, err = f.versions.Publish(f.ctx, &repositories.PublishDocumentTemplateVersionRequest{
		TenantInfo:    f.tenantInfo,
		VersionID:     version.ID,
		PublishedByID: f.userID,
	})
	require.NoError(t, err)

	return template
}

func (f *fixture) otherCustomerID(t *testing.T, exclude pulid.ID) pulid.ID {
	t.Helper()

	var id pulid.ID
	require.NoError(t, f.db.NewSelect().
		TableExpr("customers").
		Column("id").
		Where("organization_id = ?", f.tenantInfo.OrgID).
		Where("business_unit_id = ?", f.tenantInfo.BuID).
		Where("id <> ?", exclude).
		Limit(1).
		Scan(f.ctx, &id))

	return id
}

// Two organization defaults for one kind would make which template a customer
// receives depend on row order. The service refuses it; this asserts the
// database does too, because the service is not the only writer.
func TestOnlyOneOrgDefaultPerKindIsAccepted(t *testing.T) {
	f := setup(t)
	kind := documenttemplate.KindInvoiceEmail

	first := f.seededTemplate(t, kind)
	f.makeOrgDefault(t, first.ID)

	second, err := f.templates.Create(f.ctx, &documenttemplate.DocumentTemplate{
		OrganizationID: f.tenantInfo.OrgID,
		BusinessUnitID: f.tenantInfo.BuID,
		Kind:           kind,
		Code:           "SECOND_INVOICE_EMAIL",
		Name:           "Second Invoice Email",
	})
	require.NoError(t, err)

	_, err = f.db.NewUpdate().
		Model((*documenttemplate.DocumentTemplate)(nil)).
		Set("is_org_default = TRUE").
		Where("id = ?", second.ID).
		Exec(f.ctx)
	require.Error(t, err, "a second organization default for one kind must be rejected")
	require.Contains(t, err.Error(), "uq_document_templates_org_default")
}

// Two drafts would make "the draft" ambiguous in the editor.
func TestOnlyOneDraftPerTemplateIsAccepted(t *testing.T) {
	f := setup(t)

	template := f.seededTemplate(t, documenttemplate.KindReportPDF)
	require.Len(t, template.Versions, 1)
	require.Equal(t, documenttemplate.VersionStatusDraft, template.Versions[0].Status)

	_, err := f.versions.Create(f.ctx, &repositories.CreateDocumentTemplateVersionRequest{
		Version: &documenttemplate.DocumentTemplateVersion{
			OrganizationID: f.tenantInfo.OrgID,
			BusinessUnitID: f.tenantInfo.BuID,
			TemplateID:     template.ID,
			BodyHTML:       "<p>second draft</p>",
			ContentHash:    "secondhash",
		},
	})
	require.Error(t, err, "a second draft on one template must be rejected")
	require.Contains(t, err.Error(), "uq_document_template_versions_one_draft")
}

// A version that produced a document a customer received must stay readable, or
// "what exactly did we send in March" stops being answerable — which is the one
// question an invoice or detention dispute always asks.
func TestAVersionThatProducedADocumentCannotBeDeleted(t *testing.T) {
	f := setup(t)

	template := f.seededTemplate(t, documenttemplate.KindDetentionNoticePDF)
	version := template.Versions[0]

	generated := &documenttemplate.GeneratedDocument{
		OrganizationID:    f.tenantInfo.OrgID,
		BusinessUnitID:    f.tenantInfo.BuID,
		Kind:              documenttemplate.KindDetentionNoticePDF,
		TemplateID:        &template.ID,
		TemplateVersionID: &version.ID,
		ReferenceType:     "detention_occurrence",
		ReferenceID:       pulid.MustNew("dto_"),
		FileName:          "notice.pdf",
		Status:            documenttemplate.GenerationStatusPending,
	}
	_, err := f.db.NewInsert().Model(generated).Exec(f.ctx)
	require.NoError(t, err)

	// The repository refuses a non-draft delete, so this drives the constraint
	// directly to prove the database is the backstop and not just the service.
	_, err = f.db.NewDelete().
		Model((*documenttemplate.DocumentTemplateVersion)(nil)).
		Where("id = ?", version.ID).
		Exec(f.ctx)
	require.Error(t, err, "deleting a version with output attributed to it must be refused")
	require.Contains(t, err.Error(), "fk_generated_documents_template_version")
}

// A completed generated document with no stored file would be a row promising
// bytes nobody can fetch.
func TestACompletedGeneratedDocumentMustRecordWhereItLanded(t *testing.T) {
	f := setup(t)

	generated := &documenttemplate.GeneratedDocument{
		OrganizationID: f.tenantInfo.OrgID,
		BusinessUnitID: f.tenantInfo.BuID,
		Kind:           documenttemplate.KindInvoicePDF,
		ReferenceType:  "invoice",
		ReferenceID:    pulid.MustNew("inv_"),
		FileName:       "invoice.pdf",
		Status:         documenttemplate.GenerationStatusCompleted,
	}

	_, err := f.db.NewInsert().Model(generated).Exec(f.ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "chk_generated_documents_completed")
}

// The template names its live version and the version names its template, so
// the schema carries a circular foreign key. A down migration that does not
// break the cycle first leaves a group rollback unable to re-apply — and the
// failure only shows up on the next deploy that rolls back.
func TestTheMigrationRollsBackAndForwardThroughTheCircularForeignKey(t *testing.T) {
	f := setup(t)

	const stem = "20260902000000_document_templates"
	runMigrationFile(t, f.ctx, f.db, stem+".tx.down.sql")

	var exists bool
	require.NoError(t, f.db.NewRaw(
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables
		 WHERE table_name = 'document_template_versions')`,
	).Scan(f.ctx, &exists))
	require.False(t, exists, "the down migration must remove the versions table")

	runMigrationFile(t, f.ctx, f.db, stem+".tx.up.sql")

	require.NoError(t, f.db.NewRaw(
		`SELECT EXISTS (SELECT 1 FROM pg_constraint
		 WHERE conname = 'fk_document_templates_active_version')`,
	).Scan(f.ctx, &exists))
	require.True(t, exists, "re-applying must restore the circular foreign key")
}

// runMigrationFile executes one migration exactly as the migrator would, split
// on the same marker.
func runMigrationFile(t *testing.T, ctx context.Context, db *bun.DB, name string) {
	t.Helper()

	path := filepath.Join("internal", "infrastructure", "postgres", "migrations", name)
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "read migration %s", name)

	for _, statement := range strings.Split(string(raw), "--bun:split") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		_, err = db.ExecContext(ctx, statement)
		require.NoError(t, err, "apply statement from %s:\n%s", name, statement)
	}
}
