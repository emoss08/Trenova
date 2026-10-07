import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import {
  recurringDeductionTableGraphQLConfig,
  updateRecurringDeduction,
  type RecurringDeductionRow,
} from "@/lib/graphql/driver-settlement";
import type { RecurringDeductionStatus } from "@trenova/shared/types/driver-pay";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { deductionStatusInput, getColumns } from "./deduction-columns";
import { DeductionPanel } from "./deduction-panel";
import { translate } from "@trenova/shared/i18n/runtime";

export default function DeductionsTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: RecurringDeductionRow[], status: string) => {
      const eligible = rows.filter((row) => row.status !== "Completed" && row.status !== status);
      if (eligible.length === 0) {
        toast.info(t("No selected deductions can move to that status."));
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        updateRecurringDeduction(deductionStatusInput(row, status as RecurringDeductionStatus)),
      );
      notifyBulkOutcome(
        outcome,
        status === "Paused"
          ? {
              succeeded: (count) =>
                translate(
                  "{0, plural, one {# deduction paused} other {# deductions paused}}",
                  count,
                ),
              partial: (succeeded, failed) =>
                translate(
                  "{0, plural, one {# deduction paused} other {# deductions paused}}, {1} failed",
                  succeeded,
                  failed,
                ),
              allFailed: (failed) =>
                translate(
                  "{0, plural, one {The selected deduction failed} other {All # selected deductions failed}}",
                  failed,
                ),
            }
          : {
              succeeded: (count) =>
                translate(
                  "{0, plural, one {# deduction resumed} other {# deductions resumed}}",
                  count,
                ),
              partial: (succeeded, failed) =>
                translate(
                  "{0, plural, one {# deduction resumed} other {# deductions resumed}}, {1} failed",
                  succeeded,
                  failed,
                ),
              allFailed: (failed) =>
                translate(
                  "{0, plural, one {The selected deduction failed} other {All # selected deductions failed}}",
                  failed,
                ),
            },
      );
      await queryClient.invalidateQueries({ queryKey: ["recurring-deduction-list"] });
    },
    [queryClient, t],
  );

  const dockActions = useMemo<DockAction<RecurringDeductionRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update status"),
        loadingLabel: t("Updating..."),
        icon: CheckCircleIcon,
        options: [
          {
            value: "Active",
            label: t("Resume"),
            color: "var(--success)",
            description: t("Future settlements withhold the deduction again."),
          },
          {
            value: "Paused",
            label: t("Pause"),
            color: "var(--warning)",
            description: t("Future settlements skip the deduction; history is kept."),
          },
        ],
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  return (
    <DataTable<RecurringDeductionRow>
      name="Recurring Deduction"
      queryKey="recurring-deduction-list"
      graphql={recurringDeductionTableGraphQLConfig}
      resource={Resource.RecurringDeduction}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={DeductionPanel}
    />
  );
}
