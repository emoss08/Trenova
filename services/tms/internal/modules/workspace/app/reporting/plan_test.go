package reporting

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportingtest"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errPlanWrote = errors.New("a plan wrote")

type planDefinitions struct {
	reportports.ReportDefinitionRepository

	definition *report.ReportDefinition
}

func (r *planDefinitions) GetByID(
	context.Context,
	*reportports.GetReportDefinitionRequest,
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
	*reportports.DeleteReportDefinitionRequest,
) error {
	return errPlanWrote
}

type planRuns struct {
	reportports.ReportRunRepository

	run *report.ReportRun
}

func (r *planRuns) GetByID(
	_ context.Context,
	req *reportports.GetReportRunRequest,
) (*report.ReportRun, error) {
	if !req.RequestedBy.IsNil() && req.RequestedBy != r.run.RequestedByID {
		return nil, errortypes.NewNotFoundError("ReportRun not found")
	}
	copied := *r.run

	return &copied, nil
}

func planRequest(userID pulid.ID) Request {
	return Request{TenantInfo: reportingtest.Tenant(userID)}
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

	registry, err := canned.Default()
	require.NoError(t, err)
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
	svc, _ := engineService(t, engineOptions{})
	svc.defRepo = &planDefinitions{definition: fork}
	svc.canned = registry

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
	var missing *errortypes.NotFoundError
	require.ErrorAs(t, err, &missing)

	run.Status = report.RunStatusSucceeded
	_, err = svc.PlanCancelRun(t.Context(), &GetRunRequest{
		Request: planRequest(requester), RunID: run.ID,
	})
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
}
