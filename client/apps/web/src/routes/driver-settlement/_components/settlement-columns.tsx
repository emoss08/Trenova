import { AmountDisplay } from "@/components/accounting/amount-display";
import { DriverSettlementStatusBadge, PayeeClassificationBadge } from "@/components/status-badge";
import type { DriverSettlementRow } from "@/lib/graphql/driver-settlement";
import type { DriverSettlementStatus, PayeeClassification } from "@/types/driver-pay";
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

function workerName(row: DriverSettlementRow): string {
  if (!row.worker) return "—";
  return `${row.worker.firstName} ${row.worker.lastName}`.trim() || "—";
}

export function getColumns(): ColumnDef<DriverSettlementRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <DriverSettlementStatusBadge status={row.original.status as DriverSettlementStatus} />
          {row.original.hasExceptions && (
            <TriangleAlert className="size-3.5 text-amber-500" aria-label="Has exceptions" />
          )}
        </div>
      ),
      size: 150,
      meta: { apiField: "status", label: "Status" },
    },
    {
      accessorKey: "settlementNumber",
      header: "Settlement #",
      cell: ({ row }) => (
        <span className="font-mono text-xs font-medium">{row.original.settlementNumber}</span>
      ),
      size: 150,
      meta: { apiField: "settlementNumber", label: "Settlement Number" },
    },
    {
      id: "worker",
      header: "Driver",
      cell: ({ row }) => <span className="text-xs font-medium">{workerName(row.original)}</span>,
      size: 180,
    },
    {
      accessorKey: "classification",
      header: "Type",
      cell: ({ row }) => (
        <PayeeClassificationBadge
          classification={row.original.classification as PayeeClassification}
        />
      ),
      size: 130,
      meta: { apiField: "classification", label: "Classification" },
    },
    {
      accessorKey: "periodEnd",
      header: "Period End",
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.periodEnd)}</span>,
      size: 110,
      meta: { apiField: "periodEnd", label: "Period End" },
    },
    {
      accessorKey: "payDate",
      header: "Pay Date",
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.payDate)}</span>,
      size: 110,
      meta: { apiField: "payDate", label: "Pay Date" },
    },
    {
      accessorKey: "shipmentCount",
      header: () => <div className="text-right">Loads</div>,
      cell: ({ row }) => (
        <div className="text-right text-xs tabular-nums">{row.original.shipmentCount}</div>
      ),
      size: 70,
      meta: { apiField: "shipmentCount", label: "Shipment Count" },
    },
    {
      accessorKey: "grossEarningsMinor",
      header: () => <div className="text-right">Gross</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay
            value={row.original.grossEarningsMinor}
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 110,
      meta: { apiField: "grossEarningsMinor", label: "Gross Earnings Minor" },
    },
    {
      accessorKey: "deductionsMinor",
      header: () => <div className="text-right">Deductions</div>,
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
      meta: { apiField: "deductionsMinor", label: "Deductions Minor" },
    },
    {
      accessorKey: "netPayMinor",
      header: () => <div className="text-right">Net Pay</div>,
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
      meta: { apiField: "netPayMinor", label: "Net Pay Minor" },
    },
  ];
}
