import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import {
  recurringEarningTableGraphQLConfig,
  updateRecurringEarning,
  type RecurringEarningRow,
} from "@/lib/graphql/driver-settlement";
import type { RecurringEarningStatus } from "@trenova/shared/types/driver-pay";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { earningStatusInput, getColumns } from "./earning-columns";
import { EarningPanel } from "./earning-panel";
import { translate } from "@trenova/shared/i18n/runtime";

export default function EarningsTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: RecurringEarningRow[], status: string) => {
      const eligible = rows.filter((row) => row.status !== "Completed" && row.status !== status);
      if (eligible.length === 0) {
        toast.info(t("No selected earnings can move to that status."));
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        updateRecurringEarning(earningStatusInput(row, status as RecurringEarningStatus)),
      );
      notifyBulkOutcome(
        outcome,
        status === "Paused"
          ? {
              succeeded: (count) =>
                translate("{0, plural, one {# earning paused} other {# earnings paused}}", count),
              partial: (succeeded, failed) =>
                translate(
                  "{0, plural, one {# earning paused} other {# earnings paused}}, {1} failed",
                  succeeded,
                  failed,
                ),
              allFailed: (failed) =>
                translate(
                  "{0, plural, one {The selected earning failed} other {All # selected earnings failed}}",
                  failed,
                ),
            }
          : {
              succeeded: (count) =>
                translate("{0, plural, one {# earning resumed} other {# earnings resumed}}", count),
              partial: (succeeded, failed) =>
                translate(
                  "{0, plural, one {# earning resumed} other {# earnings resumed}}, {1} failed",
                  succeeded,
                  failed,
                ),
              allFailed: (failed) =>
                translate(
                  "{0, plural, one {The selected earning failed} other {All # selected earnings failed}}",
                  failed,
                ),
            },
      );
      await queryClient.invalidateQueries({ queryKey: ["recurring-earning-list"] });
    },
    [queryClient, t],
  );

  const dockActions = useMemo<DockAction<RecurringEarningRow>[]>(
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
            description: t("Future settlements include the earning again."),
          },
          {
            value: "Paused",
            label: t("Pause"),
            color: "var(--warning)",
            description: t("Future settlements skip the earning; history is kept."),
          },
        ],
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  return (
    <DataTable<RecurringEarningRow>
      name="Recurring Earning"
      emptyTitle={t("No recurring earnings yet")}
      queryKey="recurring-earning-list"
      graphql={recurringEarningTableGraphQLConfig}
      resource={Resource.RecurringEarning}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={EarningPanel}
    />
  );
}
