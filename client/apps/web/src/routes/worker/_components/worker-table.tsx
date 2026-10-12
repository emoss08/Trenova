import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { driverTypeChoices, statusChoices, workerTypeChoices } from "@/lib/choices";
import { patchWorker } from "@/lib/graphql/worker-mutations";
import { workerTableGraphQLConfigs, type WorkerRow } from "@/lib/graphql/worker-table";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import type { Worker } from "@trenova/shared/types/worker";
import { useQueryClient } from "@tanstack/react-query";
import {
  CheckCircleIcon,
  GraduationHat01Icon,
  Truck01Icon,
  User01Icon,
} from "@trenova/shared/components/icons";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { BulkAssignTrainingDialog } from "./bulk-assign-training-dialog";
import { getColumns } from "./worker-columns";
import { WorkerPanel } from "./worker-panel";

export default function WorkerTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);
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
        loading: t("Updating status..."),
        success: t(
          "{0, plural, one {Updated # worker successfully} other {Updated # workers successfully}}",
          rows.length,
        ),
        error: t("Failed to update status"),
        finally: async () => {
          await queryClient.invalidateQueries({
            queryKey: ["worker-list"],
            refetchType: "all",
          });
        },
      });
    },
    [queryClient, t],
  );

  const handleBulkTypeUpdate = useCallback(
    async (rows: WorkerRow[], type: string) => {
      const updatePromises = rows.map((r) =>
        patchWorker(r.id, {
          type: type as Worker["type"],
        }),
      );

      toast.promise(Promise.all(updatePromises), {
        loading: t("Updating worker type..."),
        success: t(
          "{0, plural, one {Updated # worker successfully} other {Updated # workers successfully}}",
          rows.length,
        ),
        error: t("Failed to update worker type"),
        finally: async () => {
          await queryClient.invalidateQueries({
            queryKey: ["worker-list"],
            refetchType: "all",
          });
        },
      });
    },
    [queryClient, t],
  );

  const handleBulkDriverTypeUpdate = useCallback(
    async (rows: WorkerRow[], driverType: string) => {
      const updatePromises = rows.map((r) =>
        patchWorker(r.id, {
          driverType: driverType as Worker["driverType"],
        }),
      );

      toast.promise(Promise.all(updatePromises), {
        loading: t("Updating driver type..."),
        success: t(
          "{0, plural, one {Updated # worker successfully} other {Updated # workers successfully}}",
          rows.length,
        ),
        error: t("Failed to update driver type"),
        finally: async () => {
          await queryClient.invalidateQueries({
            queryKey: ["worker-list"],
            refetchType: "all",
          });
        },
      });
    },
    [queryClient, t],
  );

  const dockActions = useMemo<DockAction<WorkerRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update status"),
        loadingLabel: t("Updating..."),
        icon: CheckCircleIcon,
        options: statusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
      {
        id: "type-update",
        type: "select",
        label: t("Update type"),
        loadingLabel: t("Updating..."),
        icon: User01Icon,
        options: workerTypeChoices,
        onSelect: handleBulkTypeUpdate,
        clearSelectionOnSuccess: true,
      },
      {
        id: "driver-type-update",
        type: "select",
        label: t("Update driver type"),
        loadingLabel: t("Updating..."),
        icon: Truck01Icon,
        options: driverTypeChoices,
        onSelect: handleBulkDriverTypeUpdate,
        clearSelectionOnSuccess: true,
      },
      {
        id: "assign-training",
        label: t("Assign training"),
        icon: GraduationHat01Icon,
        onClick: (rows) => setTrainingTargets(rows),
      },
    ],
    [handleBulkStatusUpdate, handleBulkTypeUpdate, handleBulkDriverTypeUpdate, t],
  );

  return (
    <>
      <DataTable<WorkerRow>
        name="Worker"
        emptyTitle={t("No workers yet")}
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
