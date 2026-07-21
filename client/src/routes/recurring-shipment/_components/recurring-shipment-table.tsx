import { DataTable } from "@/components/data-table/data-table";
import { recurringShipmentTableGraphQLConfig } from "@/lib/graphql/recurring-shipment-table";
import { apiService } from "@/services/api";
import type { RowAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import type { RecurringShipment } from "@/types/recurring-shipment";
import { useQueryClient } from "@tanstack/react-query";
import { HistoryIcon, PauseIcon, ZapIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./recurring-shipment-columns";
import { RecurringShipmentPanel } from "./recurring-shipment-panel";
import { RecurringShipmentRunsDialog } from "./recurring-shipment-runs-dialog";

export default function RecurringShipmentTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const [runsSeries, setRunsSeries] = useState<RecurringShipment | null>(null);
  const [runsOpen, setRunsOpen] = useState(false);

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({
      queryKey: ["recurring-shipment-list"],
      refetchType: "all",
    });
  }, [queryClient]);

  const handleGenerateNow = useCallback(
    (series: RecurringShipment) => {
      toast.promise(apiService.recurringShipmentService.generate(series.id as string), {
        loading: "Generating shipment...",
        success: (result) =>
          result.shipment?.proNumber
            ? `Shipment ${result.shipment.proNumber} generated from "${series.name}"`
            : `Occurrence processed for "${series.name}"`,
        error: "Failed to generate shipment",
        finally: invalidate,
      });
    },
    [invalidate],
  );

  const handleToggleStatus = useCallback(
    (series: RecurringShipment) => {
      const nextStatus = series.status === "Paused" ? "Active" : "Paused";
      toast.promise(
        apiService.recurringShipmentService.updateStatus(
          series.id as string,
          nextStatus,
          series.version ?? 0,
        ),
        {
          loading: nextStatus === "Paused" ? "Pausing series..." : "Resuming series...",
          success:
            nextStatus === "Paused"
              ? `"${series.name}" paused — no shipments will generate until resumed`
              : `"${series.name}" resumed — the schedule restarts from the next future pickup`,
          error: "Failed to update series status",
          finally: invalidate,
        },
      );
    },
    [invalidate],
  );

  const rowActions = useMemo<RowAction<RecurringShipment>[]>(
    () => [
      {
        id: "generate-now",
        label: "Generate Now",
        icon: ZapIcon,
        onClick: (row) => handleGenerateNow(row.original),
        disabled: (row) => row.original.status === "Expired",
      },
      {
        id: "toggle-status",
        label: "Pause / Resume",
        icon: PauseIcon,
        onClick: (row) => handleToggleStatus(row.original),
        hidden: (row) => row.original.status === "Expired",
      },
      {
        id: "view-runs",
        label: "View History",
        icon: HistoryIcon,
        onClick: (row) => {
          setRunsSeries(row.original);
          setRunsOpen(true);
        },
      },
    ],
    [handleGenerateNow, handleToggleStatus],
  );

  return (
    <>
      <DataTable<RecurringShipment>
        name="Recurring Shipment"
        queryKey="recurring-shipment-list"
        graphql={recurringShipmentTableGraphQLConfig}
        resource={Resource.RecurringShipment}
        columns={columns}
        contextMenuActions={rowActions}
        TablePanel={RecurringShipmentPanel}
      />
      <RecurringShipmentRunsDialog series={runsSeries} open={runsOpen} onOpenChange={setRunsOpen} />
    </>
  );
}
