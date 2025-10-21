import {
  BooleanBadge,
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import { VariableContextBadge } from "@/components/status-badge";
import { variableContextChoices } from "@/lib/choices";
import { VariableSchema } from "@/lib/schemas/variable-schema";
import { ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<VariableSchema>[] {
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
      id: "displayName",
      accessorKey: "displayName",
      header: "Display Name",
      cell: ({ row }) => <p>{row.original.displayName}</p>,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "displayName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "category",
      accessorKey: "category",
      header: "Category",
      cell: ({ row }) => <p>{row.original.category}</p>,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "category",
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
      id: "appliesTo",
      accessorKey: "appliesTo",
      header: "Applies To",
      cell: ({ row }) => (
        <VariableContextBadge value={row.original.appliesTo} />
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "appliesTo",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: variableContextChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "isValidated",
      accessorKey: "isValidated",
      header: "Validated",
      cell: ({ row }) => <BooleanBadge value={row.original.isValidated} />,
      size: 100,
      minSize: 100,
      maxSize: 100,
      meta: {
        apiField: "isValidated",
        filterable: true,
        sortable: true,
        filterType: "boolean",
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
