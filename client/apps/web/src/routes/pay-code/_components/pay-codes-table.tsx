import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import {
  payCodeTableGraphQLConfig,
  updatePayCode,
  type PayCodeRow,
} from "@/lib/graphql/driver-settlement";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { selectOptionsQueryFilter } from "@/lib/select-options-cache";
import { toast } from "sonner";
import { getColumns, payCodeStatusInput } from "./pay-code-columns";
import { PayCodePanel } from "./pay-code-panel";
import { translate } from "@trenova/shared/i18n/runtime";

export default function PayCodesTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: PayCodeRow[], status: string) => {
      const eligible = rows.filter((row) => row.status !== status);
      if (eligible.length === 0) {
        toast.info(t("Every selected pay code already has that status."));
        return;
      }
      const outcome = await settleAll(eligible, (row) =>
        updatePayCode(payCodeStatusInput(row, status as "Active" | "Inactive")),
      );
      notifyBulkOutcome(
        outcome,
        status === "Active"
          ? {
              succeeded: (count) =>
                translate(
                  "{0, plural, one {# pay code activated} other {# pay codes activated}}",
                  count,
                ),
              partial: (succeeded, failed) =>
                translate(
                  "{0, plural, one {# pay code activated} other {# pay codes activated}}, {1} failed",
                  succeeded,
                  failed,
                ),
              allFailed: (failed) =>
                translate(
                  "{0, plural, one {The selected pay code failed} other {All # selected pay codes failed}}",
                  failed,
                ),
            }
          : {
              succeeded: (count) =>
                translate(
                  "{0, plural, one {# pay code deactivated} other {# pay codes deactivated}}",
                  count,
                ),
              partial: (succeeded, failed) =>
                translate(
                  "{0, plural, one {# pay code deactivated} other {# pay codes deactivated}}, {1} failed",
                  succeeded,
                  failed,
                ),
              allFailed: (failed) =>
                translate(
                  "{0, plural, one {The selected pay code failed} other {All # selected pay codes failed}}",
                  failed,
                ),
            },
      );
      await queryClient.invalidateQueries({ queryKey: ["pay-code-list"] });
      await queryClient.invalidateQueries(selectOptionsQueryFilter("PAY_CODE"));
    },
    [queryClient, t],
  );

  const dockActions = useMemo<DockAction<PayCodeRow>[]>(
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
            label: t("Activate"),
            color: "var(--success)",
            description: t("Codes appear in dropdowns and can be used on new records."),
          },
          {
            value: "Inactive",
            label: t("Deactivate"),
            color: "var(--danger)",
            description: t("Codes stay on historical records but leave new-entry dropdowns."),
          },
        ],
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  return (
    <DataTable<PayCodeRow>
      name="Pay Code"
      emptyTitle={t("No pay codes yet")}
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
