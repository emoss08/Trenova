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
import { EquipmentStatusBadge } from "@/components/common/table/data-table-components";
import { TrailerDialog } from "@/components/trailer-table-dialog";
import { TrailerEditDialog } from "@/components/trailer-table-edit-dialog";
import { Badge } from "@/components/ui/badge";
import { equipmentStatusChoices, type Trailer } from "@/types/equipment";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

function LastInspectionDate({ lastInspection }: { lastInspection?: string }) {
  return (
    <Badge variant={lastInspection ? "active" : "inactive"}>
      {lastInspection || "Never"}
    </Badge>
  );
}
const columns: ColumnDef<Trailer>[] = [
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
    cell: ({ row }) => <EquipmentStatusBadge status={row.getValue("status")} />,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
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
    accessorFn: (row) => `${row.equipmentType?.code}`,
    header: "Equipment Type",
    cell: ({ row }) => {
      if (row.original.equipmentType?.color) {
        return (
          <div className="text-foreground flex items-center space-x-2 text-sm font-medium">
            <div
              className={"mx-2 size-2 rounded-xl"}
              style={{
                backgroundColor: row.original.equipmentType?.color,
              }}
            />
            {row.original.equipmentType?.code}
          </div>
        );
      } else {
        return row.original.equipmentType?.code;
      }
    },
  },
  {
    accessorKey: "lastInspectionDate",
    header: "Last Inspection Date",
    cell: ({ row }) => (
      <LastInspectionDate lastInspection={row.getValue("lastInspectionDate")} />
    ),
  },
];

const filters: FilterConfig<Trailer>[] = [
  {
    columnName: "status",
    title: "Status",
    options: equipmentStatusChoices,
  },
];

export default function TrailerPage() {
  return (
    <DataTable
      queryKey="trailers"
      columns={columns}
      link="/trailers/"
      name="Trailer"
      exportModelName="trailers"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={TrailerDialog}
      TableEditSheet={TrailerEditDialog}
      addPermissionName="trailer.add"
    />
  );
}
