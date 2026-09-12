import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { EDIMappingProfileRow } from "@/lib/graphql/edi-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getMappingProfileColumns(t: TranslateFn): ColumnDef<EDIMappingProfileRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      size: 240,
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "partner.name",
      header: t("Partner"),
      cell: ({ row }) =>
        row.original.partner ? (
          `${row.original.partner.code} — ${row.original.partner.name}`
        ) : (
          <DataTablePlaceholder />
        ),
      size: 260,
      meta: {
        label: t("Partner"),
        apiField: "ediPartnerId",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => row.original.description || <DataTablePlaceholder />,
      size: 280,
      meta: {
        label: t("Description"),
        apiField: "description",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "mappings",
      header: t("Mappings"),
      cell: ({ row }) => {
        const rowEntries = row.original.entries;
        const count = rowEntries ? rowEntries.length : 0;
        return count > 0 ? (
          <Badge variant="secondary">{count.toLocaleString()}</Badge>
        ) : (
          <DataTablePlaceholder text={t("None")} />
        );
      },
      size: 120,
      meta: {
        label: t("Mappings"),
        apiField: "entries",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "updatedAt",
      header: t("Updated"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt ?? undefined} />,
      size: 180,
      meta: {
        label: t("Updated"),
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
