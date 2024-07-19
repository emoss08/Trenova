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

import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { LocationTableSheet } from "@/components/location/location-table-dialog";
import { LocationTableEditSheet } from "@/components/location/location-table-edit-dialog";
import { tableStatusChoices } from "@/lib/choices";
import { upperFirst } from "@/lib/utils";
import { type Location } from "@/types/location";
import { type FilterConfig } from "@/types/tables";
import type { ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<Location>[] = [
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
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    id: "locationCategory",
    accessorFn: (row) => row.locationCategory?.name,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Location Category" />
    ),
    cell: ({ row }) => {
      if (row.original.locationCategory?.color) {
        return (
          <div className="text-foreground flex items-center space-x-2 text-sm font-medium">
            <div
              className={"mx-2 size-2 rounded-xl"}
              style={{
                backgroundColor: row.original.locationCategory?.color,
              }}
            />
            {row.getValue("locationCategory")}
          </div>
        );
      } else {
        return row.getValue("locationCategory");
      }
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorFn: (row) =>
      `${row.addressLine1} ${row.addressLine2} ${row.city} ${row.state?.name}`,
    accessorKey: "location",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Full Address" />
    ),
    cell: ({ row }) =>
      `${row.original.addressLine1}, ${row.original.addressLine2} ${upperFirst(
        row.original.city,
      )}, ${row.original.state?.name} ${row.original.postalCode}`,
  },
];

const filters: FilterConfig<Location>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function Locations() {
  return (
    <DataTable
      queryKey="locations"
      columns={columns}
      link="/locations/"
      name="Locations"
      exportModelName="locations"
      filterColumn="name"
      tableFacetedFilters={filters}
      TableSheet={LocationTableSheet}
      TableEditSheet={LocationTableEditSheet}
      getRowCanExpand={() => true}
      addPermissionName="location:create"
    />
  );
}
