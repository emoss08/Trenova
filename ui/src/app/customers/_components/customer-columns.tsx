import { StatusBadge } from "@/components/status-badge";
import { statusChoices } from "@/lib/choices";
import type { CustomerSchema } from "@/lib/schemas/customer-schema";
import type { ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<CustomerSchema>[] {
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
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.code}</span>
      ),
    },
    {
      accessorKey: "name",
      header: "Name",
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.name}</p>
          <p className="text-sm text-muted-foreground">
            {row.original.city}, {row.original.postalCode}
          </p>
        </div>
      ),
    },
    {
      accessorKey: "city",
      header: "City",
      meta: {
        apiField: "city",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "postalCode",
      header: "Postal Code",
      meta: {
        apiField: "postalCode",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created",
      meta: {
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      cell: ({ row }) => {
        if (!row.original.createdAt) return "-";
        const date = new Date(row.original.createdAt * 1000);
        return date.toLocaleDateString("en-US", {
          year: "numeric",
          month: "short",
          day: "numeric",
        });
      },
    },
  ];
}
