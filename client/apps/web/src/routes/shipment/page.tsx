import { DataTableLazyComponent, LazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@/components/ui/button";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { analytics } from "@/lib/queries/analytics";
import { queries } from "@/lib/queries";
import { usePermissionStore } from "@/stores/permission-store";
import { Operation, Resource } from "@/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, RefreshCwIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { lazy, useCallback, useMemo, useState } from "react";
import type { CommandCenterTableSummary } from "./_components/command-center/command-center-table";
import { ShipmentMapPanelBoundary } from "./_components/map/map-boundary";

const Table = lazy(() => import("./_components/shipment-table"));
const ShipmentAnalytics = lazy(() => import("./_components/analytics/kpi/kpi-rail"));
const ShipmentMapPanel = lazy(() => import("./_components/map/shipment-map-panel"));
const RightStack = lazy(() => import("./_components/command-center/right-stack"));
const BottomModules = lazy(() => import("./_components/command-center/bottom-modules"));

export function ShipmentsPage() {
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
        queryClient.invalidateQueries({ queryKey: ["shipment-list"] }),
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
        title: "Shipments",
        description: "Operations command center for shipments, assignments, and exceptions.",
        context: (
          <>
            {summary && (
              <div
                aria-label="Live shipment count"
                title={`Updated ${new Date(summary.dataUpdatedAt).toLocaleTimeString()}`}
                className="inline-flex h-5 items-center gap-1 rounded border border-success/25 bg-success/10 px-1.5 font-table text-[10px] text-success tabular-nums"
              >
                <span className="size-1 rounded-full bg-success" />
                Live · {formattedCount}
              </div>
            )}
            {currentOrg && (
              <span className="font-table text-[10px] text-muted-foreground tabular-nums">
                org · {currentOrg.name}
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
              loadingText="Refreshing"
            >
              <RefreshCwIcon className="size-3.5" />
              Refresh
            </Button>
            {canCreateShipment && (
              <Button type="button" size="sm" onClick={handleCreateShipment}>
                <PlusIcon className="size-3.5" />
                New Shipment
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
