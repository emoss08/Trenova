import { DataTable } from "@/components/data-table/data-table";
import { shipmentTypeTableGraphQLConfig } from "@/lib/graphql/shipment-type-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import type { ShipmentType } from "@/types/shipment-type";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./shipment-type-columns";
import { ShipmentTypePanel } from "./shipment-type-panel";

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
    />
  );
}
