import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent, LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { queries } from "@/lib/queries";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { ShipmentCapabilitiesProvider } from "@/lib/shipment-board/capabilities";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlusIcon } from "@trenova/shared/components/icons";
import { createLoader, useQueryStates } from "nuqs";
import { useCallback } from "react";
import { ShipmentBriefing } from "./_components/board/briefing/shipment-briefing";
import { CapacityStrip } from "./_components/board/capacity/capacity-strip";
import { ShipmentSidePanel } from "./_components/board/panel/side-panel";
import { ShipmentRecordActionsProvider } from "./_components/board/record-actions";
import { ShipmentBoard } from "./_components/board/shipment-board";
import { BoardRefreshButton } from "./_components/board/toolbar/board-refresh-button";
import { useBoardPanel } from "./_components/board/use-board-panel";
import {
  SHIPMENT_TABLE_RESOURCE_NAME,
  shipmentPanelDetailQuery,
} from "./_components/shipment-queries";

const loadPanelSearch = createLoader(panelSearchParamsParser);

export const prefetch: RoutePrefetch = ({ request }) => {
  const list: RoutePrefetchQuery[] = [
    { ...queries.shipmentBoard.capabilities(), staleTime: 60_000 },
    { ...queries.tableConfiguration.default(SHIPMENT_TABLE_RESOURCE_NAME), staleTime: Infinity },
  ];

  const { panelType, panelEntityId } = loadPanelSearch(request);
  if (panelType === "edit" && panelEntityId) {
    list.push(shipmentPanelDetailQuery(panelEntityId));
  }

  return list;
};

function ShipmentWorkspace() {
  const [panelOpen, setPanelOpen] = useBoardPanel();

  return (
    <div className="@container/board relative grid min-h-0 flex-1 grid-rows-[auto_minmax(0,1fr)]">
      <div className="border-border flex min-w-0 flex-col gap-3.5 border-b px-5 pt-3 pb-3.5">
        <LazyComponent>
          <ShipmentBriefing />
        </LazyComponent>
        <LazyComponent>
          <CapacityStrip />
        </LazyComponent>
      </div>
      <div className="flex min-h-0 min-w-0 flex-col">
        <DataTableLazyComponent>
          <ShipmentBoard panelOpen={panelOpen} onPanelOpenChange={setPanelOpen} />
        </DataTableLazyComponent>
      </div>
      {panelOpen ? (
        <LazyComponent>
          <ShipmentSidePanel onClose={() => setPanelOpen(false)} />
        </LazyComponent>
      ) : null}
    </div>
  );
}

export function ShipmentsPage() {
  const t = useT();
  const [, setSearchParams] = useQueryStates(panelSearchParamsParser);
  const canCreateShipment = usePermissionStore((state) =>
    state.hasPermission(Resource.Shipment, Operation.Create),
  );

  const handleCreateShipment = useCallback(() => {
    void setSearchParams({ panelType: "create", panelEntityId: null });
  }, [setSearchParams]);

  return (
    <PageLayout
      fill
      bleed
      pageHeaderProps={{
        title: t("Shipments"),
        description: t("Every load on the board, what needs a hand, and who can take it."),
        actions: (
          <>
            <BoardRefreshButton />
            {canCreateShipment && (
              <Button type="button" size="sm" onClick={handleCreateShipment}>
                <PlusIcon className="size-3.5" />
                {t("New shipment")}
              </Button>
            )}
          </>
        ),
      }}
    >
      <ShipmentCapabilitiesProvider>
        <ShipmentRecordActionsProvider>
          <ShipmentWorkspace />
        </ShipmentRecordActionsProvider>
      </ShipmentCapabilitiesProvider>
    </PageLayout>
  );
}
