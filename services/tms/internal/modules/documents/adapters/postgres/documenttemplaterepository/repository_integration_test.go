//go:build integration

package documenttemplaterepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documenttemplaterepository"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate/documenttemplatedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/generateddocument"
	"github.com/emoss08/trenova/internal/modules/documents/domain/generateddocument/generateddocumentdb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fixture struct {
	ctx         context.Context
	db          *keel.DB
	templates   documenttemplateports.DocumentTemplateRepository
	versions    documenttemplateports.DocumentTemplateVersionRepository
	assignments documenttemplateports.DocumentTemplateAssignmentRepository
	tenantInfo  pagination.TenantInfo
	userID      pulid.ID
}

type orgRow struct {
	ID             pulid.ID
	BusinessUnitID pulid.ID
}

func setup(t *testing.T) *fixture {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	o := tenantdb.Organizations
	org, err := keel.IntoOne[orgRow](
		dbscope.WithSystem(ctx, "find the first seeded organization"),
		db,
		o.AllTenants().Pick(o.ID, o.BusinessUnitID).Order(o.CreatedAt.Asc()).Limit(1),
	)
	require.NoError(t, err)

	tenantInfo := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())

	u := tenantdb.Users
	userID, err := keel.IntoOne[pulid.ID](
		ctx,
		db,
		u.Pick(u.ID).Where(u.CurrentOrganizationID.Eq(org.ID)).Limit(1),
	)
	require.NoError(t, err)

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
		tenantInfo: tenantInfo,
		userID:     userID,
	}
}

func (f *fixture) seededTemplate(
	t *testing.T,
	kind documenttemplate.Kind,
) *documenttemplate.DocumentTemplate {
	t.Helper()

	dt := documenttemplatedb.DocumentTemplates
	template, err := dt.Where(dt.Kind.Eq(kind)).
		Load(dt.Versions()).
		Order(dt.ID.Asc()).
		Limit(1).
		One(f.ctx, f.db)
	require.NoError(t, err)

	return template
}

func (f *fixture) customerID(t *testing.T) pulid.ID {
	t.Helper()

	c := customerdb.Customers
	id, err := keel.IntoOne[pulid.ID](f.ctx, f.db, c.Pick(c.ID).Order(c.ID.Asc()).Limit(1))
	require.NoError(t, err)

	return id
}

func (f *fixture) makeOrgDefault(t *testing.T, templateID pulid.ID) {
	t.Helper()

	require.NoError(t, f.setOrgDefault(templateID))
}

func (f *fixture) setOrgDefault(templateID pulid.ID) error {
	dt := documenttemplatedb.DocumentTemplates
	_, err := dt.Where(dt.ID.Eq(templateID)).Update(f.ctx, f.db, dt.IsOrgDefault.To(true))

	return err
}

