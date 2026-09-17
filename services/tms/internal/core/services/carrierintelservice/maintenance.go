package carrierintelservice

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	ledgerRetentionMonths = 13
	purgeBatchSize        = 5000
	maxPurgeBatches       = 200
	digestEventLimit      = 200
)

type TenantCursor = repositories.CarrierIntelTenantCursor

func (s *Service) ListConfiguredTenants(
	ctx context.Context,
	after TenantCursor,
	limit int,
) ([]pagination.TenantInfo, error) {
	controls, err := s.controlRepo.ListConfigured(ctx, &repositories.ListCarrierIntelTenantsRequest{
		After: after,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}
	tenants := make([]pagination.TenantInfo, 0, len(controls))
	for _, control := range controls {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: control.OrganizationID,
			BuID:  control.BusinessUnitID,
		})
	}
	return tenants, nil
}

type MaintenanceResult struct {
	RolledUp       int `json:"rolledUp"`
	LedgerPurged   int `json:"ledgerPurged"`
	PayloadsPurged int `json:"payloadsPurged"`
}

func (s *Service) RunGlobalMaintenance(ctx context.Context) (*MaintenanceResult, error) {
	now := s.now()
	result := &MaintenanceResult{}

	yesterday := timeutils.DayStartUTC(now) - timeutils.SecondsPerDay
	rolled, err := s.usageRepo.RollupDay(ctx, yesterday)
	if err != nil {
		return result, err
	}
	result.RolledUp = rolled
	if rolled, err = s.usageRepo.RollupDay(ctx, timeutils.DayStartUTC(now)); err != nil {
		return result, err
	}
	result.RolledUp += rolled

	cutoff := timeutils.AddMonthsUTC(timeutils.MonthStartUTC(now), -ledgerRetentionMonths)
	for range maxPurgeBatches {
		deleted, deleteErr := s.usageRepo.DeleteOlderThan(ctx, cutoff, purgeBatchSize)
		if deleteErr != nil {
			return result, deleteErr
		}
		result.LedgerPurged += deleted
		if deleted < purgeBatchSize {
			break
		}
	}

	for range maxPurgeBatches {
		purged, purgeErr := s.rawRepo.PurgeExpired(ctx, now, purgeBatchSize)
		if purgeErr != nil {
			return result, purgeErr
		}
		result.PayloadsPurged += purged
		if purged < purgeBatchSize {
			break
		}
	}
	return result, nil
}

func (s *Service) PruneTenantHistory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return 0, err
	}
	return s.snapshotRepo.PruneHistory(ctx, tenantInfo, control.SnapshotHistoryLimit)
}

type DigestResult struct {
	Sent   bool `json:"sent"`
	Events int  `json:"events"`
}

func (s *Service) SendDigest(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*DigestResult, error) {
	now := s.now()
	events, err := s.eventRepo.ListForDigest(ctx, &repositories.ListCarrierIntelDigestEventsRequest{
		TenantInfo: tenantInfo,
		Since:      now - timeutils.SecondsPerDay,
		Limit:      digestEventLimit,
	})
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return &DigestResult{}, nil
	}

	bySeverity := make(map[carrierintel.Severity]int, 5)
	subjects := make(map[string]string, len(events))
	for _, event := range events {
		bySeverity[event.Severity]++
		subjects[event.SubjectID] = event.SubjectName
	}
	names := make([]string, 0, len(subjects))
	for _, name := range subjects {
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) > 5 {
		names = append(names[:5], fmt.Sprintf("and %d more", len(subjects)-5))
	}

	parts := make([]string, 0, 5)
	for _, severity := range []carrierintel.Severity{
		carrierintel.SeverityCritical, carrierintel.SeverityHigh, carrierintel.SeverityMedium,
		carrierintel.SeverityLow, carrierintel.SeverityInfo,
	} {
		if count := bySeverity[severity]; count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, strings.ToLower(severity.String())))
		}
	}

	priority := notification.PriorityMedium
	if bySeverity[carrierintel.SeverityCritical] > 0 {
		priority = notification.PriorityHigh
	}

	sent := s.sendNotification(ctx, &notificationRequest{
		tenant:      tenantInfo,
		eventType:   EventCarrierIntelDigest,
		correlation: "ci-digest-" + timeutils.FormatDateKeyUTC(now),
		priority:    priority,
		title:       fmt.Sprintf("%d carrier changes need review", len(events)),
		message: fmt.Sprintf(
			"Unreviewed carrier intelligence changes from the last day (%s) across %s.",
			strings.Join(parts, ", "), strings.Join(names, ", "),
		),
		link: monitoringLink,
	})
	if sent {
		ids := make([]pulid.ID, 0, len(events))
		for _, event := range events {
			ids = append(ids, event.ID)
		}
		if err = s.eventRepo.MarkNotified(ctx, tenantInfo, ids, now); err != nil {
			return &DigestResult{Sent: true, Events: len(events)}, err
		}
	}
	return &DigestResult{Sent: sent, Events: len(events)}, nil
}

type UsageSummary struct {
	MonthStart     int64
	MonthToDate    decimal.Decimal
	Cap            *decimal.Decimal
	SoftCapPercent int
	ByEndpoint     []repositories.CarrierIntelUsageSummaryRow
	Daily          []*carrierintel.CarrierIntelUsageDaily
}

func (s *Service) Usage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	month time.Time,
) (*UsageSummary, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	now := s.now()
	start := timeutils.MonthStartUTC(now)
	if !month.IsZero() {
		start = timeutils.MonthStartUTC(month.Unix())
	}
	end := timeutils.AddMonthsUTC(start, 1)

	rows, err := s.usageRepo.Summary(ctx, &repositories.ListCarrierIntelUsageRequest{
		TenantInfo: tenantInfo,
		Since:      start,
		Until:      end,
	})
	if err != nil {
		return nil, err
	}
	total := decimal.Zero
	for _, row := range rows {
		total = total.Add(row.EstimatedCost)
	}

	daily, err := s.usageRepo.ListDaily(ctx, &repositories.ListCarrierIntelUsageDailyRequest{
		TenantInfo: tenantInfo,
		FromDay:    timeutils.DayKeyUTC(start),
		ToDay:      timeutils.DayKeyUTC(end - 1),
	})
	if err != nil {
		return nil, err
	}

	return &UsageSummary{
		MonthStart:     start,
		MonthToDate:    total,
		Cap:            control.MonthlySpendCap,
		SoftCapPercent: control.SoftCapPercent,
		ByEndpoint:     rows,
		Daily:          daily,
	}, nil
}
