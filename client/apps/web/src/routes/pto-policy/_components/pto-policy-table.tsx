import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import { selectOptionsQueryFilter } from "@/lib/select-options-cache";
import {
  archivePtoPolicy,
  PTO_POLICY_LIST_KEY,
  ptoPolicyTableGraphQLConfig,
  restorePtoPolicy,
  type PTOPolicyRow,
} from "@/lib/graphql/pto-policy";
import type { DockAction, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveIcon, ArchiveRestoreIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./pto-policy-columns";
import { PTOPolicyPanel } from "./pto-policy-panel";

export default function PTOPolicyTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);
  const { allowed: canArchive } = usePermission(Resource.PTOPolicy, Operation.Archive);
  const { allowed: canRestore } = usePermission(Resource.PTOPolicy, Operation.Restore);

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [PTO_POLICY_LIST_KEY], refetchType: "all" }),
      queryClient.invalidateQueries({
        ...selectOptionsQueryFilter("PTO_POLICY"),
        refetchType: "all",
      }),
    ]);
  }, [queryClient]);

  const archiveRows = useCallback(
    async (rows: readonly PTOPolicyRow[]) => {
      const eligible = rows.filter((row) => row.status === "Active");
      if (eligible.length === 0) {
        toast.info(t("Only active policies can be archived."));
        return;
      }
      const outcome = await settleAll(eligible, (row) => archivePtoPolicy(row.id, row.version));
      notifyBulkOutcome(outcome, {
        entity: "policy",
        verbPast: t("Archived"),
        skipped: rows.length - eligible.length,
      });
      await invalidate();
    },
    [invalidate, t],
  );

  const restoreRows = useCallback(
    async (rows: readonly PTOPolicyRow[]) => {
      const eligible = rows.filter((row) => row.status === "Inactive");
      if (eligible.length === 0) {
        toast.info(t("Only inactive policies can be restored."));
        return;
      }
      const outcome = await settleAll(eligible, (row) => restorePtoPolicy(row.id, row.version));
      notifyBulkOutcome(outcome, {
        entity: "policy",
        verbPast: t("Restored"),
        skipped: rows.length - eligible.length,
      });
      await invalidate();
    },
    [invalidate, t],
  );

  const dockActions = useMemo<DockAction<PTOPolicyRow>[]>(() => {
    const actions: DockAction<PTOPolicyRow>[] = [];
    if (canArchive) {
      actions.push({
        id: "archive",
        label: t("Archive"),
        loadingLabel: t("Archiving..."),
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

  const contextMenuActions = useMemo<RowAction<PTOPolicyRow>[]>(() => {
    const actions: RowAction<PTOPolicyRow>[] = [];
    if (canArchive) {
      actions.push({
        id: "archive",
        label: t("Archive"),
        icon: ArchiveIcon,
        variant: "destructive",
        hidden: (row) => row.original.status !== "Active",
        disabled: (row) => row.original.openAssignmentCount > 0,
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
    <DataTable<PTOPolicyRow>
      name="PTO Policy"
      queryKey={PTO_POLICY_LIST_KEY}
      graphql={ptoPolicyTableGraphQLConfig}
      resource={Resource.PTOPolicy}
      columns={columns}
      dockActions={dockActions}
      contextMenuActions={contextMenuActions}
      enableRowSelection={dockActions.length > 0}
      TablePanel={PTOPolicyPanel}
    />
  );
}
