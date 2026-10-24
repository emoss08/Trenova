package reportengine

import (
	"testing"

	"github.com/emoss08/keel/analytics/analyticscli"
	"github.com/emoss08/keel/analytics/ir/tsgen"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryReportableEntityHasAKind(t *testing.T) {
	t.Parallel()

	catalog, err := NewCatalog()
	require.NoError(t, err)
	require.NotEmpty(t, catalog.Entities())

	_, err = reportaccess.New(reportaccess.Params{Catalog: catalog})
	require.NoError(t, err)
}

func TestCatalogReadsInTheRequestersLanguage(t *testing.T) {
	t.Parallel()

	catalog, err := NewCatalog()
	require.NoError(t, err)

	spanish := catalog.For(i18n.WithLocale(t.Context(), i18n.ES), nil)
	english := catalog.For(i18n.WithLocale(t.Context(), i18n.EN), nil)
	require.Equal(t, len(english.Entities), len(spanish.Entities))

	for i := range english.Entities {
		if english.Entities[i].Key != "customer" {
			continue
		}
		en, es := english.Entities[i], spanish.Entities[i]
		assert.Equal(t, "Customer", en.Label)
		assert.Equal(t, i18n.Translate(i18n.ES, en.Label), es.Label)
		assert.Equal(t, i18n.Translate(i18n.ES, en.Plural), es.Plural)
		assert.NotEqual(
			t,
			en.Plural,
			es.Plural,
			"the Spanish catalog has no entry for %q",
			en.Plural,
		)
		return
	}
	t.Fatal("the catalog has no customer entity")
}

func TestClientDefinitionModuleIsCurrent(t *testing.T) {
	t.Parallel()

	assert.NoError(
		t,
		tsgen.Check("../../../../../client/packages/shared/src/generated/keel-analytics.ts"),
		"run `go generate ./internal/infrastructure/reportengine/...` and commit",
	)
}

func TestMessageCatalogIsCurrent(t *testing.T) {
	t.Parallel()

	catalog, err := NewCatalog()
	require.NoError(t, err)
	assert.NoError(t, analyticscli.CheckMessages("analytics.messages.json", catalog),
		"run `go generate ./internal/infrastructure/reportengine/...` and commit")
}
