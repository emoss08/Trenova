import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { locationTableGraphQLConfig, type LocationRow } from "@/lib/graphql/location-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@trenova/shared/types/data-table";
import type { Location } from "@trenova/shared/types/location";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./location-columns";
import { LocationPanel } from "./location-panel";

export default function LocationTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: LocationRow[], status: string) => {
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

  const dockActions = useMemo<DockAction<LocationRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update Status"),
        loadingLabel: t("Updating..."),
        icon: CircleCheckIcon,
        options: statusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  return (
    <DataTable<LocationRow>
      name="Location"
      queryKey="location-list"
      graphql={locationTableGraphQLConfig}
      resource={Resource.Location}
      columns={columns}
      dockActions={dockActions}
      TablePanel={LocationPanel}
      enableRowSelection
    />
  );
}
