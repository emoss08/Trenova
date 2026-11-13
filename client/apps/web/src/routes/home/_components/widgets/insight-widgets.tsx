import { useT } from "@trenova/shared/i18n/use-t";
import { CustomerMix } from "@/components/work-modules/customer-mix";
import { LaneHeatmap } from "@/components/work-modules/lane-heatmap";
import { WidgetNeedsSetup, WidgetShell, WidgetSkeleton } from "../widget-shell";
import { HomeReportTile } from "../home-report-tile";
import type { WidgetProps } from "../widget-registry";

export function ReportWidget({ widget }: WidgetProps) {
  const t = useT();

  const { definitionId, cannedKey, chartId, columnId, limit } = widget.config;

  if (!definitionId && !cannedKey) {
    return (
      <WidgetShell title={widget.title || t("Report")} scroll={false}>
        <WidgetNeedsSetup message={t("Choose the report this tile shows.")} />
      </WidgetShell>
    );
  }

  return (
    <HomeReportTile
      title={widget.title || undefined}
      definitionId={definitionId ?? undefined}
      cannedKey={cannedKey ?? undefined}
      chartId={chartId ?? undefined}
      columnId={columnId ?? undefined}
      limit={limit ?? undefined}
    />
  );
}

export function DashboardLinkWidget({ widget }: WidgetProps) {
  const t = useT();

  if (!widget.config.dashboardId) {
    return (
      <WidgetShell title={widget.title || t("Dashboard")} scroll={false}>
        <WidgetNeedsSetup message={t("Choose the dashboard this tile links to.")} />
      </WidgetShell>
    );
  }

  return (
    <WidgetShell
      title={widget.title || t("Dashboard")}
      href={`/reports/dashboards/${widget.config.dashboardId}`}
      hrefLabel={t("Open dashboard")}
      scroll={false}
    >
      <div className="flex flex-1 flex-col items-start justify-center gap-1">
        <span className="text-sm font-medium">{widget.title || t("Saved dashboard")}</span>
        <span className="text-2xs text-muted-foreground">
          {t("Every tile, filter, and parameter as you left it.")}
        </span>
      </div>
    </WidgetShell>
  );
}

export function LaneHeatmapWidget({ widget, data }: WidgetProps) {
  const t = useT();

  const heatmap = data.shipmentAnalytics.laneHeatmap;
  const loading = data.shipmentAnalyticsLoading;

  return (
    <WidgetShell
      title={widget.title || t("Lane heatmap")}
      hint={
        heatmap && !loading
          ? t(
              "{0, plural, one {# load} other {# loads}} in {1, plural, one {# day} other {# days}}",
              heatmap.total,
              heatmap.windowDays,
            )
          : undefined
      }
      href="/shipment-management/shipments"
      hrefLabel={t("Open shipments")}
      scroll={false}
    >
      {loading ? <WidgetSkeleton rows={5} /> : <LaneHeatmap data={heatmap} />}
    </WidgetShell>
  );
}

export function CustomerMixWidget({ widget, data }: WidgetProps) {
  const t = useT();

  const customerMix = data.shipmentAnalytics.customerMix;
  const loading = data.shipmentAnalyticsLoading;

  return (
    <WidgetShell
      title={widget.title || t("Customer mix")}
      hint={
        customerMix && !loading
          ? t("Last {0, plural, one {# day} other {# days}}", customerMix.windowDays)
          : undefined
      }
      href="/billing/configuration-files/customers"
      hrefLabel={t("Open customers")}
    >
      {loading ? <WidgetSkeleton rows={5} /> : <CustomerMix customerMix={customerMix} />}
    </WidgetShell>
  );
}
