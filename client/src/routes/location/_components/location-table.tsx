import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@/types/data-table";
import type { Location } from "@/types/location";
import { Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./location-columns";
import { LocationPanel } from "./location-panel";

export default function LocationTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: Location[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.locationService.bulkUpdateStatus({
          locationIds: ids as string[],
          status: status as Location["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["location-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<Location>[]>(
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
    <DataTable<Location>
      name="Location"
      link="/locations/"
      queryKey="location-list"
      exportModelName="location"
      resource={Resource.Location}
      columns={columns}
      dockActions={dockActions}
      TablePanel={LocationPanel}
      enableRowSelection
    />
  );
}
