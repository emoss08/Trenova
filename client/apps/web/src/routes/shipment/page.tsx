import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent, LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { analytics } from "@/lib/queries/analytics";
import { queries } from "@/lib/queries";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, RefreshCwIcon } from "lucide-react";
import { createLoader, useQueryStates } from "nuqs";
import { lazy, useCallback, useMemo, useState } from "react";
import type { CommandCenterTableSummary } from "./_components/command-center/command-center-table";
import { ShipmentMapPanelBoundary } from "./_components/map/map-boundary";
import { formatDateInUserTimezone } from "@trenova/shared/lib/date";
import {
  SHIPMENT_LIST_KEY,
  SHIPMENT_TABLE_RESOURCE_NAME,
  shipmentPanelDetailQuery,
} from "./_components/shipment-queries";

const Table = lazy(() => import("./_components/shipment-table"));
const ShipmentAnalytics = lazy(() => import("./_components/analytics/kpi-rail"));
const ShipmentMapPanel = lazy(() => import("./_components/map/shipment-map-panel"));
const RightStack = lazy(() => import("./_components/command-center/right-stack"));
const BottomModules = lazy(() => import("./_components/command-center/bottom-modules"));

const loadPanelSearch = createLoader(panelSearchParamsParser);

// What the first paint asks for unconditionally: the header's organization badge, the
// KPI rail, the table's saved default view, and the map's key. The rows, the map pins
// and the right stack wait on the table (backgroundQueriesEnabled) and are left to it.
export const prefetch: RoutePrefetch = ({ request }) => {
  const list: RoutePrefetchQuery[] = [
    queries.userOrganization.all(),
    analytics.get("shipment-management"),
    { ...queries.tableConfiguration.default(SHIPMENT_TABLE_RESOURCE_NAME), staleTime: Infinity },
    queries.integration.runtimeConfig("GoogleMaps"),
  ];

  const { panelType, panelEntityId } = loadPanelSearch(request);
  if (panelType === "edit" && panelEntityId) {
    list.push(shipmentPanelDetailQuery(panelEntityId));
  }

  return list;
};

export function ShipmentsPage() {
  const t = useT();

  const queryClient = useQueryClient();
  const [, setSearchParams] = useQueryStates(panelSearchParamsParser);
  const [summary, setSummary] = useState<CommandCenterTableSummary | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const { data: organizations } = useQuery(queries.userOrganization.all());
  const canCreateShipment = usePermissionStore((state) =>
    state.hasPermission(Resource.Shipment, Operation.Create),
  );
  const currentOrg = organizations?.find((org) => org.isCurrent);

  const formattedCount = useMemo(() => {
    if (!summary) return null;
    return new Intl.NumberFormat().format(summary.totalCount);
  }, [summary]);
  const backgroundQueriesEnabled = summary?.backgroundQueriesEnabled ?? false;

  const handleCreateShipment = useCallback(() => {
    void setSearchParams({ panelType: "create", panelEntityId: null });
  }, [setSearchParams]);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    try {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [SHIPMENT_LIST_KEY] }),
        queryClient.invalidateQueries({
          queryKey: analytics.get("shipment-management").queryKey,
        }),
      ]);
    } finally {
      setIsRefreshing(false);
    }
  }, [queryClient]);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Shipments"),
        description: t("Operations command center for shipments, assignments, and exceptions."),
        context: (
          <>
            {summary && (
              <div
                aria-label={t("Live shipment count")}
                title={`Updated ${formatDateInUserTimezone(new Date(summary.dataUpdatedAt), {
                  hour: "numeric",
                  minute: "2-digit",
                  second: "2-digit",
                })}`}
                className="border-success/25 bg-success/10 font-table text-success inline-flex h-5 items-center gap-1 rounded border px-1.5 text-[10px] tabular-nums"
              >
                <span className="bg-success size-1 rounded-full" />
                {t("Live · {0}", formattedCount)}
              </div>
            )}
            {currentOrg && (
              <span className="font-table text-muted-foreground text-[10px] tabular-nums">
                {t("org · {0}", currentOrg.name)}
              </span>
            )}
          </>
        ),
        actions: (
          <>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleRefresh}
              isLoading={isRefreshing}
              loadingText={t("Refreshing")}
            >
              <RefreshCwIcon className="size-3.5" />
              {t("Refresh")}
            </Button>
            {canCreateShipment && (
              <Button type="button" size="sm" onClick={handleCreateShipment}>
                <PlusIcon className="size-3.5" />
                {t("New Shipment")}
              </Button>
            )}
          </>
        ),
      }}
    >
      <div className="cc-workspace flex flex-col gap-3">
        <LazyComponent>
          <ShipmentAnalytics />
        </LazyComponent>
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-[minmax(0,1fr)_minmax(320px,380px)]">
          <ShipmentMapPanelBoundary>
            <ShipmentMapPanel backgroundEnabled={backgroundQueriesEnabled} />
          </ShipmentMapPanelBoundary>
          <div className="relative h-[clamp(420px,calc(100vh-380px),540px)] min-h-0">
            <LazyComponent>
              <RightStack backgroundEnabled={backgroundQueriesEnabled} />
            </LazyComponent>
          </div>
        </div>
        <DataTableLazyComponent>
          <Table onSummaryChange={setSummary} />
        </DataTableLazyComponent>
        <LazyComponent>
          <BottomModules backgroundEnabled={backgroundQueriesEnabled} />
        </LazyComponent>
      </div>
    </PageLayout>
  );
}
