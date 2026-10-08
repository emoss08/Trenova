import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  recurringShipmentTableGraphQLConfig,
  type RecurringShipmentRow,
} from "@/lib/graphql/recurring-shipment-table";
import { apiService } from "@/services/api";
import type { RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { ClockRewindIcon, PauseIcon, ZapIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./recurring-shipment-columns";
import { RecurringShipmentPanel } from "./recurring-shipment-panel";
import { RecurringShipmentRunsDialog } from "./recurring-shipment-runs-dialog";

export default function RecurringShipmentTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);
  const [runsSeries, setRunsSeries] = useState<RecurringShipmentRow | null>(null);
  const [runsOpen, setRunsOpen] = useState(false);

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({
      queryKey: ["recurring-shipment-list"],
      refetchType: "all",
    });
  }, [queryClient]);

  const handleGenerateNow = useCallback(
    (series: RecurringShipmentRow) => {
      toast.promise(apiService.recurringShipmentService.generate(series.id), {
        loading: t("Generating shipment..."),
        success: (result) =>
          result.shipment?.proNumber
            ? t('Shipment {0} generated from "{1}"', result.shipment.proNumber, series.name)
            : t('Occurrence processed for "{0}"', series.name),
        error: t("Failed to generate shipment"),
        finally: invalidate,
      });
    },
    [invalidate, t],
  );

  const handleToggleStatus = useCallback(
    (series: RecurringShipmentRow) => {
      const nextStatus = series.status === "Paused" ? "Active" : "Paused";
      toast.promise(
        apiService.recurringShipmentService.updateStatus(series.id, nextStatus, series.version),
        {
          loading: nextStatus === "Paused" ? t("Pausing series...") : t("Resuming series..."),
          success:
            nextStatus === "Paused"
              ? t('"{0}" paused — no shipments will generate until resumed', series.name)
              : t('"{0}" resumed — the schedule restarts from the next future pickup', series.name),
          error: t("Failed to update series status"),
          finally: invalidate,
        },
      );
    },
    [invalidate, t],
  );

  const rowActions = useMemo<RowAction<RecurringShipmentRow>[]>(
    () => [
      {
        id: "generate-now",
        label: t("Generate now"),
        icon: ZapIcon,
        onClick: (row) => handleGenerateNow(row.original),
        disabled: (row) => row.original.status === "Expired",
      },
      {
        id: "toggle-status",
        label: t("Pause / resume"),
        icon: PauseIcon,
        onClick: (row) => handleToggleStatus(row.original),
        hidden: (row) => row.original.status === "Expired",
      },
      {
        id: "view-runs",
        label: t("View history"),
        icon: ClockRewindIcon,
        onClick: (row) => {
          setRunsSeries(row.original);
          setRunsOpen(true);
        },
      },
    ],
    [handleGenerateNow, handleToggleStatus, t],
  );

  return (
    <>
      <DataTable<RecurringShipmentRow>
        name="Recurring Shipment"
        emptyTitle={t("No recurring shipments yet")}
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
