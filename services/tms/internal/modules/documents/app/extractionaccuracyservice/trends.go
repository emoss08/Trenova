package extractionaccuracyservice

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionaccuracyports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	driftNoticeLimit   = 25
	driftNoticeSource  = "extraction-accuracy-drift"
	driftNoticeLinkURL = "/admin/agent-control?tab=quality&quality=extraction&extraction=accuracy"
)

func (s *Service) ProviderTrends(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*extractionaccuracyports.ExtractionProviderTrends, error) {
	return s.providerTrends(ctx, tenant, s.now())
}

func (s *Service) providerTrends(
	ctx context.Context,
	tenant pagination.TenantInfo,
	now int64,
) (*extractionaccuracyports.ExtractionProviderTrends, error) {
	window := aicorrection.NewTrendWindow(now)
	totals, err := s.corrections.WeeklyTotalsByProvider(
		ctx,
		&aicorrectionports.WeeklyAICorrectionTotalsRequest{
			TenantInfo: tenant,
			Task:       aicorrection.TaskShipmentDraftExtraction,
			Since:      window.Since(),
		},
	)
	if err != nil {
		return nil, err
	}

	trends := aicorrection.BuildProviderTrends(window, totals)
	report := &extractionaccuracyports.ExtractionProviderTrends{
		Weeks:             window.Weeks,
		CheckedWeek:       window.CheckedWeek,
		BaselineStart:     window.BaselineStart,
		Providers:         make([]extractionaccuracyports.ExtractionProviderTrend, 0, len(trends)),
		DriftPoints:       aicorrection.DriftPoints,
		MinWeekFields:     aicorrection.MinDriftWeekFields,
		MinBaselineFields: aicorrection.MinDriftBaselineFields,
	}
	for i := range trends {
		trend := &trends[i]
		entry := extractionaccuracyports.ExtractionProviderTrend{
			ProviderID: trend.ProviderID,
			Weeks:      trend.Weeks,
			Checked:    trend.Checked,
			Baseline:   trend.Baseline,
			DropPoints: trend.DropPoints,
			Comparable: trend.Comparable,
			Drifting:   trend.Drifting,
		}
		provider, removed, lookupErr := s.trendProvider(ctx, tenant, trend.ProviderID)
		if lookupErr != nil {
			return nil, lookupErr
		}
		entry.ProviderRemoved = removed
		if provider != nil {
			entry.ProviderName = provider.Name
			entry.Model = provider.Model
		}
		report.Providers = append(report.Providers, entry)
	}

	return report, nil
}

func (s *Service) trendProvider(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
) (*aiprovider.Provider, bool, error) {
	provider, err := s.providers.GetByID(ctx, aiproviderports.GetAIProviderByIDRequest{
		ID:         providerID,
		TenantInfo: tenant,
	})
	switch {
	case err == nil:
		return provider, false, nil
	case errortypes.IsNotFoundError(err):
		return nil, true, nil
	default:
		return nil, false, err
	}
}

func (s *Service) CheckDrift(
	ctx context.Context,
	req *extractionaccuracyports.ExtractionDriftCheckRequest,
) (int, error) {
	now := req.Now
	if now <= 0 {
		now = s.now()
	}

	report, err := s.providerTrends(ctx, req.TenantInfo, now)
	if err != nil {
		return 0, err
	}

	notified := 0
	for i := range report.Providers {
		trend := &report.Providers[i]
		if !trend.Drifting || trend.ProviderRemoved {
			continue
		}
		s.l.Warn("extraction accuracy drifted below its own baseline",
			zap.String("organizationId", req.TenantInfo.OrgID.String()),
			zap.String("providerId", trend.ProviderID.String()),
			zap.Float64("checkedAccuracy", trend.Checked.Accuracy),
			zap.Float64("baselineAccuracy", trend.Baseline.Accuracy),
		)
		if s.notifyDrift(ctx, req.TenantInfo, report, trend, now) {
			notified++
		}
	}

	return notified, nil
}

func (s *Service) notifyDrift(
	ctx context.Context,
	tenant pagination.TenantInfo,
	report *extractionaccuracyports.ExtractionProviderTrends,
	trend *extractionaccuracyports.ExtractionProviderTrend,
	now int64,
) bool {
	correlation := trend.ProviderID.String() + ":" + strconv.FormatInt(report.CheckedWeek, 10)
	sent, err := s.notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant:      pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID},
		Resource:    permission.ResourceAIProvider,
		Operation:   permission.OpUpdate,
		Limit:       driftNoticeLimit,
		Now:         now,
		DedupeSince: report.CheckedWeek,
		Notification: notification.Notification{
			EventType:     extractionaccuracyports.ExtractionAccuracyDriftEvent,
			Priority:      notification.PriorityHigh,
			Title:         driftTitle(trend),
			Message:       driftMessage(trend),
			Source:        driftNoticeSource,
			CorrelationID: &correlation,
			Data: map[string]any{
				"link":             driftNoticeLinkURL,
				"providerId":       trend.ProviderID.String(),
				"weekStart":        report.CheckedWeek,
				"checkedAccuracy":  trend.Checked.Accuracy,
				"baselineAccuracy": trend.Baseline.Accuracy,
			},
		},
	})
	if err != nil {
		s.l.Warn("could not tell anyone extraction accuracy drifted",
			zap.String("providerId", trend.ProviderID.String()),
			zap.Error(err),
		)
		return false
	}

	return sent > 0
}

func driftTitle(trend *extractionaccuracyports.ExtractionProviderTrend) string {
	name := trend.ProviderName
	if name == "" {
		name = "An extraction provider"
	}

	return fmt.Sprintf("%s read documents less accurately last week", name)
}

func driftMessage(trend *extractionaccuracyports.ExtractionProviderTrend) string {
	return fmt.Sprintf(
		"It read %s of confirmed fields correctly in the week of %s, against %s over the %d weeks before. "+
			"Review document extraction accuracy in AI Control.",
		percent(trend.Checked.Accuracy),
		timeutils.FormatCalendarDate(trend.Checked.WeekStart, time.UTC),
		percent(trend.Baseline.Accuracy),
		aicorrection.DriftBaselineWeeks,
	)
}

func percent(rate float64) string {
	return fmt.Sprintf("%.1f%%", rate*100)
}
