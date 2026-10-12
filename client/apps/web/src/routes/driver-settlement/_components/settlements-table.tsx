import { type TranslateFn, useT } from "@trenova/shared/i18n/use-t";
import { BulkMarkPaidDialog } from "@/components/settlements/bulk-mark-paid-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { eligibleSettlements, settlementLifecycleChoices } from "@/lib/settlement-lifecycle";
import {
  bulkDriverSettlementAction,
  driverSettlementTableGraphQLConfig,
  type DriverSettlementRow,
} from "@/lib/graphql/driver-settlement";
import type { BulkSettlementActionType } from "@trenova/graphql/generated/graphql";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircleIcon, CurrencyDollarCircleIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./settlement-columns";
import { SettlementHistoryEmpty } from "./settlement-history-empty";
import { SettlementPanel } from "./settlement-panel";

function bulkActionSucceeded(t: TranslateFn, action: BulkSettlementActionType, count: number) {
  switch (action) {
    case "Submit":
      return t("{0, plural, one {# settlement submitted} other {# settlements submitted}}", count);
    case "Approve":
      return t("{0, plural, one {# settlement approved} other {# settlements approved}}", count);
    case "Post":
      return t("{0, plural, one {# settlement posted} other {# settlements posted}}", count);
    case "MarkPaid":
      return t(
        "{0, plural, one {# settlement marked paid} other {# settlements marked paid}}",
        count,
      );
  }
}

function bulkActionPartlyFailed(
  t: TranslateFn,
  action: BulkSettlementActionType,
  succeeded: number,
  failed: number,
) {
  switch (action) {
    case "Submit":
      return t("{0} submitted, {1} failed", succeeded, failed);
    case "Approve":
      return t("{0} approved, {1} failed", succeeded, failed);
    case "Post":
      return t("{0} posted, {1} failed", succeeded, failed);
    case "MarkPaid":
      return t("{0} marked paid, {1} failed", succeeded, failed);
  }
}

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
      if (result.failureCount === 0) {
        toast.success(bulkActionSucceeded(t, action, result.successCount));
      } else {
        const firstError = result.results.find((entry) => !entry.success)?.error;
        const summary = bulkActionPartlyFailed(t, action, result.successCount, result.failureCount);
        toast.warning(firstError ? t("{0} — {1}", summary, firstError) : summary);
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
        label: t("Lifecycle action"),
        loadingLabel: t("Running..."),
        icon: CheckCircleIcon,
        options: settlementLifecycleChoices,
        onSelect: (rows, value) => runLifecycleAction(rows, value as BulkSettlementActionType),
        clearSelectionOnSuccess: true,
      },
      {
        id: "mark-paid",
        label: t("Mark paid"),
        icon: CurrencyDollarCircleIcon,
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
