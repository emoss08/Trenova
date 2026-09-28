package reporting

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errPlanWrote = errors.New("a plan wrote")

type planDefinitions struct {
	repositories.ReportDefinitionRepository

	definition *report.ReportDefinition
}

func (r *planDefinitions) GetByID(
	context.Context,
	*repositories.GetReportDefinitionRequest,
) (*report.ReportDefinition, error) {
	copied := *r.definition

	return &copied, nil
}

func (r *planDefinitions) Update(
	context.Context,
	*report.ReportDefinition,
	pulid.ID,
) (*report.ReportDefinition, error) {
	return nil, errPlanWrote
}

func (r *planDefinitions) Delete(
	context.Context,
	*repositories.DeleteReportDefinitionRequest,
) error {
	return errPlanWrote
}

type planDashboards struct {
	repositories.ReportDashboardRepository

	dashboard *report.Dashboard
}

func (r *planDashboards) GetByID(
	context.Context,
	*repositories.GetReportDashboardRequest,
) (*report.Dashboard, error) {
	copied := *r.dashboard

	return &copied, nil
}

type planRuns struct {
	repositories.ReportRunRepository

	run *report.ReportRun
}

func (r *planRuns) GetByID(
	context.Context,
	*repositories.GetReportRunRequest,
) (*report.ReportRun, error) {
	copied := *r.run

	return &copied, nil
}

type planSchedules struct {
	repositories.ReportScheduleRepository

	schedule *report.ReportSchedule
}

func (r *planSchedules) GetByID(
	context.Context,
	*repositories.GetReportScheduleRequest,
) (*report.ReportSchedule, error) {
	copied := *r.schedule

	return &copied, nil
}

func (r *planSchedules) Update(
	context.Context,
	*report.ReportSchedule,
) (*report.ReportSchedule, error) {
	return nil, errPlanWrote
}

func planRequest(userID pulid.ID) Request {
	return Request{TenantInfo: pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: userID,
	}}
}

