import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { recurringDeductionStatusChoices } from "@/lib/choices";
import {
  updateRecurringDeduction,
  type RecurringDeductionRow,
} from "@/lib/graphql/driver-settlement";
import type { RecurringDeductionStatus } from "@trenova/shared/types/driver-pay";
import { useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { toast } from "sonner";

export function deductionStatusInput(row: RecurringDeductionRow, status: RecurringDeductionStatus) {
  return {
    id: row.id,
    version: row.version,
    workerId: row.workerId,
    payCodeId: row.payCodeId,
    escrowAccountId: row.escrowAccountId ?? undefined,
    status,
    frequency: row.frequency,
    description: row.description,
    amountMinor: row.amountMinor,
    totalCapMinor: row.totalCapMinor ?? undefined,
    startDate: row.startDate,
    endDate: row.endDate ?? undefined,
  };
}

const deductionStatusVariants = {
  Active: "active",
  Paused: "warning",
  Completed: "secondary",
} as const;

function StatusCell({ row }: { row: RecurringDeductionRow }) {
  const queryClient = useQueryClient();

  return (
    <EditableStatusBadge<RecurringDeductionStatus>
      status={row.status as RecurringDeductionStatus}
      options={recurringDeductionStatusChoices.filter((choice) => choice.value !== "Completed")}
      variants={deductionStatusVariants}
      disabled={row.status === "Completed"}
      disabledReason="Completed deductions reached their lifetime cap and cannot be reactivated."
      onStatusChange={async (status) => {
        await updateRecurringDeduction(deductionStatusInput(row, status));
        await queryClient.invalidateQueries({ queryKey: ["recurring-deduction-list"] });
        toast.success(status === "Paused" ? "Deduction paused" : "Deduction resumed");
      }}
    />
  );
}

export function getColumns(): ColumnDef<RecurringDeductionRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusCell row={row.original} />,
      size: 120,
      meta: { apiField: "status", label: "Status" },
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
      header: () => <div className="text-right">Deducted / Cap</div>,
      cell: ({ row }) => {
        const cap = row.original.totalCapMinor;
        return (
          <div className="text-right text-xs tabular-nums">
            <AmountDisplay
              value={row.original.deductedToDateMinor}
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
