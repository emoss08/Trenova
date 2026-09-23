package productguide

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixture = `{
  "version": "sha256:test",
  "modules": [{"id": "billing", "label": "Billing", "description": "Invoices and rates"}],
  "pages": [
    {
      "path": "/billing/invoices",
      "name": "Invoices",
      "module": "billing",
      "breadcrumb": ["Billing", "Invoices"],
      "description": "Every invoice.",
      "summary": "Where invoices are reviewed and posted.",
      "requires": [{"resource": "invoice", "operation": "read"}],
      "capabilities": [],
      "aliases": ["bills"],
      "tasks": [{"title": "Post an invoice", "keywords": [], "steps": ["Open it."]}],
      "notes": "",
      "related": [],
      "covers": ["/billing/invoices/:id/print"],
      "createAction": null,
      "inNavigation": true
    },
    {
      "path": "/billing/configuration-files/rate-matrices",
      "name": "Rate matrices",
      "module": "billing",
      "breadcrumb": ["Billing", "Configuration files", "Rate matrices"],
      "description": "Lane rates.",
      "summary": "Rates by lane.",
      "requires": [],
      "capabilities": [],
      "aliases": [],
      "tasks": [],
      "notes": "",
      "related": [],
      "covers": [],
      "createAction": {"label": "Create rate matrix", "query": {"panelType": "create"}},
      "inNavigation": true
    },
    {
      "path": "/",
      "name": "Home",
      "module": "home",
      "breadcrumb": ["Home"],
      "description": "",
      "summary": "Your day.",
      "requires": [],
      "capabilities": [],
      "aliases": [],
      "tasks": [],
      "notes": "",
      "related": [],
      "covers": [],
      "createAction": null,
      "inNavigation": false
    }
  ],
  "records": [
    {"entity": "invoice", "label": "Invoice", "path": "/billing/invoices", "params": {"item": "{id}"}},
    {"entity": "report", "label": "Report", "path": "/reports/explore/{id}", "params": {}}
  ]
}`

func load(t *testing.T) *Catalog {
	t.Helper()

	catalog, err := Load([]byte(fixture))
	require.NoError(t, err)

	return catalog
}

func TestPageForPath(t *testing.T) {
	t.Parallel()

	catalog := load(t)

	tests := []struct {
		name     string
		location string
		want     string
	}{
		{name: "the page itself", location: "/billing/invoices", want: "/billing/invoices"},
		{name: "with a record open", location: "/billing/invoices?item=inv_1", want: "/billing/invoices"},
		{name: "a trailing slash", location: "/billing/invoices/", want: "/billing/invoices"},
		{name: "a route the guide covers", location: "/billing/invoices/inv_1/print", want: "/billing/invoices"},
		{name: "a page nested under another", location: "/billing/invoices/unknown", want: "/billing/invoices"},
		{name: "home", location: "/", want: "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			page, ok := catalog.PageForPath(tt.location)
			require.True(t, ok)
			assert.Equal(t, tt.want, page.Path)
		})
	}

	_, ok := catalog.PageForPath("/nowhere")
	assert.False(t, ok, "home is not a prefix of every path")
}

func TestRecordPath(t *testing.T) {
	t.Parallel()

	catalog := load(t)

	path, ok := catalog.RecordPath("invoice", "inv_1", map[string]string{"tab": "charges"})
	require.True(t, ok)
	parsed, err := url.Parse(path)
	require.NoError(t, err)
	assert.Equal(t, "/billing/invoices", parsed.Path)
	assert.Equal(t, "inv_1", parsed.Query().Get("item"))
	assert.Equal(t, "charges", parsed.Query().Get("tab"))

	path, ok = catalog.RecordPath("report", "a/b", nil)
	require.True(t, ok)
	assert.Equal(t, "/reports/explore/a%2Fb", path, "an id in a path segment is escaped")

	_, ok = catalog.RecordPath("invoice", " ", nil)
	assert.False(t, ok, "no id, no record link")
	_, ok = catalog.RecordPath("unknown", "x", nil)
	assert.False(t, ok)
}

func TestListPath(t *testing.T) {
	t.Parallel()

	catalog := load(t)

	path, ok := catalog.ListPath("invoice")
	require.True(t, ok)
	assert.Equal(t, "/billing/invoices", path)

	_, ok = catalog.ListPath("report")
	assert.False(t, ok, "a record that opens on its own page has no list page to open")
}

func TestPageForResource(t *testing.T) {
	t.Parallel()

	catalog := load(t)

	page, ok := catalog.PageForResource("invoice")
	require.True(t, ok)
	assert.Equal(t, "/billing/invoices", page.Path)

	_, ok = catalog.PageForResource("rate_matrix")
	assert.False(t, ok, "a page that requires nothing is not claimed by every resource")
}

func TestPageLocation(t *testing.T) {
	t.Parallel()

	page, ok := load(t).Page("/billing/configuration-files/rate-matrices")
	require.True(t, ok)
	assert.Equal(t, "Billing › Configuration files › Rate matrices", page.Location())
}

// The embedded catalog is what a server ships with; it has to load, and every
// record link in it has to open on a page.
func TestDefaultLoads(t *testing.T) {
	t.Parallel()

	require.NotNil(t, Default)
	for _, record := range Default.Records {
		path, ok := Default.RecordPath(record.Entity, "rec_1", nil)
		require.True(t, ok, record.Entity)
		assert.NotEmpty(t, path)
	}
}
