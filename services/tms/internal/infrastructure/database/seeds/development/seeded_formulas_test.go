package development

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeededAccessorialConditions_PassTheModulesCheckAsPredicates(t *testing.T) {
	t.Parallel()

	svc := ratingtest.NewFormulaService(t)
	checked := 0
	for _, def := range seededAccessorials() {
		if def.condition == "" {
			continue
		}
		results, err := svc.Check(t.Context(), &formula.CheckRequest{
			SchemaID: "shipment",
			Expect:   rules.ExpectPredicate,
			Sources:  []formula.CheckSource{{Scope: def.code, Source: def.condition}},
		})
		require.NoError(t, err)
		assert.Nil(t, results[0].Check.FirstError(), "%s: %q", def.code, def.condition)
		checked++
	}
	assert.Positive(t, checked)
}
