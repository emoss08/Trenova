import { DataTable } from "@/components/data-table/data-table";
import { runBulkAction } from "@/lib/bulk-run";
import {
  recurringEarningTableGraphQLConfig,
  updateRecurringEarning,
  type RecurringEarningRow,
} from "@/lib/graphql/driver-settlement";
import type { RecurringEarningStatus } from "@/types/driver-pay";
import type { DockAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { earningStatusInput, getColumns } from "./earning-columns";
import { EarningPanel } from "./earning-panel";

export default function EarningsTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: RecurringEarningRow[], status: string) => {
      const eligible = rows.filter((row) => row.status !== "Completed" && row.status !== status);
      if (eligible.length === 0) {
        toast.info("No selected earnings can move to that status.");
        return;
      }
      await runBulkAction(
        eligible,
        (row) => updateRecurringEarning(earningStatusInput(row, status as RecurringEarningStatus)),
        { noun: "earning", verb: status === "Paused" ? "paused" : "resumed" },
      );
      await queryClient.invalidateQueries({ queryKey: ["recurring-earning-list"] });
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<RecurringEarningRow>[]>(
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
            description: "Future settlements include the earning again.",
          },
          {
            value: "Paused",
            label: "Pause",
            color: "#d97706",
            description: "Future settlements skip the earning; history is kept.",
          },
        ],
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate],
  );

  return (
    <DataTable<RecurringEarningRow>
      name="Recurring Earning"
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
