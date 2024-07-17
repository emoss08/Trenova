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
import { DelayCodeEditDialog } from "@/components/delay-code-edit-table-dialog";
import { DelayCodeDialog } from "@/components/delay-code-table-dialog";
import { Badge } from "@/components/ui/badge";
import { tableStatusChoices } from "@/lib/choices";
import { truncateText } from "@/lib/utils";
import { DelayCode } from "@/types/dispatch";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

function CarrierOrDriverBadge({
  carrierOrDriver,
}: {
  carrierOrDriver: boolean;
}) {
  return (
    <Badge variant={carrierOrDriver ? "active" : "inactive"}>
      {carrierOrDriver ? "Yes" : "No"}
    </Badge>
  );
}

const columns: ColumnDef<DelayCode>[] = [
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
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => {
      if (row.original.color) {
        return (
          <div className="flex items-center space-x-2 text-sm font-medium text-foreground">
            <div
              className={"mx-2 size-2 rounded-xl"}
              style={{ backgroundColor: row.original.color }}
            />
            {row.original.code}
          </div>
        );
      } else {
        return row.original.code;
      }
    },
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 25),
  },
  {
    accessorKey: "fCarrierOrDriver",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Fault of Carrier Or Driver"
      />
    ),
    cell: ({ row }) => (
      <CarrierOrDriverBadge carrierOrDriver={row.original.fCarrierOrDriver} />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
];

const filters: FilterConfig<DelayCode>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function DelayCodes() {
  return (
    <DataTable
      queryKey="delayCodes"
      columns={columns}
      link="/delay-codes/"
      name="Delay Code"
      exportModelName="delay_codes"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={DelayCodeDialog}
      TableEditSheet={DelayCodeEditDialog}
      addPermissionName="delaycode.add"
    />
  );
}
