package formula_test

import (
	"os"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const conformanceCorpusPath = "../../../../../../crates/rules/testdata/conformance.json"

type conformanceCase struct {
	ID         string `json:"id"`
	Expression string `json:"expression"`
	Accepted   bool   `json:"accepted"`
	Value      string `json:"value"`
	Code       string `json:"code"`
	From       *int   `json:"from"`
	Note       string `json:"note"`
}

type conformanceCorpus struct {
	Inputs []*formulatypes.VariableDefinition `json:"inputs"`
	Cases  []conformanceCase                  `json:"cases"`
}

func loadConformanceCorpus(t *testing.T) conformanceCorpus {
	t.Helper()

	raw, err := os.ReadFile(conformanceCorpusPath)
	require.NoError(t, err)

	var corpus conformanceCorpus
	require.NoError(t, sonic.Unmarshal(raw, &corpus))
	require.NotEmpty(t, corpus.Cases)
	return corpus
}

func conformanceShipment() *shipment.Shipment {
	weight := int64(1200)
	entity := ratedShipment()
	entity.Weight = &weight
	entity.Moves = []*shipment.ShipmentMove{{
		Stops: []*shipment.Stop{
			stopAt(shipment.StopTypePickup, shipment.StopStatusNew, 0, "Dallas", 1_800_000_000),
			stopAt(shipment.StopTypeDelivery, shipment.StopStatusNew, 1, "Austin", 1_800_020_000),
		},
	}}
	return entity
}

func TestConformance_EveryCorpusCaseThroughTheGoEnvironmentAndTheModule(t *testing.T) {
	t.Parallel()

	corpus := loadConformanceCorpus(t)
	svc := setupService(t)

	inputs := make(map[string]any, len(corpus.Inputs))
	for _, input := range corpus.Inputs {
		inputs[input.Name] = input.DefaultValue
	}

	seen := make(map[string]struct{}, len(corpus.Cases))
	for _, tc := range corpus.Cases {
		_, duplicate := seen[tc.ID]
		require.False(t, duplicate, "case %s is listed twice", tc.ID)
		seen[tc.ID] = struct{}{}

		t.Run(tc.ID, func(t *testing.T) {
			t.Parallel()

			checked, err := svc.Check(t.Context(), &formula.CheckRequest{
				SchemaID: "shipment",
				Inputs:   formula.TemplateInputs(corpus.Inputs),
				Expect:   rules.ExpectAmount,
				Sources:  []formula.CheckSource{{Scope: formula.MainScope, Source: tc.Expression}},
			})
			require.NoError(t, err)
			check := checked[0].Check

			if !tc.Accepted {
				require.NotEmpty(t, tc.Code, "a refused case names its code")
				require.NotNil(t, tc.From, "a refused case names where")
				found := false
				for _, diagnostic := range check.Diagnostics {
					if diagnostic.IsError() && diagnostic.Code == tc.Code &&
						diagnostic.From == *tc.From {
						found = true
					}
				}
				assert.True(t, found, "%q: want %s at %d, got %+v",
					tc.Expression, tc.Code, *tc.From, check.Diagnostics)
				return
			}
			require.True(t, check.Accepted(), "%q is refused: %+v", tc.Expression, check.Diagnostics)

			if tc.Value == "" {
				return
			}
			preview, err := svc.Preview(t.Context(), &formula.PreviewRequest{
				SchemaID:   "shipment",
				Expression: tc.Expression,
				Variables:  inputs,
				Entity:     conformanceShipment(),
				TenantInfo: testTenant(),
			})
			require.NoError(t, err)
			require.True(t, preview.Valid(), "%q: %+v %v", tc.Expression, preview.Diagnostics, preview.Error)
			assert.True(
				t,
				decimal.RequireFromString(tc.Value).Equal(preview.Evaluated.Amount),
				"%q: want %s, got %s",
				tc.Expression,
				tc.Value,
				preview.Evaluated.Amount,
			)
		})
	}
}
