import {
  BooleanBadge,
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import { VariableValueTypeBadge } from "@/components/status-badge";
import { variableValueTypeChoices } from "@/lib/choices";
import { VariableFormatSchema } from "@/lib/schemas/variable-schema";
import { ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<VariableFormatSchema>[] {
  return [
    {
      id: "active",
      accessorKey: "isActive",
      header: "Active",
      cell: ({ row }) => <BooleanBadge value={row.original.isActive} />,
      size: 100,
      minSize: 100,
      maxSize: 100,
      meta: {
        apiField: "isActive",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "name",
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <p>{row.original.name}</p>,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "description",
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
      id: "valuesType",
      accessorKey: "valueType",
      header: "Value Type",
      cell: ({ row }) => (
        <VariableValueTypeBadge valueType={row.original.valueType} />
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "valueType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: variableValueTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "isSystem",
      accessorKey: "isSystem",
      header: "System",
      cell: ({ row }) => <BooleanBadge value={row.original.isSystem} />,
      size: 100,
      minSize: 100,
      maxSize: 100,
      meta: {
        apiField: "isSystem",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
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
