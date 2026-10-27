package apikeyprovider

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey/apikeydb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ kernelports.AnalyticsPageProvider = (*Provider)(nil)

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

func (p *Provider) GetPage() kernelports.AnalyticsPage {
	return kernelports.APIKeyAnalyticsPage
}

func (p *Provider) GetAnalyticsData(
	ctx context.Context,
	opts *kernelports.AnalyticsRequestOptions,
) (kernelports.AnalyticsData, error) {
	log := p.l.With(zap.String("operation", "GetAnalyticsData"), zap.Any("opts", opts))

	tz := timeutils.NormalizeTimezone(opts.Timezone)

	counts, err := p.getKeyCounts(ctx, tz)
	if err != nil {
		log.Error("failed to get key counts", zap.Error(err))
		return nil, err
	}

	requests, err := p.getRequests30d(ctx, tz)
	if err != nil {
		log.Error("failed to get requests 30d", zap.Error(err))
		return nil, err
	}

	total := int(counts.Total)
	return kernelports.AnalyticsData{
		"totalKeys": &TotalKeysCard{
			Count:        total,
			NewThisMonth: int(counts.NewThisMonth),
		},
		"activeKeys": &ActiveKeysCard{
			Count:          int(counts.Active),
			PercentOfTotal: percentOf(counts.Active, total),
		},
		"revokedKeys": &RevokedKeysCard{
			Count:          int(counts.Revoked),
			PercentOfTotal: percentOf(counts.Revoked, total),
		},
		"requests30d": requests,
	}, nil
}

type keyCounts struct {
	Total        int64
	NewThisMonth int64
	Active       int64
	Revoked      int64
}

type dailyRequests struct {
	Day   time.Time
	Total int64
}

func percentOf(count int64, total int) float64 {
	if total <= 0 {
		return 0
	}

	return math.Round(float64(count)/float64(total)*1000) / 10
}

func (p *Provider) getKeyCounts(ctx context.Context, tz string) (keyCounts, error) {
	return dbtx.Read(ctx, p.db, func(ctx context.Context) (keyCounts, error) {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return keyCounts{}, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		now := time.Now().In(loc)
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).Unix()

		k := apikeydb.Keys
		return keel.IntoOne[keyCounts](ctx, p.db.Keel(ctx), k.Pick(
			keel.Count(),
			keel.Count().Filter(k.CreatedAt.Gte(monthStart)),
			keel.Count().Filter(k.Status.Eq(apikey.StatusActive)),
			keel.Count().Filter(k.Status.Eq(apikey.StatusRevoked)),
		))
	})
}

func (p *Provider) getRequests30d(ctx context.Context, tz string) (*Requests30dCard, error) {
	return dbtx.Read(ctx, p.db, func(ctx context.Context) (*Requests30dCard, error) {
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

		u := apikeydb.UsageDailies
		cutoffDay := time.Date(cutoff.Year(), cutoff.Month(), cutoff.Day(), 0, 0, 0, 0, time.UTC)
		rows, err := keel.Into[dailyRequests](
			ctx,
			p.db.Keel(ctx),
			u.Where(u.UsageDate.Gte(cutoffDay)).
				Group(u.UsageDate).
				Pick(u.UsageDate, u.RequestCount.Sum()).
				Order(u.UsageDate.Asc()),
		)
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
	})
}
