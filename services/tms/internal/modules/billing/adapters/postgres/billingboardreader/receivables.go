package billingboardreader

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type ReceivablesParams struct {
	fx.In

	Snapshots customerledgerports.ARSnapshotReader
}

type Receivables struct {
	snapshots customerledgerports.ARSnapshotReader
}

func NewReceivables(p ReceivablesParams) *Receivables {
	return &Receivables{snapshots: p.Snapshots}
}

func (*Receivables) Metric() string { return billingboard.MetricReceivables }

func (r *Receivables) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	kpis, err := r.snapshots.GetDashboardKPIs(ctx, customerledgerports.GetARAnalyticsRequest{
		TenantInfo: req.Tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("read the receivables figures: %w", err)
	}
	read := &boardcatalog.MetricRead{
		Title:   i18n.T(ctx, "Receivables"),
		Source:  i18n.T(ctx, "the receivables ledger"),
		Figures: map[string]*boardcatalog.Value{},
	}
	if kpis == nil {
		return read, nil
	}
	if kpis.Overview != nil {
		open := boardcatalog.Money(kpis.Overview.TotalOpenMinor, "")
		overdue := boardcatalog.Money(kpis.Overview.OverdueMinor, "")
		read.Figures["arTotalOpen"] = &open
		read.Figures["arOverdue"] = &overdue
	}
	dso := boardcatalog.Number(decimal.NewFromFloat(kpis.CurrentDSODays).Round(1))
	read.Figures["arDaysSalesOutstanding"] = &dso

	return read, nil
}
