import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@/types/data-table";
import type { EquipmentType } from "@/types/equipment-type";
import { Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./equipment-type-columns";
import { EquipmentTypePanel } from "./equipment-type-panel";

export default function EquipmentTypeTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: EquipmentType[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.equipmentTypeService.bulkUpdateStatus({
          equipmentTypeIds: ids as string[],
          status: status as EquipmentType["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["equipment-type-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<EquipmentType>[]>(
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
    <DataTable<EquipmentType>
      name="Equipment Type"
      link="/equipment-types/"
      queryKey="equipment-type-list"
      exportModelName="equipment-type"
      resource={Resource.EquipmentType}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={EquipmentTypePanel}
    />
  );
}
