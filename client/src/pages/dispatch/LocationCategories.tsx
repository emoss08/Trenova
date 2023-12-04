/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { LCTableEditDialog } from "@/components/location-categories/lc-table-edit-sheet";
import { LCTableSheet } from "@/components/location-categories/lc-table-sheet";
import { truncateText } from "@/lib/utils";
import { LocationCategory } from "@/types/location";
import { ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<LocationCategory>[] = [
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
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => {
      if (row.original.color) {
        return (
          <div className="flex items-center space-x-2 text-sm font-medium text-gray-900 dark:text-gray-100">
            <div
              className={"h-2 w-2 rounded-xl mx-2"}
              style={{ backgroundColor: row.original.color }}
            />
            {row.original.name}
          </div>
        );
      } else {
        return row.original.name;
      }
    },
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 50),
  },
];

export default function LocationCategories() {
  return (
    <DataTable
      queryKey="location-categories-table-data"
      columns={columns}
      link="/location_categories/"
      name="Location Category"
      exportModelName="LocationCategory"
      filterColumn="name"
      TableSheet={LCTableSheet}
      TableEditSheet={LCTableEditDialog}
      addPermissionName="add_locationcategory"
    />
  );
}
