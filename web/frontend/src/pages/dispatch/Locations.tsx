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
      addPermissionName="location.add"
    />
  );
}
