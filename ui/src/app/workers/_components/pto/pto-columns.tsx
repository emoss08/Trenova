import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import { PTOStatusBadge, PTOTypeBadge } from "@/components/status-badge";
import { ptoStatusChoices, ptoTypeChoices } from "@/lib/choices";
import { type WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<WorkerPTOSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <PTOStatusBadge status={status} />;
      },
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ptoStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "worker.firstName",
      header: "First Name",
      cell: (info) => {
        return <p>{info.getValue() as string}</p>;
      },
      meta: {
        apiField: "worker.firstName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "worker.lastName",
      header: "Last Name",
      cell: (info) => {
        return <p>{info.getValue() as string}</p>;
      },
      meta: {
        apiField: "worker.lastName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => {
        const type = row.original.type;
        return <PTOTypeBadge type={type} />;
      },
      meta: {
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ptoTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created",
      meta: {
        apiField: "createdAt",
        filterable: false,
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
