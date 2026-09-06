import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import {
  archiveWorkerChecklistTemplate,
  restoreWorkerChecklistTemplate,
  WORKER_CHECKLIST_TEMPLATE_LIST_KEY,
  WORKER_CHECKLIST_TEMPLATES_KEY,
  workerChecklistTemplateTableGraphQLConfig,
  type WorkerChecklistTemplateRow,
} from "@/lib/graphql/worker-checklist";
import type { DockAction, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveIcon, ArchiveRestoreIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./checklist-template-columns";
import { ChecklistTemplatePanel } from "./checklist-template-panel";

export default function ChecklistTemplateTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const { allowed: canArchive } = usePermission(
    Resource.WorkerChecklistTemplate,
    Operation.Archive,
  );
  const { allowed: canRestore } = usePermission(
    Resource.WorkerChecklistTemplate,
    Operation.Restore,
  );

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: [WORKER_CHECKLIST_TEMPLATE_LIST_KEY],
        refetchType: "all",
      }),
      queryClient.invalidateQueries({
        queryKey: [WORKER_CHECKLIST_TEMPLATES_KEY],
        refetchType: "all",
      }),
    ]);
  }, [queryClient]);

  const archiveRows = useCallback(
    async (rows: readonly WorkerChecklistTemplateRow[]) => {
      const eligible = rows.filter((row) => row.status === "Active");
      if (eligible.length === 0) {
        toast.info("Only active templates can be deactivated.");
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        archiveWorkerChecklistTemplate(row.id, row.version),
      );
      notifyBulkOutcome(outcome, {
        entity: "template",
        verbPast: "Deactivated",
        skipped: rows.length - eligible.length,
      });
      await invalidate();
    },
    [invalidate],
  );

  const restoreRows = useCallback(
    async (rows: readonly WorkerChecklistTemplateRow[]) => {
      const eligible = rows.filter((row) => row.status === "Inactive");
      if (eligible.length === 0) {
        toast.info("Only inactive templates can be restored.");
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        restoreWorkerChecklistTemplate(row.id, row.version),
      );
      notifyBulkOutcome(outcome, {
        entity: "template",
        verbPast: "Restored",
        skipped: rows.length - eligible.length,
      });
      await invalidate();
    },
    [invalidate],
  );

  const dockActions = useMemo<DockAction<WorkerChecklistTemplateRow>[]>(() => {
    const actions: DockAction<WorkerChecklistTemplateRow>[] = [];
    if (canArchive) {
      actions.push({
        id: "archive",
        label: "Deactivate",
        loadingLabel: "Deactivating...",
        icon: ArchiveIcon,
        variant: "destructive",
        onClick: archiveRows,
        clearSelectionOnSuccess: true,
      });
    }
    if (canRestore) {
      actions.push({
        id: "restore",
        label: "Restore",
        loadingLabel: "Restoring...",
        icon: ArchiveRestoreIcon,
        onClick: restoreRows,
        clearSelectionOnSuccess: true,
      });
    }
    return actions;
  }, [archiveRows, canArchive, canRestore, restoreRows]);

  const contextMenuActions = useMemo<RowAction<WorkerChecklistTemplateRow>[]>(() => {
    const actions: RowAction<WorkerChecklistTemplateRow>[] = [];
    if (canArchive) {
      actions.push({
        id: "archive",
        label: "Deactivate",
        icon: ArchiveIcon,
        variant: "destructive",
        hidden: (row) => row.original.status !== "Active",
        onClick: (row) => void archiveRows([row.original]),
      });
    }
    if (canRestore) {
      actions.push({
        id: "restore",
        label: "Restore",
        icon: ArchiveRestoreIcon,
        hidden: (row) => row.original.status !== "Inactive",
        onClick: (row) => void restoreRows([row.original]),
      });
    }
    return actions;
  }, [archiveRows, canArchive, canRestore, restoreRows]);

  return (
    <DataTable<WorkerChecklistTemplateRow>
      name="Checklist Template"
      queryKey={WORKER_CHECKLIST_TEMPLATE_LIST_KEY}
      graphql={workerChecklistTemplateTableGraphQLConfig}
      resource={Resource.WorkerChecklistTemplate}
      columns={columns}
      dockActions={dockActions}
      contextMenuActions={contextMenuActions}
      enableRowSelection={dockActions.length > 0}
      TablePanel={ChecklistTemplatePanel}
    />
  );
}
