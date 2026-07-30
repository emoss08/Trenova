package invoiceservice

import (
	"context"
	"html/template"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

// stubInliner stands in for the asset inliner. The real one is covered by its
// own tests; what matters here is what the builder does with the answer.
type stubInliner struct {
	dataURI   string
	err       error
	lastSrc   string
	callCount int
}

func (s *stubInliner) InlineAssets(
	_ context.Context,
	req *services.AssetInlineRequest,
) (*services.AssetInlineResult, error) {
	return &services.AssetInlineResult{HTML: req.HTML}, nil
}

func (s *stubInliner) ResolveImageDataURI(_ context.Context, src string) (string, error) {
	s.callCount++
	s.lastSrc = src
	return s.dataURI, s.err
}

var _ services.AssetInliner = (*stubInliner)(nil)

// The organization logo is the one field an organization controls that lands in
// a template.URL slot, where html/template will not escape it. It must therefore
// only ever arrive as base64 the inliner produced.
func TestContextBuilderResolvesTheLogoThroughTheInliner(t *testing.T) {
	t.Parallel()

	inliner := &stubInliner{dataURI: "data:image/png;base64,AAAA"}
	builder := &ContextBuilder{inliner: inliner}

	ctx, err := builder.BuildFrom(
		t.Context(),
		&invoice.Invoice{CurrencyCode: "USD"},
		&invoiceDeliveryProfile{
			Organization: &tenant.Organization{Name: "Trenova", LogoURL: "logos/org.png"},
		},
	)
	require.NoError(t, err)

	require.Equal(t, 1, inliner.callCount)
	require.Equal(t, "logos/org.png", inliner.lastSrc)
	require.Equal(t, template.URL("data:image/png;base64,AAAA"), ctx.LogoDataURI)
}

// A logo that cannot be fetched must not stop an invoice going out. The template
// drops the image; the customer still gets billed.
func TestContextBuilderLeavesTheLogoEmptyWhenItCannotBeResolved(t *testing.T) {
	t.Parallel()

	builder := &ContextBuilder{inliner: &stubInliner{dataURI: ""}}

	ctx, err := builder.BuildFrom(
		t.Context(),
		&invoice.Invoice{CurrencyCode: "USD"},
		&invoiceDeliveryProfile{
			Organization: &tenant.Organization{Name: "Trenova", LogoURL: "logos/missing.png"},
		},
	)
	require.NoError(t, err)

	require.Equal(t, template.URL(""), ctx.LogoDataURI)
	require.Equal(t, "Trenova", ctx.Organization.Name)
}

func TestContextBuilderSkipsTheInlinerWhenNoLogoIsConfigured(t *testing.T) {
	t.Parallel()

	inliner := &stubInliner{dataURI: "data:image/png;base64,AAAA"}
	builder := &ContextBuilder{inliner: inliner}

	_, err := builder.BuildFrom(
		t.Context(),
		&invoice.Invoice{CurrencyCode: "USD"},
		&invoiceDeliveryProfile{Organization: &tenant.Organization{Name: "Trenova"}},
	)
	require.NoError(t, err)

	require.Zero(t, inliner.callCount, "an organization with no logo must not cause a fetch")
}

// BillingControl continuity. These four settings were honoured by the deleted
// renderer, and an organization that never touches a template must see no
// change at all — so the values have to survive the whole way from the control
// row into the published context.
func TestBillingControlSettingsSurviveIntoTheTemplateContext(t *testing.T) {
	t.Parallel()

	builder := &ContextBuilder{inliner: &stubInliner{}}

	entity := &invoice.Invoice{
		CurrencyCode: "USD",
		Number:       "INV-1001",
		DueDate:      testDueDate(),
	}

	ctx, err := builder.BuildFrom(t.Context(), entity, &invoiceDeliveryProfile{
		Organization: &tenant.Organization{Name: "Trenova"},
		BillingControl: &tenant.BillingControl{
			DefaultInvoiceTerms:     "Net 30 from invoice date.",
			DefaultInvoiceFooter:    "Thank you for your business.",
			ShowDueDateOnInvoice:    true,
			ShowBalanceDueOnInvoice: true,
		},
	})
	require.NoError(t, err)

	require.Equal(t, []string{"Net 30 from invoice date."}, ctx.InvoiceTerms)
	require.Equal(t, "Thank you for your business.", ctx.InvoiceFooter)
	require.NotEmpty(t, ctx.DueDate, "ShowDueDateOnInvoice must still produce a due date")
	require.NotEmpty(t, ctx.BalanceDue, "ShowBalanceDueOnInvoice must still produce a balance")
}

// The same settings turned off must still suppress the fields, exactly as the
// old invoicePDFShowDueDate / ShowBalanceDue helpers did.
func TestBillingControlSuppressionSurvivesIntoTheTemplateContext(t *testing.T) {
	t.Parallel()

	builder := &ContextBuilder{inliner: &stubInliner{}}

	ctx, err := builder.BuildFrom(
		t.Context(),
		&invoice.Invoice{CurrencyCode: "USD", DueDate: testDueDate()},
		&invoiceDeliveryProfile{
			Organization: &tenant.Organization{Name: "Trenova"},
			BillingControl: &tenant.BillingControl{
				ShowDueDateOnInvoice:    false,
				ShowBalanceDueOnInvoice: false,
			},
		},
	)
	require.NoError(t, err)

	require.Empty(t, ctx.DueDate)
	require.Empty(t, ctx.BalanceDue)
	require.Empty(t, ctx.InvoiceTerms)
	require.Empty(t, ctx.InvoiceFooter)
}

// The built-in invoice template is what every organization renders until it
// customizes one. This proves the real context the builder produces satisfies
// the same gate a hand-authored template must pass — catalog coverage, required
// paths, and all.
func TestTheBuiltInInvoiceTemplateRendersTheBuilderOutput(t *testing.T) {
	t.Parallel()

	builder := &ContextBuilder{inliner: &stubInliner{}}

	data, err := builder.BuildFrom(
		t.Context(),
		testInvoiceForPDF(
			testPULID(t, "inv_"), testPULID(t, "org_"), testPULID(t, "bu_"),
		),
		testInvoicePDFDeliveryProfile(),
	)
	require.NoError(t, err)

	engine, err := templateengine.New(templateengine.Config{})
	require.NoError(t, err)

	registry := documenttemplate.NewRegistry()
	starter := builtInInvoiceStarter(t)

	compiled, diags := engine.Parse(&templateengine.ParseRequest{
		Field:        documenttemplate.ChannelPDF.Field(),
		Kind:         string(documenttemplate.KindInvoicePDF),
		Channel:      documenttemplate.ChannelPDF.Engine(),
		Source:       starter,
		AllowedPaths: registry.Paths(documenttemplate.KindInvoicePDF),
		Strict:       true,
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	output, err := engine.Render(t.Context(), compiled, data)
	require.NoError(t, err)

	// The figures a customer is billed on must actually appear.
	require.Contains(t, output, "INV-1001")
	require.NotContains(t, output, "ZgotmplZ",
		"a context value landed somewhere it cannot be escaped")
}

func testPULID(t *testing.T, prefix string) pulid.ID {
	t.Helper()
	return pulid.MustNew(prefix)
}

func builtInInvoiceStarter(t *testing.T) string {
	t.Helper()

	starter, err := starters.For(documenttemplate.KindInvoicePDF)
	require.NoError(t, err)

	return starter.BodyHTML
}

func testDueDate() *int64 {
	due := int64(1735689600)
	return &due
}
