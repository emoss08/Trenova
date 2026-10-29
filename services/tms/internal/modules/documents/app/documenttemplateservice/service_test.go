package documenttemplateservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/documenttemplateservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templateregistry"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttemplateportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	service    *documenttemplateservice.Service
	repo       *documenttemplateportsmock.MockDocumentTemplateRepository
	versions   *documenttemplateportsmock.MockDocumentTemplateVersionRepository
	inliner    *documenttemplateportsmock.MockAssetInliner
	pdf        *documenttemplateportsmock.MockPDFRenderer
	generated  *documenttemplateportsmock.MockGeneratedDocumentRepository
	registry   *templatecatalog.Registry
	tenantInfo pagination.TenantInfo
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	registry := templateregistry.NewRegistry()
	repo := documenttemplateportsmock.NewMockDocumentTemplateRepository(t)
	versions := documenttemplateportsmock.NewMockDocumentTemplateVersionRepository(t)
	inliner := documenttemplateportsmock.NewMockAssetInliner(t)
	pdf := documenttemplateportsmock.NewMockPDFRenderer(t)
	generated := documenttemplateportsmock.NewMockGeneratedDocumentRepository(t)

	engine, err := templateengine.New(templateengine.Config{})
	require.NoError(t, err)

	service := documenttemplateservice.New(documenttemplateservice.Params{
		Logger:         zap.NewNop(),
		Repo:           repo,
		VersionRepo:    versions,
		AssignmentRepo: documenttemplateportsmock.NewMockDocumentTemplateAssignmentRepository(t),
		GeneratedRepo:  generated,
		Registry:       registry,
		Engine:         engine,
		Inliner:        inliner,
		PDFRenderer:    pdf,
		Validator: documenttemplateservice.NewValidator(
			documenttemplateservice.ValidatorParams{Registry: registry},
		),
		AuditService: auditportsmock.NewMockAuditService(t),
	})

	return &harness{
		service:   service,
		repo:      repo,
		versions:  versions,
		inliner:   inliner,
		pdf:       pdf,
		generated: generated,
		registry:  registry,
		tenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}
}

// Every built-in must satisfy the gate an organization's own template has to
// pass. A starter that could not be published would mean the shipped default is
// something no administrator is allowed to author — and it is the content every
// organization renders until they customize it.
func TestBuiltInStartersPassTheirOwnPublishGate(t *testing.T) {
	t.Parallel()

	registry := templateregistry.NewRegistry()

	for _, kind := range documenttemplate.AllKinds() {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)

			starter, err := starters.For(kind)
			require.NoError(t, err)

			def, ok := registry.Get(kind)
			require.True(t, ok)

			version := &documenttemplate.DocumentTemplateVersion{
				ID:             pulid.MustNew("dtv_"),
				OrganizationID: h.tenantInfo.OrgID,
				BusinessUnitID: h.tenantInfo.BuID,
				TemplateID:     pulid.MustNew("dtpl_"),
				VersionNumber:  1,
				Status:         documenttemplate.VersionStatusDraft,
				Subject:        starter.Subject,
				BodyHTML:       starter.BodyHTML,
				BodyText:       starter.BodyText,
				CSSContent:     starter.CSS,
			}

			multiErr := h.service.VerifyForPublish(t.Context(), def, version)
			if multiErr != nil {
				t.Fatalf("built-in %s cannot be published: %s", kind, multiErr.Error())
			}
		})
	}
}

