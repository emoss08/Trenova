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

import { ACDialog } from "@/components/accessorial-charge-table-dialog";
import { AccessorialChargeTableEditDialog } from "@/components/accessorial-charge-table-edit-dialog";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { Badge } from "@/components/ui/badge";
import { tableStatusChoices } from "@/lib/choices";
import { truncateText, USDollarFormat } from "@/lib/utils";
import { AccessorialCharge } from "@/types/billing";
import { type FilterConfig } from "@/types/tables";
import { fuelMethodChoices } from "@/utils/apps/billing";
import { type ColumnDef } from "@tanstack/react-table";

function DetentionBadge({ isDetention }: { isDetention: boolean }) {
  return (
    <Badge variant={isDetention ? "active" : "inactive"}>
      {isDetention ? "Yes" : "No"}
    </Badge>
  );
}

const columns: ColumnDef<AccessorialCharge>[] = [
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
    accessorKey: "code",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Code" />
    ),
  },
  {
    accessorKey: "method",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Method" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "isDetention",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Is Detention" />
    ),
    cell: ({ row }) => (
      <DetentionBadge isDetention={row.original.isDetention} />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 25),
  },
  {
    accessorKey: "amount",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Charge Amount" />
    ),
    cell: ({ row }) => USDollarFormat(row.original.amount),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
];

const filters: FilterConfig<AccessorialCharge>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
  {
    columnName: "isDetention",
    title: "Detention",
    options: [
      { label: "Yes", value: true },
      { label: "No", value: false },
    ],
  },
  { columnName: "method", title: "Method", options: fuelMethodChoices },
];

export default function AccessorialCharges() {
  return (
    <DataTable
      queryKey="accessorialCharges"
      columns={columns}
      link="/accessorial-charges/"
      name="Accessorial Charge"
      exportModelName="accessorial_charges"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={ACDialog}
      TableEditSheet={AccessorialChargeTableEditDialog}
      addPermissionName="accessorialcharge.add"
    />
  );
}
