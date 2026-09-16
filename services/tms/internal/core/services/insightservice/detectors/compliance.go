package detectors

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/shopspring/decimal"
)

// CredentialExpiryKey identifies the expiring-credentials detector.
const CredentialExpiryKey = "credential-expiry"

// Credential thresholds.
//
// The horizon is 45 days because that is roughly how long a medical card or a
// hazmat endorsement takes to renew once someone starts chasing it. A 30-day
// warning arrives after the point where it would have been easy.
//
// Severity is by count of affected drivers rather than by days remaining,
// because one expiring licence is a calendar entry and eight at once is a
// capacity problem. An already-expired credential escalates on its own: that
// driver cannot legally run today.
const credentialHorizonDays = 45

var (
	credentialWarningWorkers  = decimal.NewFromInt(3)
	credentialCriticalWorkers = decimal.NewFromInt(8)
)

// CredentialExpiry reports groups of worker credentials falling due.
//
// It groups by credential type rather than listing drivers because that is how
// the work gets done — somebody spends an afternoon on medical cards, not one
// driver's entire file — and because the count per type is what says whether
// this costs the operation trucks next month.
type CredentialExpiry struct {
	metrics repositories.InsightMetricsRepository
}

func NewCredentialExpiry(metrics repositories.InsightMetricsRepository) *CredentialExpiry {
	return &CredentialExpiry{metrics: metrics}
}

func (d *CredentialExpiry) Key() string                   { return CredentialExpiryKey }
func (d *CredentialExpiry) Category() insight.Category    { return insight.CategoryCompliance }
func (d *CredentialExpiry) Resource() permission.Resource { return permission.ResourceWorker }
func (d *CredentialExpiry) Operation() permission.Operation {
	return permission.OpRead
}

func (d *CredentialExpiry) Detect(
	ctx context.Context,
	params detector.Params,
) ([]detector.Finding, error) {
	rows, err := d.metrics.ExpiringCredentials(ctx, repositories.ExpiringCredentialsRequest{
		TenantInfo: params.TenantInfo,
		From:       params.WindowEnd,
		Through:    params.WindowEnd + int64(credentialHorizonDays)*secondsPerDay,
	})
	if err != nil {
		return nil, fmt.Errorf("read expiring credentials: %w", err)
	}

	findings := make([]detector.Finding, 0, len(rows))
	for _, row := range rows {
		if finding, ok := d.findingFor(row, params); ok {
			findings = append(findings, finding)
		}
	}

	return findings, nil
}

func (d *CredentialExpiry) findingFor(
	row repositories.ExpiringCredentialRow,
	params detector.Params,
) (detector.Finding, bool) {
	if row.WorkerCount == 0 {
		return detector.Finding{}, false
	}

	// An optional credential expiring is administration. A required one expiring
	// takes the driver off the road, so only those raise a card on their own;
	// optional ones need enough of them to be worth an afternoon.
	if !row.IsRequired && decimal.NewFromInt(row.WorkerCount).LessThan(credentialWarningWorkers) {
		return detector.Finding{}, false
	}

	daysToFirst := daysBetween(params.WindowEnd, row.EarliestExpiry)
	severity := d.severityFor(row)

	return detector.Finding{
		DedupeKey: detector.DedupeKey(CredentialExpiryKey, row.CredentialTypeID.String()),
		Subject:   row.CredentialTypeName,
		Headline:  d.headlineFor(row, daysToFirst),
		Severity:  severity,
		Metrics:   d.metricsFor(row, daysToFirst),
		Links: []insight.Link{
			detector.FilteredLink("Workers", detector.RouteWorkers, int(row.WorkerCount)),
		},
	}, true
}

// severityFor escalates on credentials that have already lapsed. A driver whose
// medical card expired yesterday is a different problem from one whose expires
// in six weeks, and the card must not average the two into something calm.
func (d *CredentialExpiry) severityFor(row repositories.ExpiringCredentialRow) insight.Severity {
	if row.AlreadyExpired > 0 && row.IsRequired {
		return insight.SeverityCritical
	}

	severity := detector.SeverityFor(
		decimal.NewFromInt(row.WorkerCount),
		credentialWarningWorkers,
		credentialCriticalWorkers,
	)

	// A required credential is never merely informational: somebody has to act
	// before the date, and an Info card is the one nobody reads.
	if row.IsRequired && severity == insight.SeverityInfo {
		return insight.SeverityWarning
	}

	return severity
}

func (d *CredentialExpiry) headlineFor(
	row repositories.ExpiringCredentialRow,
	daysToFirst decimal.Decimal,
) string {
	if row.AlreadyExpired > 0 {
		return fmt.Sprintf(
			"%d workers hold %s that has already expired, with %d more expiring soon",
			row.AlreadyExpired,
			row.CredentialTypeName,
			row.WorkerCount-row.AlreadyExpired,
		)
	}

	return fmt.Sprintf(
		"%d workers have %s expiring within %d days, the first in %s days",
		row.WorkerCount,
		row.CredentialTypeName,
		credentialHorizonDays,
		daysToFirst.String(),
	)
}

func (d *CredentialExpiry) metricsFor(
	row repositories.ExpiringCredentialRow,
	daysToFirst decimal.Decimal,
) []insight.Metric {
	metrics := []insight.Metric{
		detector.Count(
			"affectedWorkers",
			"Workers affected",
			row.WorkerCount,
			insight.DirectionHigherIsWorse,
		),
		detector.Days(
			"daysToFirstExpiry",
			"First expiry in",
			daysToFirst,
			insight.DirectionLowerIsWorse,
		),
	}

	if row.AlreadyExpired > 0 {
		metrics = append(metrics, detector.Count(
			"alreadyExpired",
			"Already expired",
			row.AlreadyExpired,
			insight.DirectionHigherIsWorse,
		))
	}

	return metrics
}
