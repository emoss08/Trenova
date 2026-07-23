import { DataTable } from "@/components/data-table/data-table";
import { runBulkAction } from "@/lib/bulk-run";
import {
  recurringDeductionTableGraphQLConfig,
  updateRecurringDeduction,
  type RecurringDeductionRow,
} from "@/lib/graphql/driver-settlement";
import type { RecurringDeductionStatus } from "@trenova/shared/types/driver-pay";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { deductionStatusInput, getColumns } from "./deduction-columns";
import { DeductionPanel } from "./deduction-panel";

export default function DeductionsTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: RecurringDeductionRow[], status: string) => {
      const eligible = rows.filter((row) => row.status !== "Completed" && row.status !== status);
      if (eligible.length === 0) {
        toast.info("No selected deductions can move to that status.");
        return;
      }
      await runBulkAction(
        eligible,
        (row) =>
          updateRecurringDeduction(deductionStatusInput(row, status as RecurringDeductionStatus)),
        { noun: "deduction", verb: status === "Paused" ? "paused" : "resumed" },
      );
      await queryClient.invalidateQueries({ queryKey: ["recurring-deduction-list"] });
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<RecurringDeductionRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: "Update Status",
        loadingLabel: "Updating...",
        icon: CircleCheckIcon,
        options: [
          {
            value: "Active",
            label: "Resume",
            color: "#15803d",
            description: "Future settlements withhold the deduction again.",
          },
          {
            value: "Paused",
            label: "Pause",
            color: "#d97706",
            description: "Future settlements skip the deduction; history is kept.",
          },
        ],
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate],
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
