import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@/types/data-table";
import type { EquipmentManufacturer } from "@/types/equipment-manufacturer";
import { Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./equipment-manufacturer-columns";
import { EquipmentManufacturerPanel } from "./equipment-manufacturer-panel";

export default function EquipmentManufacturerTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: EquipmentManufacturer[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.equipmentManufacturerService.bulkUpdateStatus({
          equipmentManufacturerIds: ids as string[],
          status: status as EquipmentManufacturer["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["equipment-manufacturer-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<EquipmentManufacturer>[]>(
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
    <DataTable<EquipmentManufacturer>
      name="Equipment Manufacturer"
      link="/equipment-manufacturers/"
      queryKey="equipment-manufacturer-list"
      exportModelName="equipment-manufacturer"
      resource={Resource.EquipmentManufacturer}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={EquipmentManufacturerPanel}
    />
  );
}
