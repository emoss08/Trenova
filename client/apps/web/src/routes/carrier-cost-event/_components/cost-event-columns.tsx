import type { TranslateFn } from "@trenova/shared/i18n/use-t";
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

export function getColumns(t: TranslateFn): ColumnDef<CarrierCostEventRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <CarrierCostEventStatusBadge status={row.original.status as CarrierCostEventStatus} />
      ),
      size: 110,
      meta: { apiField: "status", label: t("Status") },
    },
    {
      accessorKey: "eventType",
      header: t("Type"),
      cell: ({ row }) => (
        <span className="text-xs">
          {costEventTypeLabel(row.original.eventType as CarrierCostEventType)}
        </span>
      ),
      size: 130,
      meta: { apiField: "eventType", label: t("Event Type") },
    },
    {
      id: "carrier",
      header: t("Carrier"),
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
      header: t("Pro #"),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.proNumber || "—"}</span>,
      size: 140,
      meta: { apiField: "proNumber", label: t("Pro Number") },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <span className="text-muted-foreground text-xs">{row.original.description || "—"}</span>
      ),
      size: 260,
      meta: { apiField: "description", label: t("Description") },
    },
    {
      accessorKey: "eventDate",
      header: t("Accrued"),
      cell: ({ row }) => (
        <span className="text-xs">{formatSettlementDate(row.original.eventDate)}</span>
      ),
      size: 110,
      meta: { apiField: "eventDate", label: t("Event Date") },
    },
    {
      accessorKey: "amountMinor",
      header: () => <div className="text-right">{t("Amount")}</div>,
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
      meta: { apiField: "amountMinor", label: t("Amount Minor") },
    },
  ];
}
