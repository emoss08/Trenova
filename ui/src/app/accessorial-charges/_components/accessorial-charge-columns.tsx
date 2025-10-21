import {
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import {
  AccessorialChargeMethodBadge,
  StatusBadge,
} from "@/components/status-badge";
import { accessorialChargeMethodChoices, statusChoices } from "@/lib/choices";
import { AccessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";
import { USDollarFormat } from "@/lib/utils";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<AccessorialChargeSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <StatusBadge status={status} />;
      },
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
        const code = row.original.code;
        return <p>{code}</p>;
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
      accessorKey: "method",
      header: "Method",
      cell: ({ row }) => {
        const method = row.original.method;
        return <AccessorialChargeMethodBadge method={method} />;
      },
      meta: {
        apiField: "method",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: accessorialChargeMethodChoices,
        defaultFilterOperator: "eq",
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
      accessorKey: "amount",
      header: "Amount",
      cell: ({ row }) => {
        const amount = row.original.amount;
        return <p>{USDollarFormat(amount)}</p>;
      },
      meta: {
        apiField: "amount",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
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
