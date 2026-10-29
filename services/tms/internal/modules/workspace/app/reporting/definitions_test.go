package reporting

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type listedDefinitions struct {
	reportports.ReportDefinitionRepository

	requests []*reportports.ListReportDefinitionsRequest
}

func (r *listedDefinitions) List(
	_ context.Context,
	req *reportports.ListReportDefinitionsRequest,
) ([]*report.ReportDefinition, error) {
	r.requests = append(r.requests, req)

	return []*report.ReportDefinition{}, nil
}

func TestListDefinitions_FiltersVisibilityInTheQuery(t *testing.T) {
	t.Parallel()

	repo := &listedDefinitions{}
	svc := &Service{l: zap.NewNop(), defRepo: repo}
	userID := pulid.MustNew("usr_")

	_, err := svc.ListDefinitions(t.Context(), &ListDefinitionsRequest{
		Request: planRequest(userID),
		Limit:   10,
		Offset:  20,
	})
	require.NoError(t, err)

	require.Len(t, repo.requests, 1)
	assert.Equal(t, userID, repo.requests[0].ViewerID)
	assert.Equal(t, 10, repo.requests[0].Limit)
	assert.Equal(t, 20, repo.requests[0].Offset)
}
