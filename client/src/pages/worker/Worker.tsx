/*
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

import { CommodityDialog } from "@/components/commodities/commodity-dialog";
import { CommodityEditDialog } from "@/components/commodities/commodity-edit-table-dialog";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { BoolStatusBadge } from "@/components/common/table/data-table-components";
import { Badge } from "@/components/ui/badge";
import { Worker } from "@/types/worker";
import { ColumnDef } from "@tanstack/react-table";

function HazmatBadge({ isHazmat }: { isHazmat: string }) {
  return (
    <Badge variant={isHazmat === "Y" ? "default" : "destructive"}>
      {isHazmat === "Y" ? "Yes" : "No"}
    </Badge>
  );
}

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
    accessorKey: "isActive",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Is Active?" />
    ),
    cell: ({ row }) => <BoolStatusBadge status={row.original.isActive} />,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "name",
    accessorFn: (row) => `${row.firstName} ${row.lastName}`,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
  },
];

export default function WorkerPage() {
  return (
    <DataTable
      queryKey="worker-table-data"
      columns={columns}
      link="/workers/"
      name="Worker"
      exportModelName="Worker"
      filterColumn="name"
      TableSheet={CommodityDialog}
      TableEditSheet={CommodityEditDialog}
      addPermissionName="add_worker"
    />
  );
}