func TestResolveFallsBackToTheBuiltIn(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	// No row means the organization has customized nothing — the built-in is a
	// tier, not an error.
	h.repo.EXPECT().
		Resolve(mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

	resolved, err := h.service.Resolve(t.Context(), &documenttemplateports.ResolveTemplateRequest{
		TenantInfo: h.tenantInfo,
		Kind:       documenttemplate.KindInvoicePDF,
	})
	require.NoError(t, err)

	require.Equal(t, documenttemplate.SourceBuiltIn, resolved.Source)
	require.True(t, resolved.FromBuiltIn())
	require.NotEmpty(t, resolved.Content.BodyHTML)
	require.NotEmpty(t, resolved.ContentHash)
	// A built-in has no row, so nothing may claim it does.
	require.Nil(t, resolved.TemplateID)
	require.Nil(t, resolved.VersionID)
}

// A customer means nothing to a kind that is not sent per customer. Passing one
// through would make the assignment tier look live for a scheduled report.
func TestResolveDropsTheCustomerForAKindThatIsNotCustomerScoped(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID := pulid.MustNew("cus_")

	def, ok := h.registry.Get(documenttemplate.KindReportPDF)
	require.True(t, ok)
	require.False(t, def.CustomerScoped, "this test is meaningless if the kind is scoped")

	h.repo.EXPECT().
		Resolve(mock.Anything, mock.MatchedBy(
			func(req *documenttemplateports.ResolveDocumentTemplateRequest) bool {
				return req.CustomerID == nil
			},
		)).
		Return(nil, nil).
		Once()

	_, err := h.service.Resolve(t.Context(), &documenttemplateports.ResolveTemplateRequest{
		TenantInfo: h.tenantInfo,
		Kind:       documenttemplate.KindReportPDF,
		CustomerID: &customerID,
	})
	require.NoError(t, err)
}

func TestResolveKeepsTheCustomerForACustomerScopedKind(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID := pulid.MustNew("cus_")

	def, ok := h.registry.Get(documenttemplate.KindInvoicePDF)
	require.True(t, ok)
	require.True(t, def.CustomerScoped)

	h.repo.EXPECT().
		Resolve(mock.Anything, mock.MatchedBy(
			func(req *documenttemplateports.ResolveDocumentTemplateRequest) bool {
				return req.CustomerID != nil && *req.CustomerID == customerID
			},
		)).
		Return(nil, nil).
		Once()

	_, err := h.service.Resolve(t.Context(), &documenttemplateports.ResolveTemplateRequest{
		TenantInfo: h.tenantInfo,
		Kind:       documenttemplate.KindInvoicePDF,
		CustomerID: &customerID,
	})
	require.NoError(t, err)
}

func TestResolveRejectsAnUnregisteredKind(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.service.Resolve(t.Context(), &documenttemplateports.ResolveTemplateRequest{
		TenantInfo: h.tenantInfo,
		Kind:       documenttemplate.Kind("invoice.telegram"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not registered")
}

func TestPublishGateRefusesAnythingButADraft(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	def, ok := h.registry.Get(documenttemplate.KindInvoicePDF)
	require.True(t, ok)

	starter, err := starters.For(documenttemplate.KindInvoicePDF)
	require.NoError(t, err)

	multiErr := h.service.VerifyForPublish(
		t.Context(),
		def,
		&documenttemplate.DocumentTemplateVersion{
			Status:   documenttemplate.VersionStatusActive,
			BodyHTML: starter.BodyHTML,
		},
	)
	require.NotNil(t, multiErr)
	require.Contains(t, multiErr.Error(), "Only a draft can be published")
}

// This is the check that stops an organization publishing an invoice that no
// longer prints something the business requires.
func TestPublishGateRefusesAVersionMissingARequiredPath(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	def, ok := h.registry.Get(documenttemplate.KindInvoicePDF)
	require.True(t, ok)

	required := h.registry.RequiredPaths(documenttemplate.KindInvoicePDF)
	require.NotEmpty(t, required, "the kind declares nothing required, so this proves nothing")

	multiErr := h.service.VerifyForPublish(
		t.Context(),
		def,
		&documenttemplate.DocumentTemplateVersion{
			Status:   documenttemplate.VersionStatusDraft,
			BodyHTML: `<p>Thanks for your business.</p>`,
		},
	)
	require.NotNil(t, multiErr)
}

// A template that references something the catalog does not publish would render
// blank in production, and the author would never find out from their own
// preview.
func TestPublishGateRefusesAnUnknownVariable(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	def, ok := h.registry.Get(documenttemplate.KindInvoicePDF)
	require.True(t, ok)

	starter, err := starters.For(documenttemplate.KindInvoicePDF)
	require.NoError(t, err)

	multiErr := h.service.VerifyForPublish(
		t.Context(),
		def,
		&documenttemplate.DocumentTemplateVersion{
			Status:   documenttemplate.VersionStatusDraft,
			BodyHTML: starter.BodyHTML + `<p>{{ .CustomerSecretDiscountRate }}</p>`,
		},
	)
	require.NotNil(t, multiErr)
}

func TestPublishGateRefusesAnUnregisteredKindsVersion(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	h.versions.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&documenttemplate.DocumentTemplateVersion{
			ID:         pulid.MustNew("dtv_"),
			TemplateID: pulid.MustNew("dtpl_"),
			Status:     documenttemplate.VersionStatusDraft,
			Template: &documenttemplate.DocumentTemplate{
				Kind: documenttemplate.Kind("invoice.carrier_pigeon"),
			},
		}, nil).
		Once()

	_, err := h.service.PublishVersion(t.Context(), &documenttemplateports.PublishVersionRequest{
		TenantInfo: h.tenantInfo,
		VersionID:  pulid.MustNew("dtv_"),
		UserID:     pulid.MustNew("usr_"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not registered")
}

func TestContentHashDistinguishesWhereContentLives(t *testing.T) {
	t.Parallel()

	// Moving text between channels must change the identity, or a version that
	// renders differently would be recorded as the one that produced older
	// documents.
	subjectSide := &documenttemplateports.TemplateContent{Subject: "Invoice", BodyHTML: ""}
	bodySide := &documenttemplateports.TemplateContent{Subject: "", BodyHTML: "Invoice"}

	require.NotEqual(
		t,
		documenttemplateservice.ContentHash(subjectSide),
		documenttemplateservice.ContentHash(bodySide),
	)
}

func publishableInvoice(
	t *testing.T,
) (*templatecatalog.KindDefinition, *documenttemplate.DocumentTemplateVersion) {
	t.Helper()

	registry := templateregistry.NewRegistry()
	def, ok := registry.Get(documenttemplate.KindInvoicePDF)
	require.True(t, ok)
	starter, err := starters.For(documenttemplate.KindInvoicePDF)
	require.NoError(t, err)

	return def, &documenttemplate.DocumentTemplateVersion{
		ID:             pulid.MustNew("dtv_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		TemplateID:     pulid.MustNew("dtpl_"),
		VersionNumber:  1,
		Status:         documenttemplate.VersionStatusDraft,
		BodyHTML:       starter.BodyHTML,
		CSSContent:     starter.CSS,
	}
}

func TestPublishGateRefusesAStylesheetThatBreaksOutOfItsStyleElement(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	def, version := publishableInvoice(t)
	require.Nil(t, h.service.VerifyForPublish(t.Context(), def, version))

	version.CSSContent += "\n</style><script>alert(1)</script>"
	multiErr := h.service.VerifyForPublish(t.Context(), def, version)
	require.NotNil(t, multiErr)
	assert.Contains(t, multiErr.Error(), "cssContent")
}

func TestPublishGateRefusesAHostileRunningHeader(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	def, version := publishableInvoice(t)
	version.HeaderHTML = `<div onload="alert(1)">Invoice</div>`

	multiErr := h.service.VerifyForPublish(t.Context(), def, version)
	require.NotNil(t, multiErr)
	assert.Contains(t, multiErr.Error(), "headerHtml")
}

func TestSavingADraftRefusesAStylesheetThatLoadsRemoteContent(t *testing.T) {
	t.Parallel()

	registry := templateregistry.NewRegistry()
	validator := documenttemplateservice.NewValidator(
		documenttemplateservice.ValidatorParams{Registry: registry},
	)
	_, version := publishableInvoice(t)
	version.CSSContent = `@\69mport "https://evil.example/x.css";`

	multiErr := validator.ValidateVersion(t.Context(), version, documenttemplate.KindInvoicePDF)
	require.NotNil(t, multiErr)
	assert.Contains(t, multiErr.Error(), "cssContent")
}
