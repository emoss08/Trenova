package extractionevalservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDriftNotifier struct {
	sent []notificationservice.NotifyPermittedRequest
	err  error
}

func (f *fakeDriftNotifier) NotifyPermitted(
	_ context.Context,
	req notificationservice.NotifyPermittedRequest,
) (int, error) {
	f.sent = append(f.sent, req)
	if f.err != nil {
		return 0, f.err
	}
	return 1, nil
}

func (w *world) provider(name string) *aiprovider.Provider {
	provider := &aiprovider.Provider{ID: pulid.MustNew("aip_"), Name: name, Model: name + "-model"}
	w.providers.providers[provider.ID] = provider
	return provider
}

func (w *world) drifting(providerID pulid.ID) {
	window := aicorrection.NewTrendWindow(testNow)
	w.corrections.weekly = append(w.corrections.weekly,
		aicorrection.WeekTotal{ProviderID: providerID, WeekStart: window.CheckedWeek, Corrections: 6, Scored: 120, Correct: 90},
		aicorrection.WeekTotal{ProviderID: providerID, WeekStart: window.BaselineStart, Corrections: 20, Scored: 400, Correct: 380},
	)
}

func (w *world) steady(providerID pulid.ID) {
	window := aicorrection.NewTrendWindow(testNow)
	w.corrections.weekly = append(w.corrections.weekly,
		aicorrection.WeekTotal{ProviderID: providerID, WeekStart: window.CheckedWeek, Corrections: 6, Scored: 120, Correct: 110},
		aicorrection.WeekTotal{ProviderID: providerID, WeekStart: window.BaselineStart, Corrections: 20, Scored: 400, Correct: 370},
	)
}

func TestProviderTrendsNameEachProviderAndItsDrift(t *testing.T) {
	t.Parallel()

	w := newWorld()
	fine := w.provider("Frontier")
	tuned := w.provider("Fine-tuned Qwen")
	removed := pulid.MustNew("aip_")
	w.drifting(tuned.ID)
	w.steady(fine.ID)
	w.corrections.weekly = append(w.corrections.weekly, aicorrection.WeekTotal{
		ProviderID: removed,
		WeekStart:  aicorrection.NewTrendWindow(testNow).CheckedWeek,
		Scored:     5,
		Correct:    5,
	})

	report, err := w.svc.ProviderTrends(t.Context(), testTenant())
	require.NoError(t, err)

	window := aicorrection.NewTrendWindow(testNow)
	assert.Equal(t, window.Since(), w.corrections.weeklySince)
	assert.Equal(t, window.Weeks, report.Weeks)
	assert.Equal(t, window.CheckedWeek, report.CheckedWeek)
	assert.Equal(t, aicorrection.DriftPoints, report.DriftPoints)
	require.Len(t, report.Providers, 3)

	byID := map[pulid.ID]services.ExtractionProviderTrend{}
	for _, trend := range report.Providers {
		byID[trend.ProviderID] = trend
	}
	assert.Equal(t, "Fine-tuned Qwen", byID[tuned.ID].ProviderName)
	assert.Equal(t, "Fine-tuned Qwen-model", byID[tuned.ID].Model)
	assert.True(t, byID[tuned.ID].Drifting)
	assert.InDelta(t, 20, byID[tuned.ID].DropPoints, 0.0001)
	assert.False(t, byID[fine.ID].Drifting)
	assert.True(t, byID[removed].ProviderRemoved)
	assert.Empty(t, byID[removed].ProviderName)
	assert.Len(t, byID[tuned.ID].Weeks, aicorrection.TrendWeeks)
}

func TestProviderTrendsFailWhenAProviderCannotBeRead(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.steady(pulid.MustNew("aip_"))
	w.providers.err = errUpstream

	_, err := w.svc.ProviderTrends(t.Context(), testTenant())
	require.ErrorIs(t, err, errUpstream)
}

func TestCheckDriftTellsThePeopleWhoChooseProviders(t *testing.T) {
	t.Parallel()

	w := newWorld()
	notifier := &fakeDriftNotifier{}
	w.svc.notifier = notifier
	tuned := w.provider("Fine-tuned Qwen")
	fine := w.provider("Frontier")
	w.drifting(tuned.ID)
	w.steady(fine.ID)
	w.drifting(pulid.MustNew("aip_"))

	notified, err := w.svc.CheckDrift(t.Context(), &services.ExtractionDriftCheckRequest{
		TenantInfo: testTenant(),
	})
	require.NoError(t, err)

	assert.Equal(t, 1, notified, "a removed provider is not reported")
	require.Len(t, notifier.sent, 1)
	req := notifier.sent[0]
	window := aicorrection.NewTrendWindow(testNow)
	assert.Equal(t, permission.ResourceAIProvider, req.Resource)
	assert.Equal(t, permission.OpUpdate, req.Operation)
	assert.Equal(t, window.CheckedWeek, req.DedupeSince, "one notice per provider per week")
	assert.Equal(t, services.ExtractionAccuracyDriftEvent, req.Notification.EventType)
	require.NotNil(t, req.Notification.CorrelationID)
	assert.Contains(t, *req.Notification.CorrelationID, tuned.ID.String())
	assert.Equal(t,
		"Fine-tuned Qwen read documents less accurately last week",
		req.Notification.Title,
	)
	assert.Contains(t, req.Notification.Message, "75.0%")
	assert.Contains(t, req.Notification.Message, "95.0%")
}

func TestCheckDriftSurvivesAFailedNotice(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.svc.notifier = &fakeDriftNotifier{err: errors.New("mail down")}
	w.drifting(w.provider("Fine-tuned Qwen").ID)

	notified, err := w.svc.CheckDrift(t.Context(), &services.ExtractionDriftCheckRequest{
		TenantInfo: testTenant(),
		Now:        testNow,
	})
	require.NoError(t, err)
	assert.Zero(t, notified)
}

func TestCheckDriftReturnsARepositoryFailure(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.corrections.weeklyErr = errUpstream

	_, err := w.svc.CheckDrift(t.Context(), &services.ExtractionDriftCheckRequest{
		TenantInfo: testTenant(),
	})
	require.ErrorIs(t, err, errUpstream)
}
