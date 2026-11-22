package emaildesign

import (
	"html/template"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func render(t *testing.T, source string, data any) string {
	t.Helper()

	tmpl, err := template.New("t").Funcs(Funcs(i18n.Default)).Parse(source)
	require.NoError(t, err)

	var b strings.Builder
	require.NoError(t, tmpl.Execute(&b, data))
	return b.String()
}

func TestBlocksEscapeWhatTheyAreGiven(t *testing.T) {
	t.Parallel()

	out := render(t, `{{ para .V }}{{ kicker .V }}{{ quote .V }}{{ notice .V }}{{ code .V }}`,
		map[string]string{"V": `<script>alert(1)</script>`})

	assert.NotContains(t, out, "<script>")
	assert.Equal(t, 5, strings.Count(out, "&lt;script&gt;"))
}

func TestDataCannotForgeAShellMarker(t *testing.T) {
	t.Parallel()

	out := render(t, `{{ para .V }}`, map[string]string{"V": preheaderMarker.wrap("forged")})

	assert.Zero(t, preheaderMarker.count(out), "escaped data never reads as a marker")
}

func TestLinksAcceptOnlyWebAndMailAddresses(t *testing.T) {
	t.Parallel()

	assert.Empty(t, string(action("Go", "javascript:alert(1)")))
	assert.Empty(t, string(link("data:text/html,x", "Go")))
	assert.Empty(t, string(action("Go", "")))
	assert.Contains(t, string(action("Go", "https://example.com/a?b=1&c=2")),
		`href="https://example.com/a?b=1&amp;c=2"`)
	assert.Contains(
		t,
		string(link("mailto:ap@example.com", "Write to us")),
		"mailto:ap@example.com",
	)
}

func TestHeadingHighlightsOnePhrase(t *testing.T) {
	t.Parallel()

	out := render(t, `{{ heading "Invoice {0} <hl>is attached</hl>." .N }}`,
		map[string]string{"N": "<b>INV-1</b>"})

	assert.Equal(t, 1, strings.Count(out, "<h1"))
	assert.Equal(t, 1, strings.Count(out, "background-color:"+amber))
	assert.Contains(t, out, "Invoice &lt;b&gt;INV-1&lt;/b&gt; <span")
	assert.Contains(t, out, ">is attached.</span></h1>")

	plain := plainHeadingFor(i18n.Default)("Invoice {0} <hl>is attached</hl>.", "INV-1")
	assert.Equal(t, "Invoice INV-1 is attached.", plain)
}

func TestFactsSkipEmptyValuesAndCloseWithAnInkRule(t *testing.T) {
	t.Parallel()

	out := string(facts(fact("Invoice", "INV-1"), fact("Due", ""), fact("Total", "$10.00")))

	assert.Equal(t, 2, strings.Count(out, "<tr>"))
	assert.NotContains(t, out, "Due")
	last := out[strings.LastIndex(out, "<tr>"):]
	assert.Contains(t, last, "border-bottom:1px solid "+ink)
	assert.Empty(t, string(facts(fact("Due", ""))), "no rows means no table")

	collected := string(
		facts([]struct{ Label, Value string }{{"Bank", "Ozark Trust"}, {"Note", ""}}),
	)
	assert.Equal(t, 1, strings.Count(collected, "<tr>"))
}

func TestNumberedRowsStartWithoutAHairline(t *testing.T) {
	t.Parallel()

	first := string(row(step(0), cell("Bill of lading")))
	second := string(row(step(1), cell("Proof of delivery")))

	assert.Contains(t, first, ">01<")
	assert.NotContains(t, first, "border-top", "the table's ink rule opens the list")
	assert.Contains(t, second, "border-top:1px solid "+hairline)
}

func TestTableRowsReadCollections(t *testing.T) {
	t.Parallel()

	cells := []struct {
		Text    string
		Numeric bool
	}{{"Halstead", false}, {"$98.00", true}}
	out := string(row(cells))

	assert.Equal(t, 2, strings.Count(out, "<td"))
	assert.Contains(t, out, `align="right"`)
	head := string(tableOpen([]struct {
		Label   string
		Numeric bool
	}{{"Customer", false}, {"Revenue", true}}))
	assert.Contains(t, head, `<th scope="col" align="right"`)
}

func TestComposeWrapsContentInTheShell(t *testing.T) {
	t.Parallel()

	content := render(t,
		`{{ preheader "What it is." }}{{ kicker "Area" }}{{ heading "A <hl>thing</hl>." }}`+
			`{{ para "Body." }}{{ action "Open" "https://example.com" }}{{ footer "Why you got it." }}`,
		nil)

	out, diags := Compose(&Message{
		Locale:  i18n.ES,
		Subject: "Subject & more",
		Brand:   Brand{Name: "Larkspur <Freight>"},
		Content: content,
	})
	require.False(t, diags.HasErrors(), diags.Summary())
	assert.Empty(t, diags.Warnings())

	assert.True(t, strings.HasPrefix(out, "<!DOCTYPE html PUBLIC"))
	assert.Contains(t, out, `lang="es"`)
	assert.Contains(t, out, "<title>Subject &amp; more</title>")
	assert.Contains(t, out, "Larkspur &lt;Freight&gt;")
	assert.Less(t, strings.Index(out, "What it is."), strings.Index(out, "Larkspur"))
	assert.Greater(t, strings.Index(out, "Why you got it."), strings.Index(out, "Open</a>"))
	assert.NotContains(t, out, "<!--email:")

	bandCell := out[strings.Index(out, `bgcolor="`+amber+`"`):]
	bandCell = bandCell[:strings.Index(bandCell, ">")]
	assert.NotContains(t, bandCell, "border-bottom", "a band that ends the card has no lower rule")
}

func TestComposeKeepsTheLowerRuleWhenTextFollowsTheBand(t *testing.T) {
	t.Parallel()

	content := render(t, `{{ preheader "P." }}{{ heading "<hl>H</hl>" }}`+
		`{{ action "Open" "https://example.com" }}{{ para "After." }}{{ footer "F." }}`, nil)

	out, diags := Compose(&Message{Subject: "S", Content: content})
	require.False(t, diags.HasErrors())

	bandCell := out[strings.Index(out, `bgcolor="`+amber+`"`):]
	bandCell = bandCell[:strings.Index(bandCell, ">")]
	assert.Contains(t, bandCell, "border-bottom:1px solid "+ink)
}

func TestCheckNamesWhatAnEmailIsMissing(t *testing.T) {
	t.Parallel()

	diags := Check(render(t, `{{ para "Only a paragraph." }}`, nil))
	summary := diags.Summary()

	assert.Contains(t, summary, "preheader")
	assert.Contains(t, summary, "footer")
	assert.Contains(t, summary, "heading")

	twice := render(
		t,
		`{{ preheader "P." }}{{ heading "<hl>H</hl>" }}{{ action "A" "https://a.example" }}`+
			`{{ action "B" "https://b.example" }}{{ footer "F." }}`,
		nil,
	)
	assert.Contains(t, Check(twice).Summary(), "more than one action band")
}

func TestComposeRefusesUnsafeContent(t *testing.T) {
	t.Parallel()

	_, diags := Compose(&Message{Subject: "S", Content: `<script>alert(1)</script>`})

	assert.True(t, diags.HasErrors())
}

func TestTheMarkIsSizedFromTheImage(t *testing.T) {
	t.Parallel()

	assert.Empty(t, markImage("https://example.com/logo.png"), "only an embedded image is used")
	assert.Empty(t, markImage(""))
}
