import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { DivisionCodeDialog } from "@/components/division-code-table-dialog";
import { DivisionCodeEditDialog } from "@/components/division-code-table-edit-dialog";
import { tableStatusChoices } from "@/lib/choices";
import { type DivisionCode } from "@/types/accounting";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<DivisionCode>[] = [
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
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "description",
    header: "Description",
  },
];

const filters: FilterConfig<DivisionCode>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function DivisionCodes() {
  return (
    <DataTable
      addPermissionName="add_divisioncode"
      queryKey="division-code-table-data"
      columns={columns}
      link="/division-codes/"
      name="Division Code"
      exportModelName="DivisionCode"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={DivisionCodeDialog}
      TableEditSheet={DivisionCodeEditDialog}
    />
  );
}
