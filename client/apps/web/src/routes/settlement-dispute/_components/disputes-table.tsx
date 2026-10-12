import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  settlementDisputeTableGraphQLConfig,
  startSettlementDisputeReview,
  type SettlementDisputeRow,
} from "@trenova/shared/lib/graphql/driver-portal";
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { EyeIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./dispute-columns";
import { DisputePanel } from "./dispute-panel";
import { translate } from "@trenova/shared/i18n/runtime";

export default function DisputesTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleStartReview = useCallback(
    async (rows: SettlementDisputeRow[]) => {
      const eligible = rows.filter((row) => row.status === "Open");
      if (eligible.length === 0) {
        toast.info(t("Only open disputes can be moved to review."));
        return;
      }
      notifyBulkOutcome(await settleAll(eligible, (row) => startSettlementDisputeReview(row.id)), {
        succeeded: (count) =>
          translate(
            "{0, plural, one {# dispute moved to review} other {# disputes moved to review}}",
            count,
          ),
        partial: (succeeded, failed) =>
          translate(
            "{0, plural, one {# dispute moved to review} other {# disputes moved to review}}, {1} failed",
            succeeded,
            failed,
          ),
        allFailed: (failed) =>
          translate(
            "{0, plural, one {The selected dispute failed} other {All # selected disputes failed}}",
            failed,
          ),
      });
      await queryClient.invalidateQueries({ queryKey: ["settlement-dispute-list"] });
    },
    [queryClient, t],
  );

  const dockActions = useMemo<DockAction<SettlementDisputeRow>[]>(
    () => [
      {
        id: "start-review",
        label: t("Start review"),
        loadingLabel: t("Updating..."),
        icon: EyeIcon,
        onClick: handleStartReview,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleStartReview, t],
  );

  return (
    <DataTable<SettlementDisputeRow>
      name="Settlement Dispute"
      emptyTitle={t("No settlement disputes yet")}
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
