import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { StatusBadge, WorkerTypeBadge } from "@/components/status-badge";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { generateDateOnlyString, getTodayDate, toDate } from "@/lib/date";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<WorkerSchema>[] {
  const columnHelper = createColumnHelper<WorkerSchema>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    commonColumns.selection,
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
    {
      id: "details",
      header: "Details",
      cell: ({ row }) => {
        const initials = `${row.original.firstName.charAt(0)}${row.original.lastName.charAt(0)}`;
        return (
          <div className="flex max-h-[55px] items-center">
            <div className="size-8 shrink-0">
              <Avatar className="size-8 flex-none rounded-lg">
                <AvatarImage src={row.original.profilePictureUrl || ""} />
                <AvatarFallback className="size-8 flex-none rounded-lg border border-muted-foreground/20 bg-sidebar-accent uppercase text-primary">
                  {initials}
                </AvatarFallback>
              </Avatar>
            </div>
            <div className="ml-4">
              <div className="h-4 font-medium">
                {row.original.firstName} {row.original.lastName}
              </div>
              <div className="text-xs text-muted-foreground">
                {row.original.id}
              </div>
            </div>
          </div>
        );
      },
    },
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
