import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { CarrierCostEventStatusBadge } from "@trenova/shared/components/status-badge";
import { carrierCostEventTypeChoices } from "@/lib/choices";
import type { CarrierCostEventRow } from "@/lib/graphql/carrier-settlement";
import type {
  CarrierCostEventStatus,
  CarrierCostEventType,
} from "@trenova/shared/types/carrier-settlement";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatSettlementDate } from "@trenova/shared/lib/date";

export function costEventTypeLabel(eventType: CarrierCostEventType): string {
  return (
    carrierCostEventTypeChoices.find((choice) => choice.value === eventType)?.label ?? eventType
  );
}

export function getColumns(): ColumnDef<CarrierCostEventRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <CarrierCostEventStatusBadge status={row.original.status as CarrierCostEventStatus} />
      ),
      size: 110,
      meta: { apiField: "status", label: "Status" },
    },
    {
      accessorKey: "eventType",
      header: "Type",
      cell: ({ row }) => (
        <span className="text-xs">
          {costEventTypeLabel(row.original.eventType as CarrierCostEventType)}
        </span>
      ),
      size: 130,
      meta: { apiField: "eventType", label: "Event Type" },
    },
    {
      id: "carrier",
      header: "Carrier",
      cell: ({ row }) => (
        <span className="text-xs font-medium">
          {row.original.carrier
            ? row.original.carrier.scac
              ? `${row.original.carrier.name} (${row.original.carrier.scac})`
              : row.original.carrier.name
            : "—"}
        </span>
      ),
      size: 200,
    },
    {
      accessorKey: "proNumber",
      header: "Pro #",
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.proNumber || "—"}</span>,
      size: 140,
      meta: { apiField: "proNumber", label: "Pro Number" },
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <span className="text-muted-foreground text-xs">{row.original.description || "—"}</span>
      ),
      size: 260,
      meta: { apiField: "description", label: "Description" },
    },
    {
      accessorKey: "eventDate",
      header: "Accrued",
      cell: ({ row }) => (
        <span className="text-xs">{formatSettlementDate(row.original.eventDate)}</span>
      ),
      size: 110,
      meta: { apiField: "eventDate", label: "Event Date" },
    },
    {
      accessorKey: "amountMinor",
      header: () => <div className="text-right">Amount</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium">
          <AmountDisplay
            value={row.original.amountMinor}
            variant="auto"
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 110,
      meta: { apiField: "amountMinor", label: "Amount Minor" },
    },
  ];
}
