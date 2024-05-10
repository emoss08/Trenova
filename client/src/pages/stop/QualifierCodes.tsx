import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { QualifierCodeEditDialog } from "@/components/qualifier-code-edit-table-dialog";
import { QualifierCodeDialog } from "@/components/qualifier-code-table-dialog";
import { tableStatusChoices } from "@/lib/choices";
import { truncateText } from "@/lib/utils";
import { type QualifierCode } from "@/types/stop";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<QualifierCode>[] = [
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
    cell: ({ row }) => truncateText(row.original.description as string, 30),
  },
];

const filters: FilterConfig<QualifierCode>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function QualifierCodes() {
  return (
    <DataTable
      queryKey="qualifierCodes"
      columns={columns}
      link="/qualifier-codes/"
      name="Qualifier Codes"
      exportModelName="qualifier_codes"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={QualifierCodeDialog}
      TableEditSheet={QualifierCodeEditDialog}
      addPermissionName="qualifiercode.add"
    />
  );
}
