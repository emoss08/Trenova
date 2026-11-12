package webwatchjobs

import (
	"context"
	"sync/atomic"

	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/sourcegraph/conc/pool"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Runner  agentextensionports.WebWatchRunner
	Waits   agentextensionports.WebWatchWaits
	Watches agentextensionports.WebWatchRepository
	Logger  *zap.Logger
}

type Activities struct {
	runner  agentextensionports.WebWatchRunner
	waits   agentextensionports.WebWatchWaits
	watches agentextensionports.WebWatchRepository
	l       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		runner:  p.Runner,
		waits:   p.Waits,
		watches: p.Watches,
		l:       p.Logger.Named("job.web-watch"),
	}
}

type SweepResult struct {
	Checked int `json:"checked"`
	Ended   int `json:"ended"`
	Failed  int `json:"failed"`
}

// CheckDueWebWatchesActivity claims the watches whose check is due, which
// moves each one's next check to the regular interval, and checks them a few
// at a time. A watch the check found ended leaves the sweep; one whose check
// failed is tried again soon.
func (a *Activities) CheckDueWebWatchesActivity(ctx context.Context) (*SweepResult, error) {
	now := timeutils.NowUnix()
	due, err := a.watches.ClaimDueChecks(ctx, &agentextensionports.ClaimWebWatchChecksRequest{
		Now:   now,
		Next:  now + int64(checkEvery.Seconds()),
		Limit: claimLimit,
	})
	if err != nil {
		return nil, err
	}

	var checked, ended, failed atomic.Int64
	work := pool.New().WithContext(ctx).WithMaxGoroutines(checkConcurrency)
	for _, ref := range due {
		work.Go(func(ctx context.Context) error {
			defer activity.RecordHeartbeat(ctx, ref.ID.String())

			switch a.checkOne(ctx, ref) {
			case outcomeEnded:
				ended.Add(1)
			case outcomeFailed:
				failed.Add(1)
			case outcomeChecked:
			}
			checked.Add(1)

			return nil
		})
	}
	if err = work.Wait(); err != nil {
		return nil, err
	}

	return &SweepResult{
		Checked: int(checked.Load()),
		Ended:   int(ended.Load()),
		Failed:  int(failed.Load()),
	}, nil
}

type outcome int

const (
	outcomeChecked outcome = iota
	outcomeEnded
	outcomeFailed
)

// checkOne checks one watch, bounded by checkTimeout, and settles when it is
// next checked.
func (a *Activities) checkOne(ctx context.Context, ref agentextensionports.WebWatchRef) outcome {
	checkCtx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	done, err := a.check(checkCtx, ref.Tenant, ref.ID)
	switch {
	case err != nil:
		a.l.Warn("web watch check failed; it is tried again soon",
			zap.String("watchId", ref.ID.String()), zap.Error(err))
		retryAt := timeutils.NowUnix() + int64(retryAfter.Seconds())
		if sErr := a.watches.ScheduleCheck(ctx, ref.Tenant, ref.ID, retryAt); sErr != nil {
			a.l.Warn("could not bring a failed web watch check forward; it runs at the next one",
				zap.String("watchId", ref.ID.String()), zap.Error(sErr))
		}

		return outcomeFailed
	case done:
		if cErr := a.watches.ClearCheck(ctx, ref.Tenant, ref.ID); cErr != nil {
			a.l.Warn(
				"could not take an ended web watch off the sweep; its next check ends it again",
				zap.String("watchId", ref.ID.String()),
				zap.Error(cErr),
			)
		}

		return outcomeEnded
	default:
		return outcomeChecked
	}
}

// check checks the watch once, hands what it found to the work waiting on it,
// and records it handed over only once that worked. A watch the vendor could
// no longer run, or that was stopped, tells the waiting work why it ended. It
// reports whether the watch has ended.
func (a *Activities) check(
	ctx context.Context,
	tenant pagination.TenantInfo,
	watchID pulid.ID,
) (bool, error) {
	check, err := a.runner.CheckWebWatch(ctx, tenant, watchID)
	if err != nil {
		return false, err
	}
	if check.Watch == nil {
		return true, nil
	}
	if len(check.Undelivered) > 0 {
		if err = a.waits.NotifyWebWatch(ctx, tenant, check.Watch, check.Undelivered); err != nil {
			return false, err
		}
		if err = a.runner.MarkWebWatchDelivered(ctx, tenant, check.Undelivered); err != nil {
			return false, err
		}
	}
	if check.Ended && check.Watch.Status != extensionwebwatch.WebWatchStatusEnded {
		a.waits.EndWebWatchWaits(ctx, tenant, check.Watch)
	}

	return check.Ended, nil
}
