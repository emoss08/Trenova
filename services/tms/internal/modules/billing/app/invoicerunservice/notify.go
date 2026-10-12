package invoicerunservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	eventScheduledRunFailed   = "invoice_run_scheduled_failed"
	scheduledRunNoticeSource  = "invoicerunservice.SweepDueSchedules"
	statementsLink            = "/billing/queue?view=statements"
	maxScheduledRunNotices    = 25
	maxFailedGroupsInMessage  = 3
	scheduledRunNoticeRepeats = 24 * time.Hour
)

type permittedNotifier interface {
	NotifyPermitted(ctx context.Context, req notificationservice.NotifyPermittedRequest) (int, error)
}

// A scheduled run bills unattended, so when it cannot build, cannot commit, or
// leaves some customers unbilled, nobody is watching the screen to see it. The
// people who may commit invoice runs are told instead, once a day at most for
// the same problem: the sweep retries hourly and would otherwise repeat itself.

type scheduledRunProblem struct {
	tenantInfo  pagination.TenantInfo
	key         periodKey
	correlation string
	title       string
	message     string
	runID       string
}

func (s *Service) notifyBuildFailed(ctx context.Context, tenantInfo pagination.TenantInfo, key periodKey, cause error) {
	s.notifyScheduledRunProblem(ctx, &scheduledRunProblem{
		tenantInfo:  tenantInfo,
		key:         key,
		correlation: fmt.Sprintf("build:%s:%d", key.cycle, key.periodEnd),
		title:       "A scheduled invoice run could not be built",
		message: fmt.Sprintf(
			"%s statements for the period ending %s were not billed: %s. The run is tried again every hour.",
			key.cycle, periodEndLabel(key.periodEnd), cause.Error(),
		),
	})
}

func (s *Service) notifyCommitFailed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key periodKey,
	runID string,
	cause error,
) {
	s.notifyScheduledRunProblem(ctx, &scheduledRunProblem{
		tenantInfo:  tenantInfo,
		key:         key,
		correlation: "commit:" + runID,
		runID:       runID,
		title:       "A scheduled invoice run could not be committed",
		message: fmt.Sprintf(
			"%s statements for the period ending %s were not billed: %s. Open the run to commit it by hand.",
			key.cycle, periodEndLabel(key.periodEnd), cause.Error(),
		),
	})
}

func (s *Service) notifyGroupsFailed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key periodKey,
	commit *invoicerunports.CommitInvoiceRunResult,
) {
	if commit == nil || commit.ErrorCount == 0 || commit.Run == nil {
		return
	}

	runID := commit.Run.ID.String()
	s.notifyScheduledRunProblem(ctx, &scheduledRunProblem{
		tenantInfo:  tenantInfo,
		key:         key,
		correlation: "groups:" + runID,
		runID:       runID,
		title:       fmt.Sprintf("%d customers were not billed by a scheduled invoice run", commit.ErrorCount),
		message: fmt.Sprintf(
			"%s statements for the period ending %s billed %d of %d customers. %s Their freight stays on their statements for the next run or for billing by hand.",
			key.cycle,
			periodEndLabel(key.periodEnd),
			commit.SuccessCount,
			commit.TotalCount,
			failedGroupsSummary(commit.Results),
		),
	})
}

func (s *Service) notifyScheduledRunProblem(ctx context.Context, problem *scheduledRunProblem) {
	if s.notifier == nil {
		return
	}

	now := timeutils.NowUnix()
	correlation := problem.correlation
	data := map[string]any{
		"link":      statementsLink,
		"cycle":     string(problem.key.cycle),
		"periodEnd": problem.key.periodEnd,
	}
	related := map[string]any{}
	if problem.runID != "" {
		data["runId"] = problem.runID
		related["invoiceRunId"] = problem.runID
	}

	if _, err := s.notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant:      problem.tenantInfo,
		Resource:    permission.ResourceInvoiceRun,
		Operation:   permission.OpApprove,
		Limit:       maxScheduledRunNotices,
		Now:         now,
		DedupeSince: now - int64(scheduledRunNoticeRepeats.Seconds()),
		Notification: notification.Notification{
			EventType:       eventScheduledRunFailed,
			Priority:        notification.PriorityHigh,
			Title:           problem.title,
			Message:         problem.message,
			Source:          scheduledRunNoticeSource,
			CorrelationID:   &correlation,
			Data:            data,
			RelatedEntities: related,
		},
	}); err != nil {
		s.l.Warn("could not tell anyone a scheduled invoice run failed",
			zap.String("correlation", correlation), zap.Error(err))
	}
}

func failedGroupsSummary(results []invoicerunports.CommitGroupResult) string {
	reasons := make([]string, 0, maxFailedGroupsInMessage)
	failed := 0
	for i := range results {
		result := &results[i]
		if result.Success || result.Skipped {
			continue
		}
		failed++
		if len(reasons) < maxFailedGroupsInMessage {
			reasons = append(reasons, result.GroupLabel+": "+result.Error)
		}
	}
	if len(reasons) == 0 {
		return ""
	}

	summary := strings.Join(reasons, "; ")
	if failed > len(reasons) {
		summary += fmt.Sprintf("; and %d more", failed-len(reasons))
	}

	return summary + "."
}

func periodEndLabel(periodEnd int64) string {
	return time.Unix(periodEnd, 0).UTC().Format("Jan 2, 2006")
}
