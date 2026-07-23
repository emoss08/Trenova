import { AmountDisplay } from "@/components/accounting/amount-display";
import { SettlementBatchStatusBadge } from "@/components/status-badge";
import type { SettlementBatchRow } from "@/lib/graphql/driver-settlement";
import type { SettlementBatchStatus } from "@/types/driver-pay";
import { type ColumnDef } from "@tanstack/react-table";
import { TriangleAlert } from "lucide-react";

function formatDate(unix: number): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function getColumns(): ColumnDef<SettlementBatchRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <SettlementBatchStatusBadge status={row.original.status as SettlementBatchStatus} />
      ),
      size: 110,
      meta: { apiField: "status" },
    },
    {
      accessorKey: "name",
      header: "Batch",
      cell: ({ row }) => <span className="text-xs font-medium">{row.original.name}</span>,
      size: 220,
      meta: { apiField: "name" },
    },
    {
      accessorKey: "periodStart",
      header: "Period",
      cell: ({ row }) => (
        <span className="text-xs">
          {formatDate(row.original.periodStart)} – {formatDate(row.original.periodEnd)}
        </span>
      ),
      size: 180,
      meta: { apiField: "periodStart" },
    },
    {
      accessorKey: "payDate",
      header: "Pay Date",
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.payDate)}</span>,
      size: 110,
      meta: { apiField: "payDate" },
    },
    {
      accessorKey: "settlementCount",
      header: () => <div className="text-right">Settlements</div>,
      cell: ({ row }) => (
        <div className="text-right text-xs tabular-nums">{row.original.settlementCount}</div>
      ),
      size: 100,
      meta: { apiField: "settlementCount" },
    },
    {
      accessorKey: "exceptionCount",
      header: () => <div className="text-right">Exceptions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-1 text-xs tabular-nums">
          {row.original.exceptionCount > 0 && <TriangleAlert className="size-3.5 text-amber-500" />}
          {row.original.exceptionCount}
        </div>
      ),
      size: 100,
      meta: { apiField: "exceptionCount" },
    },
    {
      accessorKey: "totalGrossMinor",
      header: () => <div className="text-right">Total Gross</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay
            value={row.original.totalGrossMinor}
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 120,
      meta: { apiField: "totalGrossMinor" },
    },
    {
      accessorKey: "totalNetMinor",
      header: () => <div className="text-right">Total Net</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium">
          <AmountDisplay
            value={row.original.totalNetMinor}
            variant="positive"
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 120,
      meta: { apiField: "totalNetMinor" },
    },
  ];
}
