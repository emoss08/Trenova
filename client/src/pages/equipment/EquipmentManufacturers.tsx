import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { EquipManuDialog } from "@/components/eqiupment-manufacturer-table-dialog";
import { EquipMenuEditDialog } from "@/components/equipment-manufacturer-edit-table-dialog";
import { tableStatusChoices } from "@/lib/choices";
import { truncateText } from "@/lib/utils";
import { type EquipmentManufacturer } from "@/types/equipment";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<EquipmentManufacturer>[] = [
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
];

const filters: FilterConfig<EquipmentManufacturer>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function EquipmentManufacturers() {
  return (
    <DataTable
      addPermissionName="equipmentmanufacturer.add"
      queryKey="equipment-manufacturer-table-data"
      columns={columns}
      link="/equipment-manufacturers/"
      name="Equip. Manufacturers"
      exportModelName="equipment_manufacturers"
      filterColumn="name"
      tableFacetedFilters={filters}
      TableSheet={EquipManuDialog}
      TableEditSheet={EquipMenuEditDialog}
    />
  );
}
