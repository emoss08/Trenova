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

import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { DataTableColumnExpand } from "@/components/common/table/data-table-expand";
import { CustomerTableSheet } from "@/components/customer/customer-table-dialog";
import { CustomerTableEditSheet } from "@/components/customer/customer-table-edit-dialog";
import { Customer } from "@/types/customer";
import { ColumnDef, Row } from "@tanstack/react-table";
import { CustomerTableSub } from "@/components/customer/customer-table-sub";
import { StatusBadge } from "@/components/common/table/data-table-components";

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
      exportModelName="Customer"
      filterColumn="code"
      renderSubComponent={renderSubComponent}
      getRowCanExpand={() => true}
      TableSheet={CustomerTableSheet}
      TableEditSheet={CustomerTableEditSheet}
      addPermissionName="add_customer"
    />
  );
}
