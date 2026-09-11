package fiscaljobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
)

const (
	eventFiscalYearCloseReadiness = "fiscal_year_close_readiness"

	// readinessBlockerLimit caps how many reasons go into one notice. A
	// controller acts on the first few; the rest are on the close screen.
	readinessBlockerLimit = 5
)

// NotifyCloseReadinessActivity reports fiscal years that are past their end date
// and still open. It never closes anything: a year-end close is gated on audit
// work no system can observe, so the job surfaces what is outstanding and leaves
// the decision with the controller.
func (a *Activities) NotifyCloseReadinessActivity(
	ctx context.Context,
	payload *CloseReadinessPayload,
) (*CloseReadinessResult, error) {
	logger := activity.GetLogger(ctx)
	result := &CloseReadinessResult{}

	tenantInfo := pagination.TenantInfo{
		OrgID: payload.OrganizationID,
		BuID:  payload.BusinessUnitID,
	}
	now := timeutils.NowUnix()

	years, err := a.fyRepo.GetExpiredOpenFiscalYears(
		ctx,
		repositories.GetExpiredOpenFiscalYearsRequest{
			OrgID:      payload.OrganizationID,
			BuID:       payload.BusinessUnitID,
			BeforeDate: now,
		},
	)
	if err != nil {
		return nil, temporaltype.NewRetryableError("Failed to list expired fiscal years", err).
			ToTemporalError()
	}

	result.YearsReviewed = len(years)

	for _, fy := range years {
		notified, readErr := a.reportCloseReadiness(ctx, tenantInfo, fy, now)
		if readErr != nil {
			logger.Warn("Failed to report close readiness",
				"fiscalYearId", fy.ID.String(),
				"error", readErr,
			)
			result.Errors = append(result.Errors, readErr.Error())
			continue
		}
		if notified {
			result.Notified++
		}
		activity.RecordHeartbeat(ctx, result.Notified)
	}

	return result, nil
}

// reportCloseReadiness sends at most one notice per year per week. The weekly key
// is what stops a daily job turning into a daily nag while still re-raising a
// year that stays open.
func (a *Activities) reportCloseReadiness(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fy *fiscalyear.FiscalYear,
	now int64,
) (bool, error) {
	correlation := fmt.Sprintf("fiscal-year-close-%s-%s", fy.ID, isoWeekKey(now))

	exists, err := a.notification.ExistsRecent(
		ctx,
		repositories.ExistsRecentNotificationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			EventType:      eventFiscalYearCloseReadiness,
			CorrelationID:  correlation,
			Since:          now - int64(7*24*time.Hour/time.Second),
		},
	)
	if err != nil || exists {
		return false, err
	}

	blockers, err := a.fySvc.GetCloseBlockers(ctx, repositories.GetFiscalYearByIDRequest{
		ID:         fy.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return false, err
	}

	title, message := readinessWording(fy, blockers, now)

	buID := tenantInfo.BuID
	if _, err = a.notification.Create(ctx, &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		Channel:        notification.ChannelGlobal,
		EventType:      eventFiscalYearCloseReadiness,
		Priority:       readinessPriority(blockers, fy, now),
		Title:          title,
		Message:        message,
		Data: map[string]any{
			"fiscalYearId":   fy.ID.String(),
			"fiscalYearName": fy.Name,
			"year":           fy.Year,
			"canClose":       blockers.CanClose,
			"blockerCount":   len(blockers.Blockers),
			"daysPastEnd":    daysPastEnd(fy, now),
		},
		CorrelationID: &correlation,
		Source:        "fiscaljobs.NotifyCloseReadiness",
	}); err != nil {
		return false, err
	}

	return true, nil
}

func readinessWording(
	fy *fiscalyear.FiscalYear,
	blockers *fiscalclose.Result,
	now int64,
) (title, message string) {
	days := daysPastEnd(fy, now)

	if blockers.CanClose {
		return fmt.Sprintf("%s is ready to close", fy.Name),
			fmt.Sprintf(
				"%s ended %d days ago and nothing is blocking its close. Closing it posts the year-end entries and carries balances into the next year.",
				fy.Name,
				days,
			)
	}

	reasons := make([]string, 0, readinessBlockerLimit)
	for _, blocker := range blockers.Blockers {
		if blocker == nil {
			continue
		}
		if len(reasons) == readinessBlockerLimit {
			reasons = append(reasons, "…")
			break
		}
		reasons = append(reasons, blocker.Message)
	}

	return fmt.Sprintf("%s cannot be closed yet", fy.Name),
		fmt.Sprintf(
			"%s ended %d days ago and is still open. Outstanding: %s",
			fy.Name,
			days,
			strings.Join(reasons, " "),
		)
}

// readinessPriority lifts the notice once a year has been hanging open long
// enough that it is a reporting problem rather than a scheduling one.
func readinessPriority(
	blockers *fiscalclose.Result,
	fy *fiscalyear.FiscalYear,
	now int64,
) notification.Priority {
	if daysPastEnd(fy, now) >= 60 {
		return notification.PriorityHigh
	}
	if blockers.CanClose {
		return notification.PriorityLow
	}

	return notification.PriorityMedium
}

func daysPastEnd(fy *fiscalyear.FiscalYear, now int64) int {
	if now <= fy.EndDate {
		return 0
	}

	return int((now - fy.EndDate) / int64(24*time.Hour/time.Second))
}

// isoWeekKey buckets a timestamp into an ISO year-week, so a daily schedule
// resolves to the same key all week.
func isoWeekKey(now int64) string {
	year, week := time.Unix(now, 0).UTC().ISOWeek()

	return fmt.Sprintf("%04d-W%02d", year, week)
}
