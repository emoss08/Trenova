/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */



import { CommodityDialog } from "@/components/commodity-dialog";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { WorkerEditDialog } from "@/components/worker/worker-table-edit-dialog";
import { getTodayDate } from "@/lib/date";
import { type Worker } from "@/types/worker";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<Worker>[] = [
  {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
        className="translate-y-[2px]"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
        className="translate-y-[2px]"
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => <StatusBadge status={row.original.status} />,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    id: "code",
    accessorFn: (row) => row.code,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Full Name" />
    ),
    cell: ({ row }) => {
      const initials = `${row.original.firstName?.charAt(
        0,
      )}${row.original.lastName?.charAt(0)}`;
      return (
        <div className="flex items-center">
          <div className="size-11 shrink-0">
            <Avatar className="size-11 flex-none rounded-lg">
              <AvatarImage src={row.original.profilePictureUrl || ""} />
              <AvatarFallback className="size-11 flex-none rounded-lg">
                {initials}
              </AvatarFallback>
            </Avatar>
          </div>
          <div className="ml-4">
            <div className="font-medium">
              {row.original.firstName} {row.original.lastName}{" "}
            </div>
            <div className="text-muted-foreground mt-1">
              {row.original.code}
            </div>
          </div>
        </div>
      );
    },
  },
  {
    accessorKey: "workerType",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Worker Type" />
    ),
  },
  {
    accessorKey: "workerProfile.licenseExpirationDate",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="License Expiration Date" />
    ),
    cell: ({ row }) => {
      // Check if the expiration date is expired.
      const expirationDate = row.original.workerProfile?.licenseExpirationDate;
      if (!expirationDate) {
        return null;
      }

      const today = getTodayDate();

      return (
        <Badge
          withDot={false}
          variant={expirationDate < today ? "inactive" : "active"}
        >
          {expirationDate}
        </Badge>
      );
    },
  },
];

export default function WorkerPage() {
  return (
    <DataTable
      queryKey="workers"
      columns={columns}
      link="/workers/"
      name="Worker"
      exportModelName="workers"
      filterColumn="code"
      TableSheet={CommodityDialog}
      TableEditSheet={WorkerEditDialog}
      addPermissionName="worker.add"
    />
  );
}
