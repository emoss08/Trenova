/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
