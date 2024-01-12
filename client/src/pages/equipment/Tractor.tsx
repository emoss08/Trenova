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
import { TractorDialog } from "@/components/tractors/tractor-table-dialog";
import { TractorTableEditSheet } from "@/components/tractors/tractor-table-edit-dialog";
import { equipmentStatusChoices, Tractor } from "@/types/equipment";
import { FilterConfig } from "@/types/tables";
import { ColumnDef } from "@tanstack/react-table";
import { EquipmentStatusBadge } from "@/components/common/table/data-table-components";

const columns: ColumnDef<Tractor>[] = [
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
];

const filters: FilterConfig<Tractor>[] = [
  {
    columnName: "status",
    title: "Status",
    options: equipmentStatusChoices,
  },
];

export default function TractorPage() {
  return (
    <DataTable
      queryKey="trailer-table-data"
      columns={columns}
      link="/tractors/"
      name="Tractor"
      exportModelName="Tractor"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={TractorDialog}
      TableEditSheet={TractorTableEditSheet}
      addPermissionName="add_tractor"
    />
  );
}
