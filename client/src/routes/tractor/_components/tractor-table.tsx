import { DataTable } from "@/components/data-table/data-table";
import { equipmentStatusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import type { Tractor } from "@/types/tractor";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./tractor-columns";
import { TractorPanel } from "./tractor-panel";

export default function Table() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: Tractor[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.tractorService.bulkUpdateStatus({
          tractorIds: ids as string[],
          status: status as Tractor["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["tractor-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<Tractor>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: "Update Status",
        loadingLabel: "Updating...",
        icon: CircleCheckIcon,
        options: equipmentStatusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate],
  );

  return (
    <DataTable<Tractor>
      name="Tractor"
      link="/tractors/"
      queryKey="tractor-list"
      exportModelName="tractor"
      resource={Resource.Tractor}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={TractorPanel}
      extraSearchParams={{
        includeFleetDetails: true,
        includeEquipmentDetails: true,
        includeWorkerDetails: true,
      }}
    />
  );
}
