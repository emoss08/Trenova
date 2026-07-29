import { useAttentionSummary } from "@/hooks/use-attention";
import { analytics } from "@/lib/queries/analytics";
import { queries } from "@/lib/queries";
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
   * Set on the home page, where the briefing bar reads the shipment payload
   * even when no widget on the canvas does. The preset editor leaves it unset
   * so an administrator arranging orientation widgets fetches nothing extra.
   */
  briefing?: boolean;
};

function widgetReadsReceivables(widget: HomeWidget): boolean {
  if (RECEIVABLES_WIDGETS.has(widget.key)) return true;
  if (widget.config.metric && RECEIVABLES_METRICS.has(widget.config.metric)) return true;
  return (widget.config.metrics ?? []).some((metric) => RECEIVABLES_METRICS.has(metric));
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
    ...queries.ar.dashboardKpis(),
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
    (options.briefing === true ||
      widgets.some((widget) => SHIPMENT_ANALYTICS_WIDGETS.has(widget.key)));
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
