import { DataTable } from "@/components/data-table/data-table";
import { runBulkAction } from "@/lib/bulk-run";
import {
  payCodeTableGraphQLConfig,
  updatePayCode,
  type PayCodeRow,
} from "@/lib/graphql/driver-settlement";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns, payCodeStatusInput } from "./pay-code-columns";
import { PayCodePanel } from "./pay-code-panel";

export default function PayCodesTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: PayCodeRow[], status: string) => {
      const eligible = rows.filter((row) => row.status !== status);
      if (eligible.length === 0) {
        toast.info("Every selected pay code already has that status.");
        return;
      }
      await runBulkAction(
        eligible,
        (row) => updatePayCode(payCodeStatusInput(row, status as "Active" | "Inactive")),
        { noun: "pay code", verb: status === "Active" ? "activated" : "deactivated" },
      );
      await queryClient.invalidateQueries({ queryKey: ["pay-code-list"] });
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<PayCodeRow>[]>(
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
            label: "Activate",
            color: "#15803d",
            description: "Codes appear in dropdowns and can be used on new records.",
          },
          {
            value: "Inactive",
            label: "Deactivate",
            color: "#dc2626",
            description: "Codes stay on historical records but leave new-entry dropdowns.",
          },
        ],
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate],
  );

  return (
    <DataTable<PayCodeRow>
      name="Pay Code"
      queryKey="pay-code-list"
      graphql={payCodeTableGraphQLConfig}
      resource={Resource.PayCode}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={PayCodePanel}
    />
  );
}
