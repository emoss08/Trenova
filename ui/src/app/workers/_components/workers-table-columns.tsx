import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { StatusBadge, WorkerTypeBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { statusChoices, workerTypeChoices } from "@/lib/choices";
import { generateDateOnlyString, getTodayDate, toDate } from "@/lib/date";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<WorkerSchema>[] {
  const columnHelper = createColumnHelper<WorkerSchema>();
  const commonColumns = createCommonColumns<WorkerSchema>();

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
      accessorKey: "firstName",
      header: "First Name",
      cell: (info) => {
        return <p>{info.getValue() as string}</p>;
      },
      meta: {
        apiField: "firstName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "lastName",
      header: "Last Name",
      cell: (info) => {
        return <p>{info.getValue() as string}</p>;
      },
      meta: {
        apiField: "lastName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "type",
      header: "Worker Type",
      cell: ({ row }) => {
        const type = row.original.type;
        return <WorkerTypeBadge type={type} />;
      },
      meta: {
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: workerTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    columnHelper.display({
      id: "licenseExpiry",
      header: "License Expiry",
      cell: ({ row }) => {
        const licenseExpiry = row.original.profile?.licenseExpiry;
        const date = toDate(licenseExpiry ?? undefined);
        const today = getTodayDate();

        return (
          <Badge
            variant={
              licenseExpiry && licenseExpiry < today ? "inactive" : "active"
            }
          >
            {date ? generateDateOnlyString(date) : "N/A"}
          </Badge>
        );
      },
    }),
    commonColumns.createdAt,
  ];
}
