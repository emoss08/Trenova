import { DataTable } from "@/components/data-table/data-table";
import { driverTypeChoices, statusChoices, workerTypeChoices } from "@/lib/choices";
import { patchWorker } from "@/lib/graphql/worker-mutations";
import { workerTableGraphQLConfigs, type WorkerRow } from "@/lib/graphql/worker-table";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import type { Worker } from "@trenova/shared/types/worker";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon, GraduationCapIcon, TruckIcon, UserIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { BulkAssignTrainingDialog } from "./bulk-assign-training-dialog";
import { getColumns } from "./worker-columns";
import { WorkerPanel } from "./worker-panel";

export default function WorkerTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  // Held here rather than inside the dock so the dialog keeps the selection
  // after the dock closes it.
  const [trainingTargets, setTrainingTargets] = useState<WorkerRow[] | null>(null);

  const handleBulkStatusUpdate = useCallback(
    async (rows: WorkerRow[], status: string) => {
      const updatePromises = rows.map((r) =>
        patchWorker(r.id, {
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
    async (rows: WorkerRow[], type: string) => {
      const updatePromises = rows.map((r) =>
        patchWorker(r.id, {
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
    async (rows: WorkerRow[], driverType: string) => {
      const updatePromises = rows.map((r) =>
        patchWorker(r.id, {
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

  const dockActions = useMemo<DockAction<WorkerRow>[]>(
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
      {
        id: "assign-training",
        label: "Assign Training",
        icon: GraduationCapIcon,
        onClick: (rows) => setTrainingTargets(rows),
      },
    ],
    [handleBulkStatusUpdate, handleBulkTypeUpdate, handleBulkDriverTypeUpdate],
  );

  return (
    <>
      <DataTable<WorkerRow>
        name="Worker"
        queryKey="worker-list"
        resource={Resource.Worker}
        columns={columns}
        graphql={workerTableGraphQLConfigs.worker}
        dockActions={dockActions}
        enableRowSelection
        TablePanel={WorkerPanel}
      />
      <BulkAssignTrainingDialog
        open={trainingTargets !== null}
        onOpenChange={(open) => {
          if (!open) setTrainingTargets(null);
        }}
        workers={trainingTargets ?? []}
      />
    </>
  );
}
