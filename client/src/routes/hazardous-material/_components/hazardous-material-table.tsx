import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@/types/data-table";
import type { HazardousMaterial } from "@/types/hazardous-material";
import { Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./hazardous-material-columns";
import { HazardousMaterialPanel } from "./hazardous-material-panel";

export default function HazardousMaterialTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: HazardousMaterial[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.hazardousMaterialService.bulkUpdateStatus({
          hazardousMaterialIds: ids as string[],
          status: status as HazardousMaterial["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["hazardous-material-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<HazardousMaterial>[]>(
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
    <DataTable<HazardousMaterial>
      name="Hazardous Material"
      link="/hazardous-materials/"
      queryKey="hazardous-material-list"
      exportModelName="hazardous-material"
      resource={Resource.HazardousMaterial}
      columns={columns}
      dockActions={dockActions}
      TablePanel={HazardousMaterialPanel}
      enableRowSelection
    />
  );
}
