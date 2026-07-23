import { AmountDisplay } from "@/components/accounting/amount-display";
import { EscrowAccountStatusBadge } from "@/components/status-badge";
import type { EscrowAccountRow } from "@/lib/graphql/driver-settlement";
import type { EscrowAccountStatus } from "@/types/driver-pay";
import { type ColumnDef } from "@tanstack/react-table";

function formatDate(unix?: number | null): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function getColumns(): ColumnDef<EscrowAccountRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <EscrowAccountStatusBadge status={row.original.status as EscrowAccountStatus} />
      ),
      size: 90,
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
      size: 200,
    },
    {
      accessorKey: "balanceMinor",
      header: () => <div className="text-right">Balance</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium">
          <AmountDisplay value={row.original.balanceMinor} currency={row.original.currencyCode} />
        </div>
      ),
      size: 110,
      meta: { apiField: "balanceMinor" },
    },
    {
      accessorKey: "targetAmountMinor",
      header: () => <div className="text-right">Target</div>,
      cell: ({ row }) => (
        <div className="text-right">
          {row.original.targetAmountMinor > 0 ? (
            <AmountDisplay
              value={row.original.targetAmountMinor}
              currency={row.original.currencyCode}
            />
          ) : (
            <span className="text-xs text-muted-foreground">—</span>
          )}
        </div>
      ),
      size: 110,
      meta: { apiField: "targetAmountMinor" },
    },
    {
      id: "funded",
      header: () => <div className="text-right">Funded</div>,
      cell: ({ row }) => {
        const target = row.original.targetAmountMinor;
        if (target <= 0) return <div className="text-right text-xs text-muted-foreground">—</div>;
        const pct = Math.min(100, Math.round((row.original.balanceMinor / target) * 100));
        return (
          <div className="flex items-center justify-end gap-2">
            <div className="h-1.5 w-16 overflow-hidden rounded-full bg-muted">
              <div className="h-full rounded-full bg-green-500" style={{ width: `${pct}%` }} />
            </div>
            <span className="text-xs tabular-nums">{pct}%</span>
          </div>
        );
      },
      size: 130,
    },
    {
      accessorKey: "annualInterestRate",
      header: () => <div className="text-right">Interest</div>,
      cell: ({ row }) => (
        <div className="text-right text-xs tabular-nums">
          {Number(row.original.annualInterestRate) > 0
            ? `${Number(row.original.annualInterestRate).toFixed(2)}%`
            : "—"}
        </div>
      ),
      size: 90,
      meta: { apiField: "annualInterestRate" },
    },
    {
      accessorKey: "openedDate",
      header: "Opened",
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.openedDate)}</span>,
      size: 110,
      meta: { apiField: "openedDate" },
    },
    {
      accessorKey: "lastInterestAccrualDate",
      header: "Last Interest",
      cell: ({ row }) => (
        <span className="text-xs">{formatDate(row.original.lastInterestAccrualDate)}</span>
      ),
      size: 110,
      meta: { apiField: "lastInterestAccrualDate" },
    },
  ];
}
