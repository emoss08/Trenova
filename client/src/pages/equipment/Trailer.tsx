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

import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { EquipmentStatusBadge } from "@/components/common/table/data-table-components";
import { Badge } from "@/components/ui/badge";
import { equipmentStatusChoices, Trailer } from "@/types/equipment";
import { FilterConfig } from "@/types/tables";
import { ColumnDef } from "@tanstack/react-table";
import { TrailerDialog } from "@/components/trailer-table-dialog";
import { TrailerEditDialog } from "@/components/trailer-table-edit-dialog";

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
    accessorKey: "equipTypeName",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Equipment Type" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "timesUsed",
    header: "Times Used",
    cell: ({ row }) =>
      row.original.timesUsed === 0
        ? "Never Used"
        : `Used ${row.original.timesUsed} times`,
  },
  {
    accessorKey: "lastInspection",
    header: "Last Inspection Date",
    cell: ({ row }) => (
      <LastInspectionDate lastInspection={row.getValue("lastInspection")} />
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
      queryKey="trailer-table-data"
      columns={columns}
      link="/trailers/"
      name="Trailer"
      exportModelName="Trailer"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={TrailerDialog}
      TableEditSheet={TrailerEditDialog}
      addPermissionName="add_trailer"
    />
  );
}
