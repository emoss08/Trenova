import { ChargeTypeDialog } from "@/components/charge-type-dialog";
import { ChargeTypeEditSheet } from "@/components/charge-type-edit-dialog";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { tableStatusChoices } from "@/lib/choices";
import { type ChargeType } from "@/types/billing";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<ChargeType>[] = [
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
  },
  {
    accessorKey: "description",
    header: "Description",
  },
];

const filters: FilterConfig<ChargeType>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function ChargeTypes() {
  return (
    <DataTable
      queryKey="chargeTypes"
      columns={columns}
      link="/charge-types/"
      name="Charge Type"
      exportModelName="charge_types"
      filterColumn="name"
      tableFacetedFilters={filters}
      TableSheet={ChargeTypeDialog}
      TableEditSheet={ChargeTypeEditSheet}
      addPermissionName="chargetype.add"
    />
  );
}
