import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import {
  shipmentTypeTableGraphQLConfig,
  type ShipmentTypeRow,
} from "@/lib/graphql/shipment-type-table";
import { shipmentTypeService } from "@/services/shipment-type";
import type { ShipmentType } from "@/types/shipment-type";
import { useQueryClient } from "@tanstack/react-query";
import type { CellEditCommitFn } from "@trenova/shared/lib/cell-editing-feature";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { getColumns } from "./shipment-type-columns";
import { shipmentTypeInlinePatch } from "./shipment-type-inline-edit";
import { ShipmentTypePanel } from "./shipment-type-panel";
import { translate } from "@trenova/shared/i18n/runtime";

export default function ShipmentTypeTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: ShipmentTypeRow[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        shipmentTypeService.bulkUpdateStatus({
          shipmentTypeIds: ids as string[],
          status: status as ShipmentType["status"],
        }),
        {
          loading: translate("Updating status..."),
          success: translate("Status updated successfully"),
          error: translate("Failed to update status"),
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["shipment-type-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const handleCellEditCommit = useCallback<CellEditCommitFn<ShipmentTypeRow>>(
    async ({ rowId, columnId, value, batch }) => {
      const patch = shipmentTypeInlinePatch(columnId, value);
      if (!patch) return;
      await shipmentTypeService.patch(rowId, patch);
      if (batch) return;
      await queryClient.invalidateQueries({
        queryKey: ["shipment-type-list"],
        refetchType: "all",
      });
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<ShipmentTypeRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update status"),
        loadingLabel: t("Updating..."),
        icon: CheckCircleIcon,
        options: statusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  return (
    <DataTable<ShipmentTypeRow>
      name="Shipment Type"
      emptyTitle={t("No shipment types yet")}
      queryKey="shipment-type-list"
      graphql={shipmentTypeTableGraphQLConfig}
      resource={Resource.ShipmentType}
      columns={columns}
      dockActions={dockActions}
      TablePanel={ShipmentTypePanel}
      enableRowSelection
      onCellEditCommit={handleCellEditCommit}
    />
  );
}
