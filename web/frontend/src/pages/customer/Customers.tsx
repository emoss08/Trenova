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
import { CustomerTableSheet } from "@/components/customer-table-dialog";
import { CustomerTableEditSheet } from "@/components/customer-table-edit-dialog";
import { type Customer } from "@/types/customer";
import type { ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<Customer>[] = [
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
  },
  {
    accessorKey: "code",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Code" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
  },
  {
    accessorKey: "totalShipments",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Total Shipments" />
    ),
  },
  {
    accessorKey: "lastShipDate",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Last Ship Date" />
    ),
  },
  {
    accessorKey: "lastBillDate",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Last Bill Date" />
    ),
  },
];

export default function Customers() {
  return (
    <DataTable
      queryKey="customers"
      columns={columns}
      link="/customers/"
      name="Customers"
      exportModelName="customers"
      filterColumn="code"
      TableSheet={CustomerTableSheet}
      TableEditSheet={CustomerTableEditSheet}
      addPermissionName="customer.add"
    />
  );
}
