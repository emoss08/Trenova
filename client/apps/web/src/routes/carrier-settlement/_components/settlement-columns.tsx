import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { CarrierSettlementStatusBadge } from "@trenova/shared/components/status-badge";
import { carrierSettlementStatusChoices } from "@/lib/choices";
import type { CarrierSettlementRow } from "@/lib/graphql/carrier-settlement";
import type { CarrierSettlementStatus } from "@trenova/shared/types/carrier-settlement";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatSettlementDate } from "@trenova/shared/lib/date";

function carrierName(row: CarrierSettlementRow): string {
  if (!row.carrier) return "—";
  return row.carrier.scac ? `${row.carrier.name} (${row.carrier.scac})` : row.carrier.name;
}

export function getColumns(): ColumnDef<CarrierSettlementRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <CarrierSettlementStatusBadge status={row.original.status as CarrierSettlementStatus} />
      ),
      size: 150,
      meta: {
        apiField: "status",
        label: "Status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: carrierSettlementStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "settlementNumber",
      header: "Settlement #",
      cell: ({ row }) => (
        <span className="font-mono text-xs font-medium">{row.original.settlementNumber}</span>
      ),
      size: 150,
      meta: {
        apiField: "settlementNumber",
        label: "Settlement Number",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "carrier",
      header: "Carrier",
      cell: ({ row }) => <span className="text-xs font-medium">{carrierName(row.original)}</span>,
      size: 200,
    },
    {
      accessorKey: "periodEnd",
      header: "Period End",
      cell: ({ row }) => (
        <span className="text-xs">{formatSettlementDate(row.original.periodEnd)}</span>
      ),
      size: 110,
      meta: {
        apiField: "periodEnd",
        label: "Period End",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "payDate",
      header: "Pay Date",
      cell: ({ row }) => (
        <span className="text-xs">{formatSettlementDate(row.original.payDate)}</span>
      ),
      size: 110,
      meta: {
        apiField: "payDate",
        label: "Pay Date",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "shipmentCount",
      header: () => <div className="text-right">Loads</div>,
      cell: ({ row }) => (
        <div className="text-right text-xs tabular-nums">{row.original.shipmentCount}</div>
      ),
      size: 70,
      meta: {
        apiField: "shipmentCount",
        label: "Shipment Count",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "grossCostMinor",
      header: () => <div className="text-right">Gross Cost</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay value={row.original.grossCostMinor} currency={row.original.currencyCode} />
        </div>
      ),
      size: 110,
      meta: {
        apiField: "grossCostMinor",
        label: "Gross Cost Minor",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "adjustmentsMinor",
      header: () => <div className="text-right">Adjustments</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay
            value={row.original.adjustmentsMinor}
            variant="auto"
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 110,
      meta: {
        apiField: "adjustmentsMinor",
        label: "Adjustments Minor",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "netPayableMinor",
      header: () => <div className="text-right">Net Payable</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium">
          <AmountDisplay
            value={row.original.netPayableMinor}
            variant="positive"
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 120,
      meta: {
        apiField: "netPayableMinor",
        label: "Net Payable Minor",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
  ];
}
