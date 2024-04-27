import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { RevenueCodeDialog } from "@/components/revenue-code-table-dialog";
import { RevenueCodeTableEditDialog } from "@/components/revenue-code-table-edit-dialog";
import { truncateText } from "@/lib/utils";
import { type RevenueCode } from "@/types/accounting";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<RevenueCode>[] = [
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
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 25),
  },
  {
    accessorFn: (row) =>
      `${row.edges?.expenseAccount?.accountNumber || "No Expense Account"}`,
    header: "Expense Account",
  },
  {
    accessorFn: (row) =>
      `${row.edges?.revenueAccount?.accountNumber || "No Revenue Account"}`,
    header: "Revenue Account",
  },
];

export default function RevenueCodes() {
  return (
    <DataTable
      addPermissionName="create_revenuecode"
      queryKey="revenue-code-table-data"
      columns={columns}
      link="/revenue-codes/"
      name="Revenue Code"
      exportModelName="revenue_codes"
      filterColumn="code"
      TableSheet={RevenueCodeDialog}
      TableEditSheet={RevenueCodeTableEditDialog}
    />
  );
}
