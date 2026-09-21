import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { CarrierSettlementBatchStatusBadge } from "@trenova/shared/components/status-badge";
import { carrierSettlementBatchStatusChoices } from "@/lib/choices";
import type { CarrierSettlementBatchRow } from "@/lib/graphql/carrier-settlement";
import type { CarrierSettlementBatchStatus } from "@trenova/shared/types/carrier-settlement";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatSettlementDate } from "@trenova/shared/lib/date";

export function getColumns(t: TranslateFn): ColumnDef<CarrierSettlementBatchRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <CarrierSettlementBatchStatusBadge
          status={row.original.status as CarrierSettlementBatchStatus}
        />
      ),
      size: 110,
      meta: {
        apiField: "status",
        label: t("Status"),
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: carrierSettlementBatchStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "name",
      header: t("Batch"),
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      size: 220,
      meta: {
        apiField: "name",
        label: t("Batch name"),
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "periodStart",
      header: t("Period"),
      cell: ({ row }) => (
        <span>
          {formatSettlementDate(row.original.periodStart)} –{" "}
          {formatSettlementDate(row.original.periodEnd)}
        </span>
      ),
      size: 180,
      meta: {
        apiField: "periodStart",
        label: t("Period start"),
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "payDate",
      header: t("Pay date"),
      cell: ({ row }) => <span>{formatSettlementDate(row.original.payDate)}</span>,
      size: 110,
      meta: {
        apiField: "payDate",
        label: t("Pay date"),
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "settlementCount",
      header: () => <div className="text-right">{t("Settlements")}</div>,
      cell: ({ row }) => (
        <div className="text-right tabular-nums">{row.original.settlementCount}</div>
      ),
      size: 100,
      meta: {
        apiField: "settlementCount",
        label: t("Settlement count"),
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "totalGrossMinor",
      header: () => <div className="text-right">{t("Total gross")}</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay
            value={row.original.totalGrossMinor}
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 120,
      meta: {
        apiField: "totalGrossMinor",
        label: t("Total gross minor"),
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "totalNetMinor",
      header: () => <div className="text-right">{t("Total net")}</div>,
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
      meta: {
        apiField: "totalNetMinor",
        label: t("Total net minor"),
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
  ];
}
