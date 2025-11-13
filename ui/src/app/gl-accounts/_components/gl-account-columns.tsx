import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { statusChoices } from "@/lib/choices";
import { GLAccountSchema } from "@/lib/schemas/gl-account-schema";
import { USDollarFormat } from "@/lib/utils";
import { ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<GLAccountSchema>[] {
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
      accessorKey: "accountCode",
      header: "Account Code",
      cell: ({ row }) => <p>{row.original.accountCode}</p>,
      meta: {
        apiField: "accountCode",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <p>{row.original.name}</p>,
      meta: {
        apiField: "name",
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
      accessorKey: "creditBalance",
      header: "Credit Balance",
      cell: ({ row }) => {
        const { creditBalance } = row.original;
        const value = creditBalance ? USDollarFormat(creditBalance) : "-";

        return <p>{value}</p>;
      },
      size: 150,
      minSize: 100,
      maxSize: 200,
      meta: {
        apiField: "creditBalance",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
    {
      accessorKey: "debitBalance",
      header: "Debit Balance",
      cell: ({ row }) => {
        const { debitBalance } = row.original;
        const value = debitBalance ? USDollarFormat(debitBalance) : "-";

        return <p>{value}</p>;
      },
      size: 150,
      minSize: 100,
      maxSize: 200,
      meta: {
        apiField: "debitBalance",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
    {
      accessorKey: "currentBalance",
      header: "Current Balance",
      cell: ({ row }) => {
        const { currentBalance } = row.original;
        const value = currentBalance ? USDollarFormat(currentBalance) : "-";

        return <p>{value}</p>;
      },
      size: 150,
      minSize: 100,
      maxSize: 200,
      meta: {
        apiField: "currentBalance",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
  ];
}
