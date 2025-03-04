import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { StatusBadge, WorkerTypeBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { generateDateOnlyString, getTodayDate, toDate } from "@/lib/date";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<WorkerSchema>[] {
  const columnHelper = createColumnHelper<WorkerSchema>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    {
      id: "select",
      header: ({ table }) => {
        const isAllSelected = table.getIsAllPageRowsSelected();
        const isSomeSelected = table.getIsSomePageRowsSelected();

        return (
          <Checkbox
            data-slot="select-all"
            checked={isAllSelected || (isSomeSelected && "indeterminate")}
            onCheckedChange={(checked) =>
              table.toggleAllPageRowsSelected(!!checked)
            }
            aria-label="Select all"
          />
        );
      },
      cell: ({ row }) => (
        <Checkbox
          data-slot="select-row"
          checked={row.getIsSelected()}
          onCheckedChange={(checked) => row.toggleSelected(!!checked)}
          aria-label="Select row"
        />
      ),
      size: 50,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Status" />
      ),
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status} />;
      },
    },
    createEntityColumn(columnHelper, "firstName", {
      accessorKey: "firstName",
      getHeaderText: "Details",
      getId: (worker) => worker.id,
      getDisplayText: (worker) => `${worker.firstName} ${worker.lastName}`,
    }),
    {
      accessorKey: "type",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Type" />
      ),
      cell: ({ row }) => {
        const type = row.original.type;
        return <WorkerTypeBadge type={type} />;
      },
    },
    {
      accessorKey: "profile.licenseExpiry",
      id: "licenseExpiry",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="License Expiry" />
      ),
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
    },
    commonColumns.createdAt,
  ];
}
