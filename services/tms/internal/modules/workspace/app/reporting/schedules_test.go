package reporting

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type listedSchedules struct {
	reportports.ReportScheduleRepository

	requests []*reportports.ListReportSchedulesRequest
}

func (r *listedSchedules) List(
	_ context.Context,
	req *reportports.ListReportSchedulesRequest,
) ([]*report.ReportSchedule, error) {
	r.requests = append(r.requests, req)

	return []*report.ReportSchedule{}, nil
}

func TestListSchedules_ScopesTheListToTheCaller(t *testing.T) {
	t.Parallel()

	repo := &listedSchedules{}
	svc := &Service{l: zap.NewNop(), scheduleRepo: repo}
	userID := pulid.MustNew("usr_")
	definitionID := pulid.MustNew("rdef_")

	_, err := svc.ListSchedules(t.Context(), &ListSchedulesRequest{
		Request:      planRequest(userID),
		DefinitionID: definitionID,
		EnabledOnly:  true,
		Limit:        25,
	})
	require.NoError(t, err)

	require.Len(t, repo.requests, 1)
	assert.Equal(t, userID, repo.requests[0].VisibleTo)
	assert.Equal(t, userID, repo.requests[0].TenantInfo.UserID)
	assert.Equal(t, definitionID, repo.requests[0].DefinitionID)
	assert.True(t, repo.requests[0].EnabledOnly)
	assert.Equal(t, 25, repo.requests[0].Limit)
}

func TestListSchedules_RefusesARequestWithNoPerson(t *testing.T) {
	t.Parallel()

	repo := &listedSchedules{}
	svc := &Service{l: zap.NewNop(), scheduleRepo: repo}

	_, err := svc.ListSchedules(t.Context(), &ListSchedulesRequest{
		Request: planRequest(pulid.Nil),
	})

	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	assert.Empty(t, repo.requests)
}
