import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { BooleanBadge } from "@/components/status-badge";
import { rateTableLookupTypeChoices } from "@/lib/choices";
import type { RateTableRow } from "@/types/rate-table";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<RateTableRow>[] {
  return [
    {
      accessorKey: "active",
      header: "Active",
      cell: ({ row }) => <BooleanBadge value={row.original.active} />,
      size: 100,
      minSize: 100,
      maxSize: 100,
      meta: {
        label: "Active",
        apiField: "active",
        filterable: true,
        sortable: true,
        filterType: "boolean",
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => row.original.name,
      size: 220,
      minSize: 200,
      maxSize: 280,
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "key",
      header: "Key",
      cell: ({ row }) => (
        <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">{row.original.key}</span>
      ),
      size: 200,
      minSize: 180,
      maxSize: 250,
      meta: {
        label: "Key",
        apiField: "key",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "lookupType",
      header: "Lookup Type",
      cell: ({ row }) => {
        const choice = rateTableLookupTypeChoices.find(
          (option) => option.value === row.original.lookupType,
        );
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.lookupType
        );
      },
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        label: "Lookup Type",
        apiField: "lookupType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: rateTableLookupTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description} truncateLength={100} />
      ),
      size: 350,
      minSize: 250,
      maxSize: 450,
      meta: {
        label: "Description",
        apiField: "description",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "createdAt",
        filterable: true,
        sortable: true,
      },
    },
  ];
}
