import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import {
  DriverSettlementStatusBadge,
  PayeeClassificationBadge,
} from "@trenova/shared/components/status-badge";
import type { DriverSettlementRow } from "@/lib/graphql/driver-settlement";
import type { DriverSettlementStatus, PayeeClassification } from "@trenova/shared/types/driver-pay";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { TriangleAlert } from "lucide-react";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

function formatDate(unix: number): string {
  return formatUnixDateMedium(unix, { fallback: "—" });
}

function workerName(row: DriverSettlementRow): string {
  if (!row.worker) return "—";
  return `${row.worker.firstName} ${row.worker.lastName}`.trim() || "—";
}

export function getColumns(t: TranslateFn): ColumnDef<DriverSettlementRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <DriverSettlementStatusBadge status={row.original.status as DriverSettlementStatus} />
          {row.original.hasExceptions && (
            <TriangleAlert className="size-3.5 text-amber-500" aria-label={t("Has exceptions")} />
          )}
        </div>
      ),
      size: 150,
      meta: { apiField: "status", label: t("Status") },
    },
    {
      accessorKey: "settlementNumber",
      header: t("Settlement #"),
      cell: ({ row }) => (
        <span className="font-mono text-xs font-medium">{row.original.settlementNumber}</span>
      ),
      size: 150,
      meta: { apiField: "settlementNumber", label: t("Settlement Number") },
    },
    {
      id: "worker",
      header: t("Driver"),
      cell: ({ row }) => <span className="text-xs font-medium">{workerName(row.original)}</span>,
      size: 180,
    },
    {
      accessorKey: "classification",
      header: t("Type"),
      cell: ({ row }) => (
        <PayeeClassificationBadge
          classification={row.original.classification as PayeeClassification}
        />
      ),
      size: 130,
      meta: { apiField: "classification", label: t("Classification") },
    },
    {
      accessorKey: "periodEnd",
      header: t("Period End"),
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.periodEnd)}</span>,
      size: 110,
      meta: { apiField: "periodEnd", label: t("Period End") },
    },
    {
      accessorKey: "payDate",
      header: t("Pay Date"),
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.payDate)}</span>,
      size: 110,
      meta: { apiField: "payDate", label: t("Pay Date") },
    },
    {
      accessorKey: "shipmentCount",
      header: () => <div className="text-right">{t("Loads")}</div>,
      cell: ({ row }) => (
        <div className="text-right text-xs tabular-nums">{row.original.shipmentCount}</div>
      ),
      size: 70,
      meta: { apiField: "shipmentCount", label: t("Shipment Count") },
    },
    {
      accessorKey: "grossEarningsMinor",
      header: () => <div className="text-right">{t("Gross")}</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay
            value={row.original.grossEarningsMinor}
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 110,
      meta: { apiField: "grossEarningsMinor", label: t("Gross Earnings Minor") },
    },
    {
      accessorKey: "deductionsMinor",
      header: () => <div className="text-right">{t("Deductions")}</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay
            value={-row.original.deductionsMinor}
            variant="negative"
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 110,
      meta: { apiField: "deductionsMinor", label: t("Deductions Minor") },
    },
    {
      accessorKey: "netPayMinor",
      header: () => <div className="text-right">{t("Net Pay")}</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium">
          <AmountDisplay
            value={row.original.netPayMinor}
            variant="positive"
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 120,
      meta: { apiField: "netPayMinor", label: t("Net Pay Minor") },
    },
  ];
}
