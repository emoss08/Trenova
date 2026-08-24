import { DataTable } from "@/components/data-table/data-table";
import { carrierStatusChoices } from "@/lib/choices";
import { carrierTableGraphQLConfig } from "@/lib/graphql/carrier-table";
import { apiService } from "@/services/api";
import type { Carrier } from "@trenova/shared/types/carrier";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./carrier-columns";
import { CarrierPanel } from "./carrier-panel";

export default function CarrierTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: Carrier[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.carrierService.bulkUpdateStatus({
          carrierIds: ids as string[],
          status: status as Carrier["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["carrier-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<Carrier>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: "Update Status",
        loadingLabel: "Updating...",
        icon: CircleCheckIcon,
        options: carrierStatusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate],
  );

  return (
    <DataTable<Carrier>
      name="Carrier"
      queryKey="carrier-list"
      resource={Resource.Carrier}
      columns={columns}
      dockActions={dockActions}
      TablePanel={CarrierPanel}
      enableRowSelection
      graphql={carrierTableGraphQLConfig}
    />
  );
}
