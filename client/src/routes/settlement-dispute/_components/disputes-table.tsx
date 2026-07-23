import { DataTable } from "@/components/data-table/data-table";
import {
  settlementDisputeTableGraphQLConfig,
  startSettlementDisputeReview,
  type SettlementDisputeRow,
} from "@/lib/graphql/driver-portal";
import { runBulkAction } from "@/lib/bulk-run";
import type { DockAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { EyeIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./dispute-columns";
import { DisputePanel } from "./dispute-panel";

export default function DisputesTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleStartReview = useCallback(
    async (rows: SettlementDisputeRow[]) => {
      const eligible = rows.filter((row) => row.status === "Open");
      if (eligible.length === 0) {
        toast.info("Only open disputes can be moved to review.");
        return;
      }
      await runBulkAction(eligible, (row) => startSettlementDisputeReview(row.id), {
        noun: "dispute",
        verb: "moved to review",
      });
      await queryClient.invalidateQueries({ queryKey: ["settlement-dispute-list"] });
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<SettlementDisputeRow>[]>(
    () => [
      {
        id: "start-review",
        label: "Start Review",
        loadingLabel: "Updating...",
        icon: EyeIcon,
        onClick: handleStartReview,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleStartReview],
  );

  return (
    <DataTable<SettlementDisputeRow>
      name="Settlement Dispute"
      queryKey="settlement-dispute-list"
      graphql={settlementDisputeTableGraphQLConfig}
      resource={Resource.SettlementDispute}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={DisputePanel}
      enableCreateAction={false}
    />
  );
}
