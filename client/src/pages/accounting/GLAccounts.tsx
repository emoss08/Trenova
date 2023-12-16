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
import { GLTableEditSheet } from "@/components/gl-accounts/gl-table-edit-sheet";
import { GLTableSheet } from "@/components/gl-accounts/gl-table-sheet";
import { GLAccountTableSub } from "@/components/gl-accounts/gl-table-sub";
import { tableAccountTypeChoices } from "@/lib/choices";
import { tableStatusChoices } from "@/lib/constants";
import { GeneralLedgerAccount } from "@/types/accounting";
import { FilterConfig } from "@/types/tables";
import { ColumnDef, Row } from "@tanstack/react-table";
import { StatusBadge } from "@/components/common/table/data-table-components";

const renderSubComponent = ({ row }: { row: Row<GeneralLedgerAccount> }) => {
  // const original = row.original;
  return <GLAccountTableSub />;
};

const columns: ColumnDef<GeneralLedgerAccount>[] = [
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
    accessorKey: "accountNumber",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Account Number" />
    ),
  },
  {
    accessorKey: "accountType",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Account Type" />
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

const filters: FilterConfig<GeneralLedgerAccount>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
  {
    columnName: "accountType",
    title: "Account Type",
    options: tableAccountTypeChoices,
  },
];

export default function GLAccounts() {
  return (
    <DataTable
      queryKey="gl-account-table-data"
      columns={columns}
      link="/gl_accounts/"
      name="Gl Account"
      exportModelName="GeneralLedgerAccount"
      filterColumn="accountNumber"
      tableFacetedFilters={filters}
      TableSheet={GLTableSheet}
      TableEditSheet={GLTableEditSheet}
      renderSubComponent={renderSubComponent}
      getRowCanExpand={() => true}
      addPermissionName="add_generalledgeraccount"
    />
  );
}
