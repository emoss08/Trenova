package agentshadowservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentshadow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type definitions struct {
	repositories.AgentDefinitionRepository
	missing bool
}

func (d definitions) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	if d.missing {
		return nil, errortypes.NewNotFoundError("Agent")
	}
	return &agentdefinition.Definition{}, nil
}

type shadowRows struct {
	proposals []agentshadow.Proposal
	changes   []agentshadow.PersonChange
	asked     *repositories.ListPersonChangesRequest
	since     int64
}

func (s *shadowRows) ListShadowProposals(
	_ context.Context,
	req *repositories.ListShadowProposalsRequest,
) ([]agentshadow.Proposal, error) {
	s.since = req.Since
	return s.proposals, nil
}

func (s *shadowRows) ListPersonChanges(
	_ context.Context,
	req *repositories.ListPersonChangesRequest,
) ([]agentshadow.PersonChange, error) {
	s.asked = req
	return s.changes, nil
}

func TestReportReadsEachRecordOnceOverTheClampedPeriod(t *testing.T) {
	t.Parallel()

	rows := &shadowRows{
		proposals: []agentshadow.Proposal{
			{TargetID: "shp_1", CreatedAt: 1000, Fields: []string{"status"}},
			{TargetID: "shp_1", CreatedAt: 1100, Fields: []string{"status"}},
			{CreatedAt: 1200},
		},
		changes: []agentshadow.PersonChange{
			{ResourceID: "shp_1", Timestamp: 1500, Fields: []string{"status"}},
		},
	}
	svc := &Service{definitions: definitions{}, shadow: rows, now: func() int64 { return 100 * 24 * 60 * 60 }}

	report, err := svc.Report(t.Context(), &services.AgentShadowReportRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		AgentID:    pulid.MustNew("agdef_"),
		Days:       365,
	})

	require.NoError(t, err)
	assert.Equal(t, agentshadow.MaxDays, report.Days)
	assert.Equal(t, int64(10*24*60*60), rows.since)
	assert.Equal(t, []string{"shp_1"}, rows.asked.ResourceIDs)
	assert.Equal(t, 3, report.Recorded)
	assert.Equal(t, 2, report.Matched)
	assert.Equal(t, 1, report.Unanswered)
}

func TestReportOfAMissingAgentReadsNothing(t *testing.T) {
	t.Parallel()

	rows := &shadowRows{}
	svc := &Service{definitions: definitions{missing: true}, shadow: rows, now: func() int64 { return 0 }}

	_, err := svc.Report(t.Context(), &services.AgentShadowReportRequest{})

	require.Error(t, err)
	assert.Nil(t, rows.asked)
}
