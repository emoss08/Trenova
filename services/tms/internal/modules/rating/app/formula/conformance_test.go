package formula_test

import (
	"os"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type conformanceCase struct {
	ID          string `json:"id"`
	Expression  string `json:"expression"`
	Accepted    bool   `json:"accepted"`
	Value       string `json:"value"`
	ServerError string `json:"serverError"`
	ErrorFrom   *int   `json:"errorFrom"`
	Note        string `json:"note"`
}

type conformanceCorpus struct {
	Inputs []*formulatypes.VariableDefinition `json:"inputs"`
	Cases  []conformanceCase                  `json:"cases"`
}

func loadConformanceCorpus(t *testing.T) conformanceCorpus {
	t.Helper()

	raw, err := os.ReadFile("testdata/conformance.json")
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

func TestConformance_TheEngineSavesExactlyWhatTheCorpusSays(t *testing.T) {
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

			env, err := svc.BuildValidationEnvironment("shipment", inputs)
			require.NoError(t, err)

			err = svc.ValidateExpressionWithEnv(t.Context(), tc.Expression, env)
			if !tc.Accepted {
				require.Error(t, err, "the engine saves %q", tc.Expression)
				if tc.ServerError != "" {
					assert.Contains(t, err.Error(), tc.ServerError)
				}
				return
			}
			require.NoError(t, err, "the engine refuses %q", tc.Expression)

			if tc.Value == "" {
				return
			}
			resp, err := svc.EvaluateExpression(t.Context(), &formula.EvaluateExpressionRequest{
				Expression: tc.Expression,
				Entity:     conformanceShipment(),
				SchemaID:   "shipment",
				Variables:  inputs,
				TenantInfo: testTenant(),
			})
			require.NoError(t, err)
			assert.True(
				t,
				decimal.RequireFromString(tc.Value).Equal(resp.Amount),
				"%q: want %s, got %s",
				tc.Expression,
				tc.Value,
				resp.Amount,
			)
		})
	}
}

func TestConformance_ErrorPositionsAreGivenWithTheMessage(t *testing.T) {
	t.Parallel()

	for _, tc := range loadConformanceCorpus(t).Cases {
		if tc.ServerError == "" {
			continue
		}
		assert.NotNil(t, tc.ErrorFrom, "%s says where the engine points", tc.ID)
		assert.True(t, strings.Contains(tc.ServerError, "("), "%s carries a position", tc.ID)
	}
}
