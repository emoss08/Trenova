import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { WorkerTypeBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { generateDateOnlyString, getTodayDate, toDate } from "@/lib/date";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<WorkerSchema>[] {
  const columnHelper = createColumnHelper<WorkerSchema>();
  const commonColumns = createCommonColumns();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "name",
      header: "Name",
      cell: ({ row }) => {
        const { firstName, lastName } = row.original;
        return <p>{`${firstName} ${lastName}`}</p>;
      },
    }),
    {
      accessorKey: "type",
      header: "Worker Type",
      cell: ({ row }) => {
        const type = row.original.type;
        return <WorkerTypeBadge type={type} />;
      },
    },
    columnHelper.display({
      id: "licenseExpiry",
      header: "License Expiry",
      cell: ({ row }) => {
        const licenseExpiry = row.original.profile.licenseExpiry;
        const date = toDate(licenseExpiry);
        const today = getTodayDate();

        return (
          <Badge variant={licenseExpiry < today ? "inactive" : "active"}>
            {date ? generateDateOnlyString(date) : "N/A"}
          </Badge>
        );
      },
    }),
    commonColumns.createdAt,
  ];
}
