import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import {
  archivePerformanceReviewTemplate,
  restorePerformanceReviewTemplate,
  REVIEW_TEMPLATE_LIST_KEY,
  REVIEW_TEMPLATES_KEY,
  reviewTemplateTableGraphQLConfig,
  type ReviewTemplateRow,
} from "@/lib/graphql/performance-review";
import type { DockAction, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveIcon, ArchiveRestoreIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./review-template-columns";
import { ReviewTemplatePanel } from "./review-template-panel";

export default function ReviewTemplateTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const { allowed: canArchive } = usePermission(
    Resource.PerformanceReviewTemplate,
    Operation.Archive,
  );
  const { allowed: canRestore } = usePermission(
    Resource.PerformanceReviewTemplate,
    Operation.Restore,
  );

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [REVIEW_TEMPLATE_LIST_KEY], refetchType: "all" }),
      queryClient.invalidateQueries({ queryKey: [REVIEW_TEMPLATES_KEY], refetchType: "all" }),
    ]);
  }, [queryClient]);

  const archiveRows = useCallback(
    async (rows: readonly ReviewTemplateRow[]) => {
      const eligible = rows.filter((row) => row.status === "Active" && row.openReviewCount === 0);
      if (eligible.length === 0) {
        toast.info(t("Only active templates with no open reviews can be deactivated."));
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        archivePerformanceReviewTemplate(row.id, row.version),
      );
      notifyBulkOutcome(outcome, {
        entity: "template",
        verbPast: "Deactivated",
        skipped: rows.length - eligible.length,
      });
      await invalidate();
    },
    [invalidate, t],
  );

  const restoreRows = useCallback(
    async (rows: readonly ReviewTemplateRow[]) => {
      const eligible = rows.filter((row) => row.status === "Inactive");
      if (eligible.length === 0) {
        toast.info(t("Only inactive templates can be restored."));
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        restorePerformanceReviewTemplate(row.id, row.version),
      );
      notifyBulkOutcome(outcome, {
        entity: "template",
        verbPast: "Restored",
        skipped: rows.length - eligible.length,
      });
      await invalidate();
    },
    [invalidate, t],
  );

  const dockActions = useMemo<DockAction<ReviewTemplateRow>[]>(() => {
    const actions: DockAction<ReviewTemplateRow>[] = [];
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

  const contextMenuActions = useMemo<RowAction<ReviewTemplateRow>[]>(() => {
    const actions: RowAction<ReviewTemplateRow>[] = [];
    if (canArchive) {
      actions.push({
        id: "archive",
        label: "Deactivate",
        icon: ArchiveIcon,
        variant: "destructive",
        hidden: (row) => row.original.status !== "Active",
        disabled: (row) => row.original.openReviewCount > 0,
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
    <DataTable<ReviewTemplateRow>
      name="Review Template"
      queryKey={REVIEW_TEMPLATE_LIST_KEY}
      graphql={reviewTemplateTableGraphQLConfig}
      resource={Resource.PerformanceReviewTemplate}
      columns={columns}
      dockActions={dockActions}
      contextMenuActions={contextMenuActions}
      enableRowSelection={dockActions.length > 0}
      TablePanel={ReviewTemplatePanel}
    />
  );
}
