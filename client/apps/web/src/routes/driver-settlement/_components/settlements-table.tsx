import { useT } from "@trenova/shared/i18n/use-t";
import { BulkMarkPaidDialog } from "@/components/settlements/bulk-mark-paid-dialog";
import { DataTable } from "@/components/data-table/data-table";
import {
  bulkActionVerbs,
  eligibleSettlements,
  settlementLifecycleChoices,
} from "@/lib/settlement-lifecycle";
import {
  bulkDriverSettlementAction,
  driverSettlementTableGraphQLConfig,
  type DriverSettlementRow,
} from "@/lib/graphql/driver-settlement";
import type { BulkSettlementActionType } from "@trenova/graphql/generated/graphql";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon, CircleDollarSignIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./settlement-columns";
import { SettlementHistoryEmpty } from "./settlement-history-empty";
import { SettlementPanel } from "./settlement-panel";

export default function SettlementsTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);
  const [payRows, setPayRows] = useState<DriverSettlementRow[]>([]);
  const [payPending, setPayPending] = useState(false);

  const invalidate = useCallback(async () => {
    for (const key of [
      "driver-settlement-list",
      "driver-settlement-detail",
      "settlement-workspace-summary",
      "settlement-workspace-settlements",
    ]) {
      await queryClient.invalidateQueries({ queryKey: [key] });
    }
  }, [queryClient]);

  const runLifecycleAction = useCallback(
    async (
      rows: DriverSettlementRow[],
      action: BulkSettlementActionType,
      paymentMethod?: string,
      paymentReference?: string,
    ) => {
      const eligible = eligibleSettlements(rows, action);
      if (eligible.length === 0) {
        toast.info(
          t("None of the selected settlements are in an eligible status for that action."),
        );
        return;
      }
      const result = await bulkDriverSettlementAction({
        settlementIds: eligible.map((row) => row.id),
        action,
        paymentMethod,
        paymentReference,
      });
      const verb = bulkActionVerbs[action];
      if (result.failureCount === 0) {
        toast.success(
          `${result.successCount} settlement${result.successCount === 1 ? "" : "s"} ${verb}`,
        );
      } else {
        const firstError = result.results.find((entry) => !entry.success)?.error;
        toast.warning(
          `${result.successCount} ${verb}, ${result.failureCount} failed${firstError ? ` — ${firstError}` : ""}`,
        );
      }
      await invalidate();
    },
    [invalidate, t],
  );

  const openMarkPaidDialog = useCallback(
    (rows: DriverSettlementRow[]) => {
      const eligible = eligibleSettlements(rows, "MarkPaid");
      if (eligible.length === 0) {
        toast.info(t("Only posted settlements can be marked paid."));
        return;
      }
      setPayRows(eligible);
    },
    [t],
  );

  const dockActions = useMemo<DockAction<DriverSettlementRow>[]>(
    () => [
      {
        id: "lifecycle",
        type: "select",
        label: t("Lifecycle Action"),
        loadingLabel: t("Running..."),
        icon: CircleCheckIcon,
        options: settlementLifecycleChoices,
        onSelect: (rows, value) => runLifecycleAction(rows, value as BulkSettlementActionType),
        clearSelectionOnSuccess: true,
      },
      {
        id: "mark-paid",
        label: t("Mark Paid"),
        icon: CircleDollarSignIcon,
        onClick: openMarkPaidDialog,
      },
    ],
    [runLifecycleAction, openMarkPaidDialog, t],
  );

  return (
    <>
      <DataTable<DriverSettlementRow>
        name="Driver Settlement"
        queryKey="driver-settlement-list"
        graphql={driverSettlementTableGraphQLConfig}
        resource={Resource.DriverSettlement}
        columns={columns}
        dockActions={dockActions}
        enableRowSelection
        TablePanel={SettlementPanel}
        enableCreateAction={false}
        renderEmptyState={(state) => <SettlementHistoryEmpty {...state} />}
      />
      <BulkMarkPaidDialog
        open={payRows.length > 0}
        count={payRows.length}
        pending={payPending}
        onOpenChange={(open) => !open && setPayRows([])}
        onConfirm={(paymentMethod, paymentReference) => {
          setPayPending(true);
          void runLifecycleAction(payRows, "MarkPaid", paymentMethod, paymentReference).finally(
            () => {
              setPayPending(false);
              setPayRows([]);
            },
          );
        }}
      />
    </>
  );
}
