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
import { DataTable, StatusBadge } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { EquipTypeEditSheet } from "@/components/equipment-type/equip-type-edit-table-dialog";
import { EquipTypeDialog } from "@/components/equipment-type/equip-type-table-dialog";
import { equipmentClassChoices } from "@/lib/choices";
import { tableStatusChoices } from "@/lib/constants";
import { truncateText } from "@/lib/utils";
import { EquipmentType } from "@/types/equipment";
import { FilterConfig } from "@/types/tables";
import { ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<EquipmentType>[] = [
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
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 30),
  },
  {
    accessorKey: "equipmentClass",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Equip. Class" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
];

const filters: FilterConfig<EquipmentType>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
  {
    columnName: "equipmentClass",
    title: "Equip. Class",
    options: equipmentClassChoices,
  },
];

export default function EquipmentTypes() {
  return (
    <DataTable
      queryKey="equipment-type-table-data"
      columns={columns}
      link="/equipment_types/"
      name="Equip. Types"
      exportModelName="EquipmentType"
      filterColumn="name"
      tableFacetedFilters={filters}
      TableSheet={EquipTypeDialog}
      TableEditSheet={EquipTypeEditSheet}
      addPermissionName="add_equipmenttype"
    />
  );
}
