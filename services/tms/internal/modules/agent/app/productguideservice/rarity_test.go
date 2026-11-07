package productguideservice

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentsearch"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rarityPage(path, name string, aliases []string, notes string) string {
	quoted := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		quoted = append(quoted, fmt.Sprintf("%q", alias))
	}

	return fmt.Sprintf(`{
      "path": %q, "name": %q, "module": "admin", "breadcrumb": [%q],
      "description": "", "summary": "", "requires": [], "capabilities": [],
      "aliases": [%s], "tasks": [], "notes": %q, "related": [], "covers": [],
      "createAction": null, "inNavigation": true
    }`, path, name, name, strings.Join(quoted, ", "), notes)
}

func rarityCatalog(t *testing.T) *productguide.Catalog {
	t.Helper()

	catalog, err := productguide.Load(
		[]byte(`{"version": "sha256:test", "modules": [], "pages": [` +
			strings.Join([]string{
				rarityPage("/admin/roles", "Roles", []string{"access control"},
					"Needs read access to roles."),
				rarityPage("/billing/queue", "Billing queue", nil,
					"Needs read access to billing."),
				rarityPage("/admin/billing-controls", "Billing controls", nil,
					"Needs read access to billing controls."),
				rarityPage("/billing/customers", "Customer billing profiles", nil,
					"Needs read access to customers."),
			}, ", ") + `], "records": []}`),
	)
	require.NoError(t, err)

	return catalog
}

/*
Every guide's notes say "Needs read access to …", so counted over every field
"access" was on every page and worth nothing, and "where do I give someone
access to billing" ranked three billing pages over Roles. A word's rarity is
counted over the fields that say what a page is for.
*/
func TestRank_CountsAWordsRarityWhereAPageSaysWhatItIsFor(t *testing.T) {
	t.Parallel()

	service := NewOffline(rarityCatalog(t))

	ranked := service.rank(agentsearch.TokenSet("where do I give someone access to billing"), "")

	require.NotEmpty(t, ranked)
	assert.Equal(t, "/admin/roles", ranked[0].entry.page.Path)
}

func TestRarity_WeighsAWordOnFewPagesAboveOneOnMany(t *testing.T) {
	t.Parallel()

	service := NewOffline(rarityCatalog(t))

	assert.Greater(t, service.rarity["access"], service.rarity["billing"],
		"one page describes itself by access, three by billing")
	assert.Greater(t, service.rarity["billing"], 0.0)
}
