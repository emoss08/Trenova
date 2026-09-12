import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { fuelCardProviderChoices } from "@/lib/choices";
import type { FuelPurchaseImportBatch } from "@/lib/graphql/fuel-purchase-import";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { FUEL_CARD_PROVIDER_LABELS } from "@trenova/shared/types/fuel-ifta-enums";

/**
 * A run reads like a receipt: who it came from, what it read, and what became of
 * the rows. The held count is the one that needs acting on, so it is the column
 * that stands out.
 */
export function getColumns(t: TranslateFn): ColumnDef<FuelPurchaseImportBatch>[] {
  return [
    {
      accessorKey: "provider",
      header: t("Provider"),
      cell: ({ row }) => (
        <span className="font-medium">
          {FUEL_CARD_PROVIDER_LABELS[row.original.provider] ?? row.original.provider}
        </span>
      ),
      size: 130,
      meta: {
        apiField: "provider",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: fuelCardProviderChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "feedReference",
      header: t("Read"),
      cell: ({ row }) => {
        const reference = row.original.feedReference;
        if (!reference) {
          return <span className="text-muted-foreground">—</span>;
        }

        return (
          <span className="text-muted-foreground font-mono text-xs" title={reference}>
            {reference}
          </span>
        );
      },
      meta: {
        apiField: "feedReference",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "rowCount",
      header: t("Rows"),
      cell: ({ row }) => <span className="font-table tabular-nums">{row.original.rowCount}</span>,
      size: 90,
      meta: { apiField: "rowCount", sortable: true },
    },
    {
      accessorKey: "committedCount",
      header: t("Posted"),
      cell: ({ row }) => (
        <span className="font-table tabular-nums">{row.original.committedCount}</span>
      ),
      size: 90,
      meta: { apiField: "committedCount", sortable: true },
    },
    {
      accessorKey: "errorCount",
      header: t("Held"),
      cell: ({ row }) => {
        const held = row.original.errorCount;
        if (held === 0) {
          return <span className="text-muted-foreground font-table tabular-nums">0</span>;
        }

        return <Badge variant="inactive">{t("{0} waiting", held)}</Badge>;
      },
      size: 120,
      meta: { apiField: "errorCount", sortable: true },
    },
    {
      accessorKey: "createdAt",
      header: t("Ran"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 160,
      meta: { apiField: "createdAt", sortable: true },
    },
  ];
}
