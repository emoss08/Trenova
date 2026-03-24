import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@/types/data-table";
import type { FleetCode } from "@/types/fleet-code";
import { Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./fleet-code-columns";
import { FleetCodePanel } from "./fleet-code-panel";

export default function FleetCodeTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: FleetCode[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.fleetCodeService.bulkUpdateStatus({
          fleetCodeIds: ids as string[],
          status: status as FleetCode["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["fleet-code-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<FleetCode>[]>(
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
    <DataTable<FleetCode>
      name="Fleet Code"
      link="/fleet-codes/"
      queryKey="fleet-code-list"
      exportModelName="fleet-code"
      resource={Resource.FleetCode}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={FleetCodePanel}
      extraSearchParams={{
        includeManagerDetails: true,
      }}
    />
  );
}