func TestPlanDeleteDefinition_OnlyTheOwner(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	definition := &report.ReportDefinition{ID: pulid.MustNew("rd_"), Name: "Aging", OwnerID: owner}
	svc := &Service{l: zap.NewNop(), defRepo: &planDefinitions{definition: definition}}

	planned, err := svc.PlanDeleteDefinition(t.Context(), &GetDefinitionRequest{
		Request: planRequest(owner), DefinitionID: definition.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, definition.ID, planned.ID)

	_, err = svc.PlanDeleteDefinition(t.Context(), &GetDefinitionRequest{
		Request: planRequest(pulid.MustNew("usr_")), DefinitionID: definition.ID,
	})
	var denied *errortypes.AuthorizationError
	require.ErrorAs(t, err, &denied)
}

func TestPlanResetCannedFork_PutsBackTheBuiltInDefinition(t *testing.T) {
	t.Parallel()

	registry := canned.Default()
	entries := registry.All()
	require.NotEmpty(t, entries)
	entry := entries[0]

	owner := pulid.MustNew("usr_")
	fork := &report.ReportDefinition{
		ID:        pulid.MustNew("rd_"),
		Name:      "My aging",
		OwnerID:   owner,
		Kind:      report.DefinitionKindCannedFork,
		CannedKey: entry.Key,
		Status:    report.DefinitionStatusDraft,
	}
	svc := &Service{l: zap.NewNop(), defRepo: &planDefinitions{definition: fork}, canned: registry}

	change, err := svc.PlanResetCannedFork(t.Context(), &GetDefinitionRequest{
		Request: planRequest(owner), DefinitionID: fork.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, report.DefinitionStatusDraft, change.Before.Status)
	assert.Equal(t, report.DefinitionStatusActive, change.After.Status)
	assert.Equal(t, entry.Version, change.After.CannedVersion)

	fork.Kind = report.DefinitionKindCustom
	_, err = svc.PlanResetCannedFork(t.Context(), &GetDefinitionRequest{
		Request: planRequest(owner), DefinitionID: fork.ID,
	})
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
}

func TestPlanDeleteDashboard_OnlyTheOwner(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	dashboard := &report.Dashboard{ID: pulid.MustNew("rdb_"), Name: "Ops", OwnerID: owner}
	svc := &Service{l: zap.NewNop(), dashboardRepo: &planDashboards{dashboard: dashboard}}

	planned, err := svc.PlanDeleteDashboard(t.Context(), &GetDashboardRequest{
		Request: planRequest(owner), DashboardID: dashboard.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, dashboard.ID, planned.ID)

	_, err = svc.PlanDeleteDashboard(t.Context(), &GetDashboardRequest{
		Request: planRequest(pulid.MustNew("usr_")), DashboardID: dashboard.ID,
	})
	var denied *errortypes.AuthorizationError
	require.ErrorAs(t, err, &denied)
}

func TestPlanCancelRun_OnlyAnUnfinishedRunOfTheRequester(t *testing.T) {
	t.Parallel()

	requester := pulid.MustNew("usr_")
	run := &report.ReportRun{
		ID:            pulid.MustNew("rrun_"),
		RequestedByID: requester,
		Status:        report.RunStatusRunning,
	}
	svc := &Service{l: zap.NewNop(), runRepo: &planRuns{run: run}}

	change, err := svc.PlanCancelRun(t.Context(), &GetRunRequest{
		Request: planRequest(requester), RunID: run.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, report.RunStatusRunning, change.Before.Status)
	assert.Equal(t, report.RunStatusCanceled, change.After.Status)

	_, err = svc.PlanCancelRun(t.Context(), &GetRunRequest{
		Request: planRequest(pulid.MustNew("usr_")), RunID: run.ID,
	})
	var denied *errortypes.AuthorizationError
	require.ErrorAs(t, err, &denied)

	run.Status = report.RunStatusSucceeded
	_, err = svc.PlanCancelRun(t.Context(), &GetRunRequest{
		Request: planRequest(requester), RunID: run.ID,
	})
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
}

func TestPlanUpdateAndDeleteSchedule_OnlyTheOwnerAndNothingSaved(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	definition := &report.ReportDefinition{ID: pulid.MustNew("rd_"), Name: "Aging", OwnerID: owner}
	schedule := &report.ReportSchedule{
		ID:             pulid.MustNew("rsch_"),
		DefinitionID:   definition.ID,
		CronExpression: "0 7 * * 1",
		Timezone:       "America/Chicago",
		Formats:        []string{"xlsx"},
		Delivery:       &report.ScheduleDelivery{EmailRecipients: []string{"ops@example.com"}},
		Enabled:        true,
		RunAsID:        owner,
		Version:        4,
	}
	svc := &Service{
		l:            zap.NewNop(),
		defRepo:      &planDefinitions{definition: definition},
		scheduleRepo: &planSchedules{schedule: schedule},
		orgRepo:      previewOrganizations{},
	}

	change, err := svc.PlanUpdateSchedule(t.Context(), &SaveScheduleRequest{
		Request:         planRequest(owner),
		ScheduleID:      schedule.ID,
		DefinitionID:    definition.ID,
		CronExpression:  "0 6 * * 1-5",
		Timezone:        "America/Chicago",
		Formats:         []string{"csv"},
		EmailRecipients: []string{"ops@example.com"},
		Enabled:         true,
		Version:         schedule.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, "0 7 * * 1", change.Before.CronExpression)
	assert.Equal(t, "0 6 * * 1-5", change.After.CronExpression)
	assert.Equal(t, []string{"csv"}, change.After.Formats)
	assert.Equal(t, definition.ID, change.Definition.ID)

	_, err = svc.PlanUpdateSchedule(t.Context(), &SaveScheduleRequest{
		Request:        planRequest(owner),
		ScheduleID:     schedule.ID,
		DefinitionID:   definition.ID,
		CronExpression: "every monday",
		Timezone:       "America/Chicago",
		Formats:        []string{"csv"},
		Enabled:        true,
		Version:        schedule.Version,
	})
	require.Error(t, err)

	planned, err := svc.PlanDeleteSchedule(t.Context(), &GetScheduleRequest{
		Request: planRequest(owner), ScheduleID: schedule.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, schedule.ID, planned.ID)

	_, err = svc.PlanDeleteSchedule(t.Context(), &GetScheduleRequest{
		Request: planRequest(pulid.MustNew("usr_")), ScheduleID: schedule.ID,
	})
	var denied *errortypes.AuthorizationError
	require.ErrorAs(t, err, &denied)
}
