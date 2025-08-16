import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import {
  BooleanBadge,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { HoldSeverityBadge, HoldTypeBadge } from "@/components/status-badge";
import { holdSeverityChoices, holdTypeChoices } from "@/lib/choices";
import { HoldReasonSchema } from "@/lib/schemas/hold-reason-schema";
import { ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<HoldReasonSchema>[] {
  const commonColumns = createCommonColumns<HoldReasonSchema>();

  return [
    {
      id: "active",
      accessorKey: "active",
      header: "Active",
      cell: ({ row }) => <BooleanBadge value={row.original.active} />,
      size: 100,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "active",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "type",
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => <HoldTypeBadge type={row.original.type} />,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: holdTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "code",
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => <p>{row.original.code}</p>,
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
      id: "label",
      accessorKey: "label",
      header: "Label",
      cell: ({ row }) => <p>{row.original.label}</p>,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "label",
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
      minSize: 400,
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
      id: "defaultSeverity",
      accessorKey: "defaultSeverity",
      header: "Default Severity",
      cell: ({ row }) => (
        <HoldSeverityBadge severity={row.original.defaultSeverity} />
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "defaultSeverity",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: holdSeverityChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "defaultBlocksDispatch",
      accessorKey: "defaultBlocksDispatch",
      header: "Default Blocks Dispatch",
      cell: ({ row }) => (
        <BooleanBadge value={row.original.defaultBlocksDispatch} />
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "defaultBlocksDispatch",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "defaultBlocksDelivery",
      accessorKey: "defaultBlocksDelivery",
      header: "Default Blocks Delivery",
      cell: ({ row }) => (
        <BooleanBadge value={row.original.defaultBlocksDelivery} />
      ),
      meta: {
        apiField: "defaultBlocksDelivery",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "defaultBlocksBilling",
      accessorKey: "defaultBlocksBilling",
      header: "Default Blocks Billing",
      cell: ({ row }) => (
        <BooleanBadge value={row.original.defaultBlocksBilling} />
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "defaultBlocksBilling",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "defaultVisibleToCustomer",
      accessorKey: "defaultVisibleToCustomer",
      header: "Default Visible to Customer",
      cell: ({ row }) => (
        <BooleanBadge value={row.original.defaultVisibleToCustomer} />
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "defaultVisibleToCustomer",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    commonColumns.createdAt,
  ];
}
