import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { SettlementBatchStatusBadge } from "@trenova/shared/components/status-badge";
import type { SettlementBatchRow } from "@/lib/graphql/driver-settlement";
import type { SettlementBatchStatus } from "@trenova/shared/types/driver-pay";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { TriangleAlert } from "lucide-react";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

function formatDate(unix: number): string {
  return formatUnixDateMedium(unix, { fallback: "—" });
}

export function getColumns(t: TranslateFn): ColumnDef<SettlementBatchRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <SettlementBatchStatusBadge status={row.original.status as SettlementBatchStatus} />
      ),
      size: 110,
      meta: { apiField: "status" },
    },
    {
      accessorKey: "name",
      header: t("Batch"),
      cell: ({ row }) => <span className="text-xs font-medium">{row.original.name}</span>,
      size: 220,
      meta: { apiField: "name" },
    },
    {
      accessorKey: "periodStart",
      header: t("Period"),
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
      header: t("Pay Date"),
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.payDate)}</span>,
      size: 110,
      meta: { apiField: "payDate" },
    },
    {
      accessorKey: "settlementCount",
      header: () => <div className="text-right">{t("Settlements")}</div>,
      cell: ({ row }) => (
        <div className="text-right text-xs tabular-nums">{row.original.settlementCount}</div>
      ),
      size: 100,
      meta: { apiField: "settlementCount" },
    },
    {
      accessorKey: "exceptionCount",
      header: () => <div className="text-right">{t("Exceptions")}</div>,
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
      header: () => <div className="text-right">{t("Total Gross")}</div>,
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
      header: () => <div className="text-right">{t("Total Net")}</div>,
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
