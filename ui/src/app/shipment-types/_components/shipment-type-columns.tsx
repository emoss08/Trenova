import {
  DataTableColorColumn,
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { statusChoices } from "@/lib/choices";
import { type ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<ShipmentTypeSchema>[] {
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
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => {
        const { code, color } = row.original;
        return <DataTableColorColumn text={code} color={color} />;
      },
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
      size: 400,
      minSize: 300,
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
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}
