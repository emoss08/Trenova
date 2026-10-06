package documenttemplateservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documenttemplateservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	service    *documenttemplateservice.Service
	repo       *mocks.MockDocumentTemplateRepository
	versions   *mocks.MockDocumentTemplateVersionRepository
	inliner    *mocks.MockAssetInliner
	pdf        *mocks.MockPDFRenderer
	generated  *mocks.MockGeneratedDocumentRepository
	registry   *documenttemplate.Registry
	tenantInfo pagination.TenantInfo
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	registry := documenttemplate.NewRegistry()
	repo := mocks.NewMockDocumentTemplateRepository(t)
	versions := mocks.NewMockDocumentTemplateVersionRepository(t)
	inliner := mocks.NewMockAssetInliner(t)
	pdf := mocks.NewMockPDFRenderer(t)
	generated := mocks.NewMockGeneratedDocumentRepository(t)

	engine, err := templateengine.New(templateengine.Config{})
	require.NoError(t, err)

	service := documenttemplateservice.New(documenttemplateservice.Params{
		Logger:         zap.NewNop(),
		Repo:           repo,
		VersionRepo:    versions,
		AssignmentRepo: mocks.NewMockDocumentTemplateAssignmentRepository(t),
		GeneratedRepo:  generated,
		Registry:       registry,
		Engine:         engine,
		Inliner:        inliner,
		PDFRenderer:    pdf,
		Validator: documenttemplateservice.NewValidator(
			documenttemplateservice.ValidatorParams{Registry: registry},
		),
		AuditService: mocks.NewMockAuditService(t),
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

	registry := documenttemplate.NewRegistry()

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

	resolved, err := h.service.Resolve(t.Context(), &services.ResolveTemplateRequest{
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
			func(req *repositories.ResolveDocumentTemplateRequest) bool {
				return req.CustomerID == nil
			},
		)).
		Return(nil, nil).
		Once()

	_, err := h.service.Resolve(t.Context(), &services.ResolveTemplateRequest{
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
			func(req *repositories.ResolveDocumentTemplateRequest) bool {
				return req.CustomerID != nil && *req.CustomerID == customerID
			},
		)).
		Return(nil, nil).
		Once()

	_, err := h.service.Resolve(t.Context(), &services.ResolveTemplateRequest{
		TenantInfo: h.tenantInfo,
		Kind:       documenttemplate.KindInvoicePDF,
		CustomerID: &customerID,
	})
	require.NoError(t, err)
}

func TestResolveRejectsAnUnregisteredKind(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.service.Resolve(t.Context(), &services.ResolveTemplateRequest{
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

	_, err := h.service.PublishVersion(t.Context(), &services.PublishVersionRequest{
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
	subjectSide := &services.TemplateContent{Subject: "Invoice", BodyHTML: ""}
	bodySide := &services.TemplateContent{Subject: "", BodyHTML: "Invoice"}

	require.NotEqual(
		t,
		documenttemplateservice.ContentHash(subjectSide),
		documenttemplateservice.ContentHash(bodySide),
	)
}
