package pagemanifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/emoss08/trenova/internal/api/pagemanifest"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noKinds(string) bool { return false }

func TestViewWords_ReadAViewAsAPersonDoes(t *testing.T) {
	t.Parallel()

	isKind := func(field string) bool { return field == "invoice" }
	assert.Equal(t, []pagemanifest.ViewWord{
		{Label: "code", Fields: []string{"code"}},
		{Label: "address", Fields: []string{"addressLine1", "city", "stateId", "state", "postalCode"}},
		{Label: "MC number", Fields: []string{"mcNumber"}},
		{Label: "fleet code", Fields: []string{"fleetCodeId", "fleetCode"}},
		{Label: "invoice", Fields: []string{"invoiceId"}},
		{Label: "provider message ID", Fields: []string{"providerMessageId"}},
		{Label: "system", Fields: []string{"isSystem"}},
	}, pagemanifest.ViewWords([]string{
		"id", "code", "addressLine1", "city", "stateId", "state", "postalCode", "mcNumber",
		"fleetCodeId", "fleetCode", "createdAt", "invoiceId", "providerMessageId", "isSystem",
	}, isKind))
}

func TestFieldWords_SpellsInitialismsAndNumbers(t *testing.T) {
	t.Parallel()

	for field, want := range map[string]string{
		"proNumber":     "PRO number",
		"scac":          "SCAC",
		"profilePicUrl": "profile pic URL",
		"addressLine2":  "address line 2",
		"ediPartner":    "EDI partner",
		"externalId":    "external ID",
	} {
		assert.Equal(t, want, pagemanifest.FieldWords(field, noKinds), field)
	}
}

// The role editor's words are what the generator writes from the registry
// and the schema, so a changed view or a new link between kinds fails until
// it is regenerated and its words are translated.
func TestViewWordsAreGenerated(t *testing.T) {
	t.Parallel()

	schema, err := schemadiff.LoadDir(filepath.Join(repoRoot, "services/tms/internal/api/graphql/schema"))
	require.NoError(t, err)
	rendered, err := pagemanifest.RenderWords(schema, fieldsensitivity.DefaultRegistry())
	require.NoError(t, err)

	current, err := os.ReadFile(filepath.Join(repoRoot, pagemanifest.WordsPath))
	require.NoError(t, err)
	require.Equal(t, rendered, string(current),
		"viewwords_gen.go is stale. Run `go run ./internal/api/graphql/gqlext/refgen` from "+
			"services/tms, then `task i18n` and translate the new words")
}
