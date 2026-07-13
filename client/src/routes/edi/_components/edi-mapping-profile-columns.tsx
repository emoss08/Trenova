import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import type { EDIMappingProfile } from "@/types/edi";
import type { ColumnDef } from "@tanstack/react-table";

export function getMappingProfileColumns(): ColumnDef<EDIMappingProfile>[] {
  return [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      size: 240,
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "partner.name",
      header: "Partner",
      cell: ({ row }) =>
        row.original.partner ? (
          `${row.original.partner.code} — ${row.original.partner.name}`
        ) : (
          <DataTablePlaceholder />
        ),
      size: 260,
      meta: {
        label: "Partner",
        apiField: "ediPartnerId",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => row.original.description || <DataTablePlaceholder />,
      size: 280,
      meta: {
        label: "Description",
        apiField: "description",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "mappings",
      header: "Mappings",
      cell: ({ row }) => {
        const rowEntries = row.original.entries;
        const count = rowEntries ? rowEntries.length : 0;
        return count > 0 ? (
          <Badge variant="secondary">{count.toLocaleString()}</Badge>
        ) : (
          <DataTablePlaceholder text="None" />
        );
      },
      size: 120,
      meta: {
        label: "Mappings",
        apiField: "entries",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "updatedAt",
      header: "Updated",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt ?? undefined} />,
      size: 180,
      meta: {
        label: "Updated",
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
