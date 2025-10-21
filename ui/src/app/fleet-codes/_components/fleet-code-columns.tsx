import {
  DataTableColorColumn,
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { statusChoices } from "@/lib/choices";
import type { FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<FleetCodeSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status} />;
      },
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "code",
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => {
        const { code, color } = row.original;

        return <DataTableColorColumn text={code} color={color} />;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.description}
          truncateLength={100}
        />
      ),
      size: 100,
      minSize: 100,
      maxSize: 500,
      meta: {
        apiField: "description",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "manager",
      header: "Manager",
      cell: ({ row }) => {
        const { manager } = row.original;
        if (!manager) return <p className="text-muted-foreground">-</p>;
        return <p>{manager.name}</p>;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "manager.name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        return (
          <HoverCardTimestamp
            className="shrink-0"
            timestamp={row.original.createdAt}
          />
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "createdAt",
        label: "Created At",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
