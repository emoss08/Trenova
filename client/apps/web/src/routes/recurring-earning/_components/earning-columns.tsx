import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { recurringEarningStatusChoices } from "@/lib/choices";
import { updateRecurringEarning, type RecurringEarningRow } from "@/lib/graphql/driver-settlement";
import type { RecurringEarningStatus } from "@trenova/shared/types/driver-pay";
import { useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { toast } from "sonner";

export function earningStatusInput(row: RecurringEarningRow, status: RecurringEarningStatus) {
  return {
    id: row.id,
    version: row.version,
    workerId: row.workerId,
    payCodeId: row.payCodeId,
    status,
    frequency: row.frequency,
    description: row.description,
    amountMinor: row.amountMinor,
    totalCapMinor: row.totalCapMinor ?? undefined,
    startDate: row.startDate,
    endDate: row.endDate ?? undefined,
  };
}

const earningStatusVariants = {
  Active: "active",
  Paused: "warning",
  Completed: "secondary",
} as const;

function StatusCell({ row }: { row: RecurringEarningRow }) {
  const queryClient = useQueryClient();

  return (
    <EditableStatusBadge<RecurringEarningStatus>
      status={row.status as RecurringEarningStatus}
      options={recurringEarningStatusChoices.filter((choice) => choice.value !== "Completed")}
      variants={earningStatusVariants}
      disabled={row.status === "Completed"}
      disabledReason="Completed earnings reached their lifetime cap and cannot be reactivated."
      onStatusChange={async (status) => {
        await updateRecurringEarning(earningStatusInput(row, status));
        await queryClient.invalidateQueries({ queryKey: ["recurring-earning-list"] });
        toast.success(status === "Paused" ? "Earning paused" : "Earning resumed");
      }}
    />
  );
}

export function getColumns(): ColumnDef<RecurringEarningRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusCell row={row.original} />,
      size: 120,
      meta: { apiField: "status" },
    },
    {
      id: "worker",
      header: "Driver",
      cell: ({ row }) => (
        <span className="text-xs font-medium">
          {row.original.worker
            ? `${row.original.worker.firstName} ${row.original.worker.lastName}`.trim()
            : "—"}
        </span>
      ),
      size: 180,
    },
    {
      id: "payCode",
      header: "Code",
      cell: ({ row }) => (
        <span className="text-xs">
          <span className="font-mono font-medium">{row.original.payCode?.code ?? "—"}</span>
          {row.original.payCode?.name && (
            <span className="ml-1.5 text-muted-foreground">{row.original.payCode.name}</span>
          )}
        </span>
      ),
      size: 170,
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => <span className="text-xs">{row.original.description}</span>,
      size: 220,
      meta: { apiField: "description" },
    },
    {
      accessorKey: "frequency",
      header: "Frequency",
      cell: ({ row }) => (
        <span className="text-xs">
          {row.original.frequency === "EverySettlement" ? "Every settlement" : "Monthly"}
        </span>
      ),
      size: 110,
      meta: { apiField: "frequency" },
    },
    {
      accessorKey: "amountMinor",
      header: () => <div className="text-right">Amount</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay value={row.original.amountMinor} currency={row.original.currencyCode} />
        </div>
      ),
      size: 100,
      meta: { apiField: "amountMinor" },
    },
    {
      id: "progress",
      header: () => <div className="text-right">Paid / Cap</div>,
      cell: ({ row }) => {
        const cap = row.original.totalCapMinor;
        return (
          <div className="text-right text-xs tabular-nums">
            <AmountDisplay
              value={row.original.paidToDateMinor}
              currency={row.original.currencyCode}
            />
            {cap != null && (
              <span className="text-muted-foreground">
                {" "}
                / <AmountDisplay value={cap} currency={row.original.currencyCode} />
              </span>
            )}
          </div>
        );
      },
      size: 160,
    },
  ];
}
