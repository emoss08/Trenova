import { DataTable } from "@/components/data-table/data-table";
import { fleetCodeTableGraphQLConfig, type FleetCodeRow } from "@/lib/graphql/fleet-code-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@trenova/shared/types/data-table";
import type { FleetCode } from "@trenova/shared/types/fleet-code";
import { Resource } from "@trenova/shared/types/permission";
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
    async (rows: FleetCodeRow[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.fleetCodeService.bulkUpdateStatus({
          fleetCodeIds: ids,
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

  const dockActions = useMemo<DockAction<FleetCodeRow>[]>(
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
    <DataTable<FleetCodeRow>
      name="Fleet Code"
      queryKey="fleet-code-list"
      graphql={fleetCodeTableGraphQLConfig}
      resource={Resource.FleetCode}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={FleetCodePanel}
    />
  );
}
