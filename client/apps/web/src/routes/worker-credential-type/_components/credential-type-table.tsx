import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import {
  archiveWorkerCredentialType,
  restoreWorkerCredentialType,
  WORKER_CREDENTIAL_TYPE_LIST_KEY,
  WORKER_CREDENTIAL_TYPES_KEY,
  workerCredentialTypeTableGraphQLConfig,
  type WorkerCredentialTypeRow,
} from "@/lib/graphql/worker-credential";
import type { DockAction, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveIcon, ArchiveRestoreIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./credential-type-columns";
import { CredentialTypePanel } from "./credential-type-panel";

export default function CredentialTypeTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);
  const { allowed: canArchive } = usePermission(Resource.WorkerCredentialType, Operation.Archive);
  const { allowed: canRestore } = usePermission(Resource.WorkerCredentialType, Operation.Restore);

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: [WORKER_CREDENTIAL_TYPE_LIST_KEY],
        refetchType: "all",
      }),
      queryClient.invalidateQueries({
        queryKey: [WORKER_CREDENTIAL_TYPES_KEY],
        refetchType: "all",
      }),
    ]);
  }, [queryClient]);

  const archiveRows = useCallback(
    async (rows: readonly WorkerCredentialTypeRow[]) => {
      const eligible = rows.filter((row) => row.status === "Active" && !row.profileField);
      if (eligible.length === 0) {
        toast.info(t("Only active, non-mirrored credential types can be deactivated."));
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        archiveWorkerCredentialType(row.id, row.version),
      );
      notifyBulkOutcome(outcome, {
        entity: "credential type",
        verbPast: t("Deactivated"),
        skipped: rows.length - eligible.length,
      });
      await invalidate();
    },
    [invalidate, t],
  );

  const restoreRows = useCallback(
    async (rows: readonly WorkerCredentialTypeRow[]) => {
      const eligible = rows.filter((row) => row.status === "Inactive");
      if (eligible.length === 0) {
        toast.info(t("Only inactive credential types can be restored."));
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        restoreWorkerCredentialType(row.id, row.version),
      );
      notifyBulkOutcome(outcome, {
        entity: "credential type",
        verbPast: t("Restored"),
        skipped: rows.length - eligible.length,
      });
      await invalidate();
    },
    [invalidate, t],
  );

  const dockActions = useMemo<DockAction<WorkerCredentialTypeRow>[]>(() => {
    const actions: DockAction<WorkerCredentialTypeRow>[] = [];
    if (canArchive) {
      actions.push({
        id: "archive",
        label: t("Deactivate"),
        loadingLabel: t("Deactivating..."),
        icon: ArchiveIcon,
        variant: "destructive",
        onClick: archiveRows,
        clearSelectionOnSuccess: true,
      });
    }
    if (canRestore) {
      actions.push({
        id: "restore",
        label: t("Restore"),
        loadingLabel: t("Restoring..."),
        icon: ArchiveRestoreIcon,
        onClick: restoreRows,
        clearSelectionOnSuccess: true,
      });
    }
    return actions;
  }, [archiveRows, canArchive, canRestore, restoreRows, t]);

  const contextMenuActions = useMemo<RowAction<WorkerCredentialTypeRow>[]>(() => {
    const actions: RowAction<WorkerCredentialTypeRow>[] = [];
    if (canArchive) {
      actions.push({
        id: "archive",
        label: t("Deactivate"),
        icon: ArchiveIcon,
        variant: "destructive",
        hidden: (row) => row.original.status !== "Active",
        disabled: (row) =>
          Boolean(row.original.profileField) || row.original.activeCredentialCount > 0,
        onClick: (row) => void archiveRows([row.original]),
      });
    }
    if (canRestore) {
      actions.push({
        id: "restore",
        label: t("Restore"),
        icon: ArchiveRestoreIcon,
        hidden: (row) => row.original.status !== "Inactive",
        onClick: (row) => void restoreRows([row.original]),
      });
    }
    return actions;
  }, [archiveRows, canArchive, canRestore, restoreRows, t]);

  return (
    <DataTable<WorkerCredentialTypeRow>
      name="Credential Type"
      queryKey={WORKER_CREDENTIAL_TYPE_LIST_KEY}
      graphql={workerCredentialTypeTableGraphQLConfig}
      resource={Resource.WorkerCredentialType}
      columns={columns}
      dockActions={dockActions}
      contextMenuActions={contextMenuActions}
      enableRowSelection={dockActions.length > 0}
      TablePanel={CredentialTypePanel}
    />
  );
}
