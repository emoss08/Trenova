import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
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
        <div className="flex flex-col gap-0.5 leading-tight">
          <p className="font-mono text-sm">{row.original.code}</p>
          <p className="text-sm text-muted-foreground">{row.original.name}</p>
        </div>
      ),
    },
    {
      id: "addressLine",
      header: "Address Line",
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "addressLine1",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      cell: ({ row }) => {
        return (
          <p>
            {row.original.addressLine1}, {row.original.city}{" "}
            {row.original.state?.abbreviation}, {row.original.postalCode}
          </p>
        );
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

        return (
          <HoverCardTimestamp
            className="font-table tracking-tight"
            timestamp={row.original.createdAt}
          />
        );
      },
    },
  ];
}
