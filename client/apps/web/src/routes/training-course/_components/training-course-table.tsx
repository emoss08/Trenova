import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import {
  archiveTrainingCourse,
  restoreTrainingCourse,
  TRAINING_COURSE_LIST_KEY,
  TRAINING_COURSES_KEY,
  trainingCourseTableGraphQLConfig,
  type TrainingCourseRow,
} from "@/lib/graphql/worker-training";
import type { DockAction, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveIcon, ArchiveRestoreIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./training-course-columns";
import { TrainingCoursePanel } from "./training-course-panel";

export default function TrainingCourseTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const { allowed: canArchive } = usePermission(Resource.TrainingCourse, Operation.Archive);
  const { allowed: canRestore } = usePermission(Resource.TrainingCourse, Operation.Restore);

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [TRAINING_COURSE_LIST_KEY], refetchType: "all" }),
      queryClient.invalidateQueries({ queryKey: [TRAINING_COURSES_KEY], refetchType: "all" }),
    ]);
  }, [queryClient]);

  const archiveRows = useCallback(
    async (rows: readonly TrainingCourseRow[]) => {
      const eligible = rows.filter((row) => row.status === "Active" && row.openRecordCount === 0);
      if (eligible.length === 0) {
        toast.info(t("Only active courses with no open assignments can be deactivated."));
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        archiveTrainingCourse(row.id, row.version),
      );
      notifyBulkOutcome(outcome, {
        entity: "course",
        verbPast: "Deactivated",
        skipped: rows.length - eligible.length,
      });
      await invalidate();
    },
    [invalidate, t],
  );

  const restoreRows = useCallback(
    async (rows: readonly TrainingCourseRow[]) => {
      const eligible = rows.filter((row) => row.status === "Inactive");
      if (eligible.length === 0) {
        toast.info(t("Only inactive courses can be restored."));
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        restoreTrainingCourse(row.id, row.version),
      );
      notifyBulkOutcome(outcome, {
        entity: "course",
        verbPast: "Restored",
        skipped: rows.length - eligible.length,
      });
      await invalidate();
    },
    [invalidate, t],
  );

  const dockActions = useMemo<DockAction<TrainingCourseRow>[]>(() => {
    const actions: DockAction<TrainingCourseRow>[] = [];
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

  const contextMenuActions = useMemo<RowAction<TrainingCourseRow>[]>(() => {
    const actions: RowAction<TrainingCourseRow>[] = [];
    if (canArchive) {
      actions.push({
        id: "archive",
        label: "Deactivate",
        icon: ArchiveIcon,
        variant: "destructive",
        hidden: (row) => row.original.status !== "Active",
        disabled: (row) => row.original.openRecordCount > 0,
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
    <DataTable<TrainingCourseRow>
      name="Training Course"
      queryKey={TRAINING_COURSE_LIST_KEY}
      graphql={trainingCourseTableGraphQLConfig}
      resource={Resource.TrainingCourse}
      columns={columns}
      dockActions={dockActions}
      contextMenuActions={contextMenuActions}
      enableRowSelection={dockActions.length > 0}
      TablePanel={TrainingCoursePanel}
    />
  );
}
