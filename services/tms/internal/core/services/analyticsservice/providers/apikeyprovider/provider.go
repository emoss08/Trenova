package apikeyprovider

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ services.AnalyticsPageProvider = (*Provider)(nil)

type ProviderParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type Provider struct {
	l  *zap.Logger
	db *postgres.Connection
}

func NewProvider(p ProviderParams) *Provider {
	return &Provider{
		l:  p.Logger.Named("analyticsprovider.apikey"),
		db: p.DB,
	}
}

func (p *Provider) GetPage() services.AnalyticsPage {
	return services.APIKeyAnalyticsPage
}

func (p *Provider) GetAnalyticsData(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (services.AnalyticsData, error) {
	log := p.l.With(zap.String("operation", "GetAnalyticsData"), zap.Any("opts", opts))

	tz := timeutils.NormalizeTimezone(opts.Timezone)

	totalKeys, err := p.getTotalKeys(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get total keys", zap.Error(err))
		return nil, err
	}

	activeKeys, err := p.getActiveKeys(ctx, opts.OrgID, opts.BuID, totalKeys.Count)
	if err != nil {
		log.Error("failed to get active keys", zap.Error(err))
		return nil, err
	}

	revokedKeys, err := p.getRevokedKeys(ctx, opts.OrgID, opts.BuID, totalKeys.Count)
	if err != nil {
		log.Error("failed to get revoked keys", zap.Error(err))
		return nil, err
	}

	requests, err := p.getRequests30d(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get requests 30d", zap.Error(err))
		return nil, err
	}

	return services.AnalyticsData{
		"totalKeys":   totalKeys,
		"activeKeys":  activeKeys,
		"revokedKeys": revokedKeys,
		"requests30d": requests,
	}, nil
}

func (p *Provider) getTotalKeys(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*TotalKeysCard, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	total, err := p.db.DB().NewSelect().
		Model((*apikey.Key)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ak.organization_id = ?", orgID).
				Where("ak.business_unit_id = ?", buID)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(loc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).Unix()

	newThisMonth, err := p.db.DB().NewSelect().
		Model((*apikey.Key)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ak.organization_id = ?", orgID).
				Where("ak.business_unit_id = ?", buID).
				Where("ak.created_at >= ?", monthStart)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return &TotalKeysCard{
		Count:        total,
		NewThisMonth: newThisMonth,
	}, nil
}

func (p *Provider) getActiveKeys(
	ctx context.Context,
	orgID, buID pulid.ID,
	total int,
) (*ActiveKeysCard, error) {
	count, err := p.db.DB().NewSelect().
		Model((*apikey.Key)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ak.organization_id = ?", orgID).
				Where("ak.business_unit_id = ?", buID).
				Where("ak.status = ?", apikey.StatusActive)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	var pct float64
	if total > 0 {
		pct = math.Round(float64(count)/float64(total)*1000) / 10
	}

	return &ActiveKeysCard{
		Count:          count,
		PercentOfTotal: pct,
	}, nil
}

func (p *Provider) getRevokedKeys(
	ctx context.Context,
	orgID, buID pulid.ID,
	total int,
) (*RevokedKeysCard, error) {
	count, err := p.db.DB().NewSelect().
		Model((*apikey.Key)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ak.organization_id = ?", orgID).
				Where("ak.business_unit_id = ?", buID).
				Where("ak.status = ?", apikey.StatusRevoked)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	var pct float64
	if total > 0 {
		pct = math.Round(float64(count)/float64(total)*1000) / 10
	}

	return &RevokedKeysCard{
		Count:          count,
		PercentOfTotal: pct,
	}, nil
}

func (p *Provider) getRequests30d(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*Requests30dCard, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	now := time.Now().In(loc)
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	cutoff := time.Date(
		thirtyDaysAgo.Year(),
		thirtyDaysAgo.Month(),
		thirtyDaysAgo.Day(),
		0,
		0,
		0,
		0,
		loc,
	)

	type dailyRow struct {
		Day   time.Time `bun:"day"`
		Total int64     `bun:"total"`
	}

	rows := make([]dailyRow, 0)
	err = p.db.DB().NewSelect().
		TableExpr("api_key_usage_daily u").
		ColumnExpr("u.usage_date AS day").
		ColumnExpr("SUM(u.request_count) AS total").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("u.organization_id = ?", orgID).
				Where("u.business_unit_id = ?", buID).
				Where("u.usage_date >= ?", cutoff.Format("2006-01-02"))
		}).
		GroupExpr("u.usage_date").
		OrderExpr("u.usage_date ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	rowMap := make(map[string]int64, len(rows))
	var grandTotal int64
	for _, row := range rows {
		grandTotal += row.Total
		rowMap[row.Day.Format("2006-01-02")] = row.Total
	}

	sparkline := make([]*SparklinePoint, 0, 31)
	for d := cutoff; !d.After(now); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		sparkline = append(sparkline, &SparklinePoint{
			Day:   d.Format("Jan 2"),
			Value: rowMap[key],
		})
	}

	return &Requests30dCard{
		Total:     grandTotal,
		Sparkline: sparkline,
	}, nil
}
