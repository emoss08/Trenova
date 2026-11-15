import { useAttentionSummary } from "@/hooks/use-attention";
import { analytics } from "@/lib/queries/analytics";
import { ar } from "@/lib/queries/ar";
import {
  mergeShipmentAnalyticsWithDefaults,
  type DeepPartial,
  type ShipmentAnalyticsData,
} from "@/lib/shipment-analytics";
import type { HomeWidget } from "@/lib/graphql/home-layout";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

/**
 * Widget keys that read the shipment analytics payload. The canvas issues one
 * request covering all of them rather than one per widget: the provider
 * computes the whole payload in a single pass, so twenty tiles cost what one
 * does.
 */
const SHIPMENT_ANALYTICS_WIDGETS = new Set([
  "kpi",
  "kpi-row",
  "revenue-trend",
  "on-time-goal",
  "fleet-status",
  "lane-heatmap",
  "customer-mix",
  "detention-watch",
  "tomorrows-pickups",
  "map",
]);

/** Metric keys served by the receivables query rather than the shipment one. */
const RECEIVABLES_METRICS = new Set(["arTotalOpen", "arOverdue", "arDaysSalesOutstanding"]);

const RECEIVABLES_WIDGETS = new Set(["ar-snapshot"]);

export type HomeData = {
  shipmentAnalytics: ShipmentAnalyticsData;
  shipmentAnalyticsLoading: boolean;
  /** True once the shipment payload holds real numbers rather than defaults. */
  shipmentAnalyticsReady: boolean;
  attention: ReturnType<typeof useAttentionSummary>["data"];
  attentionLoading: boolean;
  receivables: ReturnType<typeof useHomeReceivables>["data"];
  receivablesLoading: boolean;
};

export type UseHomeDataOptions = {
  /**
   * Set on the home page, where the brief reads the shipment payload
   * even when no widget on the canvas does. The preset editor leaves it unset
   * so an administrator arranging orientation widgets fetches nothing extra.
   */
  brief?: boolean;
};

/**
 * The measures of a widget the plan does not leave out. A locked widget, and a locked
 * measure of a strip, are drawn as the add-on that carries them and read nothing.
 */
export function liveMetricsOf(widget: HomeWidget): string[] {
  if (widget.planLock) return [];
  const locked = new Set(widget.lockedMetrics.map((entry) => entry.metric));
  const configured = [widget.config.metric, ...(widget.config.metrics ?? [])];
  return configured.filter((metric): metric is string => !!metric && !locked.has(metric));
}

/**
 * Whether the plan leaves out everything a widget would show: the widget itself, or every
 * measure it is set to show. Such a widget is not drawn on Home, only in the editor, where the
 * person can see what it was and remove it.
 */
export function isWidgetOffPlan(widget: HomeWidget): boolean {
  if (widget.planLock) return true;
  return widget.lockedMetrics.length > 0 && liveMetricsOf(widget).length === 0;
}

function widgetReadsReceivables(widget: HomeWidget): boolean {
  if (widget.planLock) return false;
  if (RECEIVABLES_WIDGETS.has(widget.key)) return true;
  return liveMetricsOf(widget).some((metric) => RECEIVABLES_METRICS.has(metric));
}

/**
 * Receivables figures, fetched only when a widget on the canvas asks for them.
 * The AR endpoint is heavier than the shipment payload, so it is never pulled
 * speculatively.
 */
function useHomeReceivables(enabled: boolean) {
  const canReadReceivables = usePermissionStore((state) =>
    state.hasPermission(Resource.AccountsReceivable, Operation.Read),
  );

  return useQuery({
    ...ar.dashboardKpis(),
    enabled: enabled && canReadReceivables,
  });
}

/**
 * Everything the canvas fetches on behalf of its widgets. Fetching here rather
 * than inside each widget means every tile shows the same instant's numbers,
 * and a layout with six shipment metrics still makes one request.
 */
export function useHomeData(widgets: HomeWidget[], options: UseHomeDataOptions = {}): HomeData {
  const canReadShipments = usePermissionStore((state) =>
    state.hasPermission(Resource.Shipment, Operation.Read),
  );

  const needsShipmentAnalytics =
    canReadShipments &&
    (options.brief === true ||
      widgets.some((widget) => !widget.planLock && SHIPMENT_ANALYTICS_WIDGETS.has(widget.key)));
  const needsReceivables = widgets.some(widgetReadsReceivables);

  const shipmentQuery = useQuery({
    ...analytics.get("shipment-management"),
    enabled: needsShipmentAnalytics,
  });

  const attentionQuery = useAttentionSummary();
  const receivablesQuery = useHomeReceivables(needsReceivables);

  const shipmentAnalytics = useMemo(
    () =>
      mergeShipmentAnalyticsWithDefaults(
        (shipmentQuery.data ?? {}) as DeepPartial<ShipmentAnalyticsData>,
      ),
    [shipmentQuery.data],
  );

  return {
    shipmentAnalytics,
    shipmentAnalyticsLoading: needsShipmentAnalytics && shipmentQuery.isLoading,
    shipmentAnalyticsReady: shipmentQuery.data != null,
    attention: attentionQuery.data,
    attentionLoading: attentionQuery.isLoading,
    receivables: receivablesQuery.data,
    receivablesLoading: needsReceivables && receivablesQuery.isLoading,
  };
}
