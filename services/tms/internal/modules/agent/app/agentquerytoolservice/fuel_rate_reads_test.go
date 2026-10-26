package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateimportports"
	"github.com/emoss08/trenova/pkg/pagination"
	pkgrateimport "github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodedRows(t *testing.T, result any) []map[string]any {
	t.Helper()

	raw, err := sonic.Marshal(result)
	require.NoError(t, err)
	var decoded struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, sonic.Unmarshal(raw, &decoded))

	return decoded.Items
}

type fakeRateImports struct {
	rateimportports.RateImportRepository

	captured *rateimportports.ListRateImportBatchesRequest
}

func (f *fakeRateImports) List(
	_ context.Context,
	req *rateimportports.ListRateImportBatchesRequest,
) (*pagination.ListResult[*rateimport.RateImportBatch], error) {
	f.captured = req

	return &pagination.ListResult[*rateimport.RateImportBatch]{
		Items: []*rateimport.RateImportBatch{{
			ID:              pulid.MustNew("rib_"),
			RateAgreementID: pulid.MustNew("rag_"),
			FileName:        "acme-2027.csv",
			Status:          rateimport.StatusParsed,
			Summary:         &pkgrateimport.Summary{Added: 2, Changed: 5},
		}},
		Total:      1,
		TotalExact: true,
	}, nil
}

func TestListRateImports_NarrowsToOneAgreement(t *testing.T) {
	t.Parallel()

	repo := &fakeRateImports{}
	tool := newListRateImportsTool(repo)
	agreementID := pulid.MustNew("rag_")

	_, err := tool.Query(t.Context(), testParams(map[string]any{"rateAgreementId": "acme"}))
	require.Error(t, err)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"rateAgreementId": agreementID.String(),
	}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured.RateAgreementID)
	assert.Equal(t, agreementID, *repo.captured.RateAgreementID)
	rows := result.(map[string]any)["imports"].([]rateImportRow)
	require.Len(t, rows, 1)
	assert.Equal(t, 5, rows[0].Changed)
}
