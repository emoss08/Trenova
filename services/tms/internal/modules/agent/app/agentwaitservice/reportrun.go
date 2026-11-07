package agentwaitservice

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentwaitjobs/agentwaitcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	// reportRunRecheckSeconds is how often a run being waited on is read
	// again, for a finish whose notice never reached the wait.
	reportRunRecheckSeconds int64 = 2 * 60
	// reportRunLifetimeSeconds outlasts the longest run (thirty minutes) and
	// its time in the queue.
	reportRunLifetimeSeconds int64 = 2 * 60 * 60
	reportRunSuffix                = " to finish running"
	maxWaitedReportNameRunes       = agentwait.MaxDescriptionRunes - len(reportRunSuffix)
	maxRunErrorRunes               = 300
)

func NewReportRunWaiter(s *Service) agentwaitports.ReportRunWaiter { return s }

// WaitForReportRun parks the conversation that started a report run still
// going. One wait serves a run in a conversation: starting it again keeps the
// wait already open. A run is read as the person who asked for it, so only a
// conversation, which has one, parks on it.
func (s *Service) WaitForReportRun(
	ctx context.Context,
	req *agentwaitports.WaitForReportRunRequest,
) (*agentwait.Wait, error) {
	if req.Actor == nil || req.ReportRunID.IsNil() || req.ThreadID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"A report run is waited on by the conversation that started it",
		)
	}
	open, err := s.waits.ListOpenWatching(ctx, &agentwaitports.ListOpenWaitsWatchingRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: req.Actor.OrganizationID,
			BuID:  req.Actor.BusinessUnitID,
		},
		Kinds:    []agentwait.Kind{agentwait.KindReportRun},
		WatchIDs: []pulid.ID{req.ReportRunID},
	})
	if err != nil {
		return nil, err
	}
	if idx := slices.IndexFunc(open, func(wait *agentwait.Wait) bool {
		return wait.ThreadID == req.ThreadID
	}); idx >= 0 {
		return open[idx], nil
	}

	name := clip(cmp.Or(req.ReportName, "The report"), maxWaitedReportNameRunes)

	return s.Register(ctx, &agentwaitports.RegisterWaitRequest{
		Actor:           req.Actor,
		DefinitionID:    req.DefinitionID,
		ThreadID:        req.ThreadID,
		Delegated:       req.Delegated,
		Kind:            agentwait.KindReportRun,
		Condition:       agentwait.Condition{ReportRunID: req.ReportRunID},
		Description:     name + reportRunSuffix,
		Then:            req.Then,
		LifetimeSeconds: reportRunLifetimeSeconds,
	})
}

// reportRun reads the run a wait follows, as the person who requested it:
// a run is only ever theirs to read.
func (f *facts) reportRun(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID, requester pulid.ID,
) (*report.ReportRun, error) {
	run, err := f.reportRuns.GetByID(ctx, &reportports.GetReportRunRequest{
		TenantInfo:  tenant,
		RunID:       runID,
		RequestedBy: requester,
	})
	if err != nil {
		return nil, missing(err, "report run")
	}

	return run, nil
}

func (f *facts) assessReportRun(
	ctx context.Context,
	wait *agentwait.Wait,
	now int64,
) (*int64, error) {
	run, err := f.reportRun(ctx, tenantOf(wait), wait.Condition.ReportRunID, wait.UserID)
	if err != nil {
		return nil, err
	}
	if run.Status.IsTerminal() {
		return nil, alreadyTrue("%s", runOutcome(run))
	}
	due := now + reportRunRecheckSeconds

	return &due, nil
}

func (f *facts) recheckReportRun(
	ctx context.Context,
	wait *agentwait.Wait,
	now int64,
) (*agentwaitcontract.Check, error) {
	run, err := f.reportRun(ctx, tenantOf(wait), wait.Condition.ReportRunID, wait.UserID)
	if err != nil {
		return goneOr(err, "The report run was removed before it finished.")
	}
	if !run.Status.IsTerminal() {
		due := now + reportRunRecheckSeconds
		return &agentwaitcontract.Check{DueAt: &due}, nil
	}

	return met("%s", runOutcome(run)), nil
}

// reportRunChanged reads the run a stream notice was about and ends the wait
// once the run has finished. The run is announced as it moves; only its end
// picks the work up.
func (s *Service) reportRunChanged(ctx context.Context, wait *agentwait.Wait) {
	run, err := s.facts.reportRun(ctx, tenantOf(wait), wait.Condition.ReportRunID, wait.UserID)
	if err != nil {
		s.l.Debug("could not read a report run a wait follows; its timer reads it again",
			zap.String("wait", wait.ID.String()), zap.Error(err))
		return
	}
	if !run.Status.IsTerminal() {
		return
	}
	s.meetWith(ctx, wait, agentwaitcontract.Met{
		Detail: runOutcome(run),
		At:     s.now(),
		Key:    run.ID.String(),
	})
}

// runOutcome says what a finished run came to, never what its rows hold:
// the work reads them with report_get_run, as the person, when it picks up.
func runOutcome(run *report.ReportRun) string {
	switch {
	case run.Status == report.RunStatusSucceeded:
		outcome := fmt.Sprintf("Report run %s finished with %d rows.", run.ID, run.RowCount)
		if run.Truncated {
			outcome += " It hit the row cap, so it is not the full set."
		}
		return outcome + " Read it with report_get_run before you say anything about it."
	case run.Error != nil && run.Error.Message != "":
		return fmt.Sprintf("Report run %s did not finish: %s", run.ID,
			clip(run.Error.Message, maxRunErrorRunes))
	default:
		return fmt.Sprintf("Report run %s ended as %s and produced nothing.", run.ID, run.Status)
	}
}
