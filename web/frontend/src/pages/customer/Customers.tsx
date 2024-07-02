import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { CustomerTableSheet } from "@/components/customer-table-dialog";
import { CustomerTableEditSheet } from "@/components/customer-table-edit-dialog";
import { type Customer } from "@/types/customer";
import type { ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<Customer>[] = [
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
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
  },
  {
    accessorKey: "totalShipments",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Total Shipments" />
    ),
  },
  {
    accessorKey: "lastShipDate",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Last Ship Date" />
    ),
  },
  {
    accessorKey: "lastBillDate",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Last Bill Date" />
    ),
  },
];

export default function Customers() {
  return (
    <DataTable
      queryKey="customers"
      columns={columns}
      link="/customers/"
      name="Customers"
      exportModelName="customers"
      filterColumn="code"
      TableSheet={CustomerTableSheet}
      TableEditSheet={CustomerTableEditSheet}
      addPermissionName="customer.add"
    />
  );
}
