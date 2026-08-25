import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import { shipmentTypeTableGraphQLConfig } from "@/lib/graphql/shipment-type-table";
import { apiService } from "@/services/api";
import type { ShipmentType } from "@/types/shipment-type";
import { useQueryClient } from "@tanstack/react-query";
import type { CellEditCommitFn } from "@trenova/shared/lib/cell-editing-feature";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./shipment-type-columns";
import { ShipmentTypePanel } from "./shipment-type-panel";

const INLINE_EDITABLE_FIELDS = new Set<keyof ShipmentType>(["code", "description"]);

export default function ShipmentTypeTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: ShipmentType[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.shipmentTypeService.bulkUpdateStatus({
          shipmentTypeIds: ids as string[],
          status: status as ShipmentType["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
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

  const handleCellEditCommit = useCallback<CellEditCommitFn<ShipmentType>>(
    async ({ rowId, columnId, value }) => {
      const field = columnId as keyof ShipmentType;
      if (!INLINE_EDITABLE_FIELDS.has(field)) return;
      if (field === "code" && (value === null || value === "")) {
        throw new Error("Code is required.");
      }

      await apiService.shipmentTypeService.patch(rowId, { [field]: value });
      await queryClient.invalidateQueries({
        queryKey: ["shipment-type-list"],
        refetchType: "all",
      });

      toast.success("Shipment Type updated");
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<ShipmentType>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: "Update Status",
        loadingLabel: "Updating...",
        icon: CircleCheckIcon,
        options: statusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate],
  );

  return (
    <DataTable<ShipmentType>
      name="Shipment Type"
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