// Resolution is the load-bearing query: it decides what every customer of an
// organization receives. Each tier is asserted against real rows because the
// tie-break lives in SQL, not in Go.
func TestResolveWalksAllThreeTiers(t *testing.T) {
	f := setup(t)
	kind := documenttemplate.KindInvoicePDF

	// Tier three. The seed leaves every template a draft, so nothing is live and
	// the caller is told to render the built-in.
	row, err := f.templates.Resolve(f.ctx, &documenttemplateports.ResolveDocumentTemplateRequest{
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
		&documenttemplateports.PublishDocumentTemplateVersionRequest{
			TenantInfo:    f.tenantInfo,
			VersionID:     orgTemplate.Versions[0].ID,
			PublishedByID: f.userID,
			PublishNotes:  "org default",
		},
	)
	require.NoError(t, err)
	require.Equal(t, documenttemplate.VersionStatusActive, published.Status)

	// Tier two.
	row, err = f.templates.Resolve(f.ctx, &documenttemplateports.ResolveDocumentTemplateRequest{
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

	row, err = f.templates.Resolve(f.ctx, &documenttemplateports.ResolveDocumentTemplateRequest{
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
	row, err = f.templates.Resolve(f.ctx, &documenttemplateports.ResolveDocumentTemplateRequest{
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
		&documenttemplateports.UnassignDocumentTemplateRequest{
			TenantInfo: f.tenantInfo,
			Kind:       kind,
			CustomerID: customerID,
		},
	))

	row, err = f.templates.Resolve(f.ctx, &documenttemplateports.ResolveDocumentTemplateRequest{
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
		&documenttemplateports.CreateDocumentTemplateVersionRequest{
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

	_, err = f.versions.Publish(f.ctx, &documenttemplateports.PublishDocumentTemplateVersionRequest{
		TenantInfo:    f.tenantInfo,
		VersionID:     version.ID,
		PublishedByID: f.userID,
	})
	require.NoError(t, err)

	return template
}

func (f *fixture) otherCustomerID(t *testing.T, exclude pulid.ID) pulid.ID {
	t.Helper()

	c := customerdb.Customers
	id, err := keel.IntoOne[pulid.ID](
		f.ctx,
		f.db,
		c.Pick(c.ID).Where(c.ID.Ne(exclude)).Order(c.ID.Asc()).Limit(1),
	)
	require.NoError(t, err)

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

	err = f.setOrgDefault(second.ID)
	require.Error(t, err, "a second organization default for one kind must be rejected")
	require.Contains(t, err.Error(), "uq_document_templates_org_default")
}

// Two drafts would make "the draft" ambiguous in the editor.
func TestOnlyOneDraftPerTemplateIsAccepted(t *testing.T) {
	f := setup(t)

	template := f.seededTemplate(t, documenttemplate.KindReportPDF)
	require.Len(t, template.Versions, 1)
	require.Equal(t, documenttemplate.VersionStatusDraft, template.Versions[0].Status)

	_, err := f.versions.Create(f.ctx, &documenttemplateports.CreateDocumentTemplateVersionRequest{
		Version: &documenttemplate.DocumentTemplateVersion{
			OrganizationID: f.tenantInfo.OrgID,
			BusinessUnitID: f.tenantInfo.BuID,
			TemplateID:     template.ID,
			BodyHTML:       "<p>second draft</p>",
			ContentHash:    "secondhash",
		},
	})
	require.Error(t, err, "a second draft on one template must be rejected")
	require.Contains(t, err.Error(), "This template already has a draft")
}

// A version that produced a document a customer received must stay readable, or
// "what exactly did we send in March" stops being answerable — which is the one
// question an invoice or detention dispute always asks.
func TestAVersionThatProducedADocumentCannotBeDeleted(t *testing.T) {
	f := setup(t)

	template := f.seededTemplate(t, documenttemplate.KindDetentionNoticePDF)
	version := template.Versions[0]

	generated := &generateddocument.GeneratedDocument{
		OrganizationID:    f.tenantInfo.OrgID,
		BusinessUnitID:    f.tenantInfo.BuID,
		Kind:              documenttemplate.KindDetentionNoticePDF,
		TemplateID:        &template.ID,
		TemplateVersionID: &version.ID,
		ReferenceType:     "detention_occurrence",
		ReferenceID:       pulid.MustNew("dto_"),
		FileName:          "notice.pdf",
		Status:            generateddocument.GenerationStatusPending,
	}
	err := generateddocumentdb.GeneratedDocuments.Create(f.ctx, f.db, generated)
	require.NoError(t, err)

	// The repository refuses a non-draft delete, so this drives the constraint
	// directly to prove the database is the backstop and not just the service.
	dtv := documenttemplatedb.DocumentTemplateVersions
	_, err = dtv.Where(dtv.ID.Eq(version.ID)).Delete(f.ctx, f.db)
	require.Error(t, err, "deleting a version with output attributed to it must be refused")
	require.Contains(t, err.Error(), "fk_generated_documents_template_version")
}

// A completed generated document with no stored file would be a row promising
// bytes nobody can fetch.
func TestACompletedGeneratedDocumentMustRecordWhereItLanded(t *testing.T) {
	f := setup(t)

	generated := &generateddocument.GeneratedDocument{
		OrganizationID: f.tenantInfo.OrgID,
		BusinessUnitID: f.tenantInfo.BuID,
		Kind:           documenttemplate.KindInvoicePDF,
		ReferenceType:  "invoice",
		ReferenceID:    pulid.MustNew("inv_"),
		FileName:       "invoice.pdf",
		Status:         generateddocument.GenerationStatusCompleted,
	}

	err := generateddocumentdb.GeneratedDocuments.Create(f.ctx, f.db, generated)
	require.Error(t, err)
	require.Contains(t, err.Error(), "chk_generated_documents_completed")
}
