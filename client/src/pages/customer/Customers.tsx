import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { DataTableColumnExpand } from "@/components/common/table/data-table-expand";
import { CustomerTableSheet } from "@/components/customer-table-dialog";
import { CustomerTableEditSheet } from "@/components/customer-table-edit-dialog";
import { CustomerTableSub } from "@/components/customer-table-sub";
import { type Customer } from "@/types/customer";
import type { ColumnDef, Row } from "@tanstack/react-table";

const renderSubComponent = ({ row }: { row: Row<Customer> }) => {
  return <CustomerTableSub row={row} />;
};

const columns: ColumnDef<Customer>[] = [
  {
    id: "expander",
    footer: (props) => props.column.id,
    header: () => null,
    cell: ({ row }) => {
      return <DataTableColumnExpand row={row} />;
    },
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
      queryKey="customers-table-data"
      columns={columns}
      link="/customers/"
      name="Customers"
      exportModelName="customers"
      filterColumn="code"
      renderSubComponent={renderSubComponent}
      getRowCanExpand={() => true}
      TableSheet={CustomerTableSheet}
      TableEditSheet={CustomerTableEditSheet}
      addPermissionName="create_customer"
    />
  );
}
