import { CustomerMix } from "@/components/work-modules/customer-mix";
import { LaneHeatmap } from "@/components/work-modules/lane-heatmap";
import { WidgetNeedsSetup, WidgetShell, WidgetSkeleton } from "../widget-shell";
import { HomeReportTile } from "../home-report-tile";
import type { WidgetProps } from "../widget-registry";

export function ReportWidget({ widget }: WidgetProps) {
  const { definitionId, cannedKey, chartId, columnId, limit } = widget.config;

  if (!definitionId && !cannedKey) {
    return (
      <WidgetShell title={widget.title || "Report"} scroll={false}>
        <WidgetNeedsSetup message="Choose the report this tile shows." />
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
  if (!widget.config.dashboardId) {
    return (
      <WidgetShell title={widget.title || "Dashboard"} scroll={false}>
        <WidgetNeedsSetup message="Choose the dashboard this tile links to." />
      </WidgetShell>
    );
  }

  return (
    <WidgetShell
      title={widget.title || "Dashboard"}
      href={`/reports/dashboards/${widget.config.dashboardId}`}
      hrefLabel="Open dashboard"
      scroll={false}
    >
      <div className="flex flex-1 flex-col items-start justify-center gap-1">
        <span className="text-sm font-medium">{widget.title || "Saved dashboard"}</span>
        <span className="text-2xs text-muted-foreground">
          Every tile, filter, and parameter as you left it.
        </span>
      </div>
    </WidgetShell>
  );
}

export function LaneHeatmapWidget({ widget, data }: WidgetProps) {
  return (
    <WidgetShell
      title={widget.title || "Lane Heatmap"}
      href="/shipment-management/shipments"
      scroll={false}
      bodyClassName="p-0"
    >
      {data.shipmentAnalyticsLoading ? (
        <div className="p-3">
          <WidgetSkeleton rows={5} />
        </div>
      ) : (
        // The command-center modules bring their own card chrome, which the
        // widget shell already provides — stripping it avoids a nested frame.
        <div className="min-h-0 flex-1 [&>section]:h-full [&>section]:min-h-0 [&>section]:border-0 [&>section]:bg-transparent">
          <LaneHeatmap data={data.shipmentAnalytics.laneHeatmap} />
        </div>
      )}
    </WidgetShell>
  );
}

export function CustomerMixWidget({ widget, data }: WidgetProps) {
  return (
    <WidgetShell
      title={widget.title || "Customer Mix"}
      href="/customers"
      scroll={false}
      bodyClassName="p-0"
    >
      {data.shipmentAnalyticsLoading ? (
        <div className="p-3">
          <WidgetSkeleton rows={5} />
        </div>
      ) : (
        <div className="min-h-0 flex-1 [&>section]:h-full [&>section]:min-h-0 [&>section]:border-0 [&>section]:bg-transparent">
          <CustomerMix
            customerMix={data.shipmentAnalytics.customerMix}
            tomorrowsPickups={data.shipmentAnalytics.tomorrowsPickups}
          />
        </div>
      )}
    </WidgetShell>
  );
}
