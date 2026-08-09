import { BulkMarkPaidDialog } from "@/components/settlements/bulk-mark-paid-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { runBulkAction } from "@/lib/bulk-run";
import {
  carrierLifecycleVerbs,
  carrierSettlementLifecycleChoices,
  eligibleCarrierSettlements,
  type CarrierSettlementLifecycleAction,
} from "@/lib/carrier-settlement-lifecycle";
import {
  approveCarrierSettlement,
  carrierSettlementTableGraphQLConfig,
  markCarrierSettlementPaid,
  postCarrierSettlement,
  submitCarrierSettlement,
  type CarrierSettlementRow,
} from "@/lib/graphql/carrier-settlement";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon, CircleDollarSignIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./settlement-columns";
import { CarrierSettlementPanel } from "./settlement-panel";

export default function CarrierSettlementsTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const [payRows, setPayRows] = useState<CarrierSettlementRow[]>([]);
  const [payPending, setPayPending] = useState(false);

  const invalidate = useCallback(async () => {
    for (const key of [
      "carrier-settlement-list",
      "carrier-settlement-detail",
      "carrier-settlement-workspace-summary",
      "carrier-settlement-workspace-settlements",
    ]) {
      await queryClient.invalidateQueries({ queryKey: [key] });
    }
  }, [queryClient]);

  const runLifecycleAction = useCallback(
    async (
      rows: CarrierSettlementRow[],
      action: CarrierSettlementLifecycleAction,
      paymentMethod?: string,
      paymentReference?: string,
    ) => {
      const eligible = eligibleCarrierSettlements(rows, action);
      if (eligible.length === 0) {
        toast.info("None of the selected settlements are in an eligible status for that action.");
        return;
      }
      await runBulkAction(
        eligible,
        (row) => {
          switch (action) {
            case "Submit":
              return submitCarrierSettlement({ settlementId: row.id });
            case "Approve":
              return approveCarrierSettlement({ settlementId: row.id });
            case "Post":
              return postCarrierSettlement({ settlementId: row.id });
            case "MarkPaid":
              return markCarrierSettlementPaid({
                settlementId: row.id,
                paymentMethod: paymentMethod ?? "Check",
                paymentReference: paymentReference || undefined,
              });
          }
        },
        { noun: "settlement", verb: carrierLifecycleVerbs[action] },
      );
      await invalidate();
    },
    [invalidate],
  );

  const openMarkPaidDialog = useCallback((rows: CarrierSettlementRow[]) => {
    const eligible = eligibleCarrierSettlements(rows, "MarkPaid");
    if (eligible.length === 0) {
      toast.info("Only posted settlements can be marked paid.");
      return;
    }
    setPayRows(eligible);
  }, []);

  const dockActions = useMemo<DockAction<CarrierSettlementRow>[]>(
    () => [
      {
        id: "lifecycle",
        type: "select",
        label: "Lifecycle Action",
        loadingLabel: "Running...",
        icon: CircleCheckIcon,
        options: carrierSettlementLifecycleChoices,
        onSelect: (rows, value) =>
          runLifecycleAction(rows, value as CarrierSettlementLifecycleAction),
        clearSelectionOnSuccess: true,
      },
      {
        id: "mark-paid",
        label: "Mark Paid",
        icon: CircleDollarSignIcon,
        onClick: openMarkPaidDialog,
      },
    ],
    [runLifecycleAction, openMarkPaidDialog],
  );

  return (
    <>
      <DataTable<CarrierSettlementRow>
        name="Carrier Settlement"
        queryKey="carrier-settlement-list"
        graphql={carrierSettlementTableGraphQLConfig}
        resource={Resource.CarrierSettlement}
        columns={columns}
        dockActions={dockActions}
        enableRowSelection
        TablePanel={CarrierSettlementPanel}
        enableCreateAction={false}
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
