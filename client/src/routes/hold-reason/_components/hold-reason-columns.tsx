import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { BooleanBadge } from "@/components/status-badge";
import { holdSeverityChoices, holdTypeChoices } from "@/lib/choices";
import type { HoldReason } from "@/types/hold-reason";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<HoldReason>[] {
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
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => {
        const choice = holdTypeChoices.find(
          (c) => c.value === row.original.type,
        );
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.type
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        label: "Type",
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: holdTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => row.original.code,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        label: "Code",
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "label",
      header: "Label",
      cell: ({ row }) => row.original.label,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        label: "Label",
        apiField: "label",
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
        label: "Description",
        apiField: "description",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "defaultSeverity",
      header: "Default Severity",
      cell: ({ row }) => {
        const choice = holdSeverityChoices.find(
          (c) => c.value === row.original.defaultSeverity,
        );
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.defaultSeverity
        );
      },
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        label: "Default Severity",
        apiField: "defaultSeverity",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: holdSeverityChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "defaultBlocksDispatch",
      header: "Blocks Dispatch",
      cell: ({ row }) => (
        <BooleanBadge value={row.original.defaultBlocksDispatch} />
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        label: "Blocks Dispatch",
        apiField: "defaultBlocksDispatch",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "defaultBlocksDelivery",
      header: "Blocks Delivery",
      cell: ({ row }) => (
        <BooleanBadge value={row.original.defaultBlocksDelivery} />
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        label: "Blocks Delivery",
        apiField: "defaultBlocksDelivery",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "defaultBlocksBilling",
      header: "Blocks Billing",
      cell: ({ row }) => (
        <BooleanBadge value={row.original.defaultBlocksBilling} />
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        label: "Blocks Billing",
        apiField: "defaultBlocksBilling",
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
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
