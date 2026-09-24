package airetrievalstatusservice

import (
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const moneyPlaces = 2

func buildStatus(in *statusInputs, monthStart int64) *serviceports.AIRetrievalStatus {
	status := &serviceports.AIRetrievalStatus{
		Availability:           AvailabilityView(in.availability),
		Settings:               SettingsView(in.settings),
		Sources:                SourceStatuses(in.settings, in.totals, in.active),
		MonthStartedAt:         monthStart,
		IndexingCostMonthUSD:   costOf(in.indexing).StringFixed(moneyPlaces),
		IndexingUnpricedCalls:  unpricedOf(in.indexing),
		RetrievalCostMonthUSD:  costOf(in.retrieval).StringFixed(moneyPlaces),
		RetrievalUnpricedCalls: unpricedOf(in.retrieval),
		ModelChange:            ModelChangeProgress(in.settings, in.totals, in.pending),
		ConfiguredModelDiffers: in.configured != "" && in.configured != in.settings.ActiveModelKey,
	}
	status.ConfiguredModelKey = stringutils.Ptr(in.configured)
	for _, source := range status.Sources {
		status.LastIndexedAt = intutils.MaxPointer(status.LastIndexedAt, source.LastIndexedAt)
	}

	return status
}

func costOf(cost *repositories.AIUsageCost) decimal.Decimal {
	if cost == nil {
		return decimal.Zero
	}

	return cost.CostUSD
}

func unpricedOf(cost *repositories.AIUsageCost) int {
	if cost == nil {
		return 0
	}

	return cost.UnpricedCalls
}

func AvailabilityView(availability airetrieval.Availability) *serviceports.AIRetrievalAvailability {
	view := &serviceports.AIRetrievalAvailability{
		Available:          availability.Available,
		ExtensionInstalled: availability.ExtensionInstalled,
	}
	if !availability.Available && availability.Reason.IsValid() {
		view.Reason = stringutils.NilIfEmpty(availability.Reason)
	}
	view.ExtensionVersion = stringutils.Ptr(availability.ExtensionVersion)

	return view
}

func SettingsView(settings *airetrieval.Settings) *serviceports.AIRetrievalSettings {
	view := &serviceports.AIRetrievalSettings{
		MemoryEnabled:            settings.MemoryEnabled,
		DocumentsEnabled:         settings.DocumentsEnabled,
		InboundMessagesEnabled:   settings.InboundMessagesEnabled,
		MonthlyIndexingBudgetUSD: settings.MonthlyIndexingBudgetUSD.StringFixed(moneyPlaces),
		Paused:                   settings.Paused,
		PausedAt:                 settings.PausedAt,
		ActiveModelKey:           stringutils.Ptr(settings.ActiveModelKey),
		Dimensions:               intutils.NilIfZero(settings.Dimensions),
		PendingModelKey:          stringutils.Ptr(settings.PendingModelKey),
		PendingDimensions:        intutils.NilIfZero(settings.PendingDimensions),
		Version:                  settings.Version,
	}
	if settings.Paused && settings.PausedReason.IsValid() {
		view.PausedReason = stringutils.NilIfEmpty(settings.PausedReason)
	}
	view.UpdatedAt = intutils.NilIfZero(settings.UpdatedAt)

	return view
}

func SourceStatuses(
	settings *airetrieval.Settings,
	totals map[airetrieval.SourceType]int,
	counts []repositories.IndexEntryCount,
) []*serviceports.AIRetrievalSourceStatus {
	sourceTypes := airetrieval.AllSourceTypes()
	statuses := make([]*serviceports.AIRetrievalSourceStatus, 0, len(sourceTypes))
	bySource := make(
		map[airetrieval.SourceType]*serviceports.AIRetrievalSourceStatus,
		len(sourceTypes),
	)
	for _, sourceType := range sourceTypes {
		status := &serviceports.AIRetrievalSourceStatus{
			SourceType: sourceType,
			Enabled:    settings.SourceEnabled(sourceType),
			Total:      totals[sourceType],
		}
		statuses = append(statuses, status)
		bySource[sourceType] = status
	}

	for _, count := range counts {
		status, ok := bySource[count.SourceType]
		if !ok {
			continue
		}
		switch count.Status {
		case airetrieval.IndexStatusIndexed:
			status.Indexed += count.Count
		case airetrieval.IndexStatusPending:
			status.Pending += count.Count
		case airetrieval.IndexStatusFailed:
			status.Failed += count.Count
		case airetrieval.IndexStatusSkipped:
			status.Skipped += count.Count
		}
		status.LastIndexedAt = intutils.MaxPointer(status.LastIndexedAt, count.LastIndexedAt)
		status.LastAttemptAt = intutils.MaxPointer(status.LastAttemptAt, count.LastAttemptAt)
	}

	return statuses
}

func ModelChangeProgress(
	settings *airetrieval.Settings,
	totals map[airetrieval.SourceType]int,
	counts []repositories.IndexEntryCount,
) *serviceports.AIRetrievalModelChange {
	if !settings.HasPendingModel() {
		return nil
	}

	change := &serviceports.AIRetrievalModelChange{
		FromModelKey: settings.ActiveModelKey,
		ToModelKey:   settings.PendingModelKey,
		Dimensions:   settings.PendingDimensions,
	}
	for _, sourceType := range settings.EnabledSourceTypes() {
		change.Total += totals[sourceType]
	}
	for _, count := range counts {
		if !settings.SourceEnabled(count.SourceType) {
			continue
		}
		switch count.Status {
		case airetrieval.IndexStatusIndexed, airetrieval.IndexStatusSkipped:
			change.Indexed += count.Count
		case airetrieval.IndexStatusPending:
			change.Pending += count.Count
		case airetrieval.IndexStatusFailed:
			change.Failed += count.Count
		}
	}
	change.Total = max(change.Total, change.Indexed+change.Pending+change.Failed)

	return change
}
