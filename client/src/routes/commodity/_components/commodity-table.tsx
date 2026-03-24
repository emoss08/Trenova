import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { Commodity } from "@/types/commodity";
import type { DockAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./commodity-columns";
import { CommodityPanel } from "./commodity-panel";

export default function CommodityTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: Commodity[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.commodityService.bulkUpdateStatus({
          commodityIds: ids as string[],
          status: status as Commodity["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["commodity-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<Commodity>[]>(
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
    <DataTable<Commodity>
      name="Commodity"
      link="/commodities/"
      queryKey="commodity-list"
      exportModelName="commodity"
      resource={Resource.Commodity}
      columns={columns}
      dockActions={dockActions}
      TablePanel={CommodityPanel}
      enableRowSelection
    />
  );
}
