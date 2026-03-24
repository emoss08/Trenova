import { DataTable } from "@/components/data-table/data-table";
import {
  driverTypeChoices,
  statusChoices,
  workerTypeChoices,
} from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import type { Worker } from "@/types/worker";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon, TruckIcon, UserIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./worker-columns";
import { WorkerPanel } from "./worker-panel";



export default function WorkerTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: Worker[], status: string) => {
      const updatePromises = rows.map((r) =>
        apiService.workerService.patch(r.id, {
          status: status as Worker["status"],
        }),
      );

      toast.promise(Promise.all(updatePromises), {
        loading: "Updating status...",
        success: `Updated ${rows.length} worker(s) successfully`,
        error: "Failed to update status",
        finally: async () => {
          await queryClient.invalidateQueries({
            queryKey: ["worker-list"],
            refetchType: "all",
          });
        },
      });
    },
    [queryClient],
  );

  const handleBulkTypeUpdate = useCallback(
    async (rows: Worker[], type: string) => {
      const updatePromises = rows.map((r) =>
        apiService.workerService.patch(r.id, {
          type: type as Worker["type"],
        }),
      );

      toast.promise(Promise.all(updatePromises), {
        loading: "Updating worker type...",
        success: `Updated ${rows.length} worker(s) successfully`,
        error: "Failed to update worker type",
        finally: async () => {
          await queryClient.invalidateQueries({
            queryKey: ["worker-list"],
            refetchType: "all",
          });
        },
      });
    },
    [queryClient],
  );

  const handleBulkDriverTypeUpdate = useCallback(
    async (rows: Worker[], driverType: string) => {
      const updatePromises = rows.map((r) =>
        apiService.workerService.patch(r.id, {
          driverType: driverType as Worker["driverType"],
        }),
      );

      toast.promise(Promise.all(updatePromises), {
        loading: "Updating driver type...",
        success: `Updated ${rows.length} worker(s) successfully`,
        error: "Failed to update driver type",
        finally: async () => {
          await queryClient.invalidateQueries({
            queryKey: ["worker-list"],
            refetchType: "all",
          });
        },
      });
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<Worker>[]>(
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
      {
        id: "type-update",
        type: "select",
        label: "Update Type",
        loadingLabel: "Updating...",
        icon: UserIcon,
        options: workerTypeChoices,
        onSelect: handleBulkTypeUpdate,
        clearSelectionOnSuccess: true,
      },
      {
        id: "driver-type-update",
        type: "select",
        label: "Update Driver Type",
        loadingLabel: "Updating...",
        icon: TruckIcon,
        options: driverTypeChoices,
        onSelect: handleBulkDriverTypeUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, handleBulkTypeUpdate, handleBulkDriverTypeUpdate],
  );

  return (
    <DataTable<Worker>
      name="Worker"
      link="/workers/"
      queryKey="worker-list"
      exportModelName="worker"
      resource={Resource.Worker}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={WorkerPanel}
      extraSearchParams={{
        includeFleetDetails: true,
        includeStateDetails: true,
        includeProfileDetails: true,
      }}
    />
  );
}
