/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { DataTableColumnExpand } from "@/components/common/table/data-table-expand";
import { GeneralLedgerAccountTableEditSheet } from "@/components/general-ledger-account-table-edit-sheet";
import { GeneralLedgerAccountTableSheet } from "@/components/general-ledger-account-table-sheet";
import { GLAccountTableSub } from "@/components/general-ledger-account-table-sub";
import { tableAccountTypeChoices, tableStatusChoices } from "@/lib/choices";
import { type GeneralLedgerAccount } from "@/types/accounting";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const renderSubComponent = () => {
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
    accessorKey: "notes",
    header: "Notes",
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
      queryKey="glAccounts"
      columns={columns}
      link="/general-ledger-accounts/"
      name="Gl Account"
      exportModelName="general_ledger_accounts"
      filterColumn="accountNumber"
      tableFacetedFilters={filters}
      TableSheet={GeneralLedgerAccountTableSheet}
      TableEditSheet={GeneralLedgerAccountTableEditSheet}
      renderSubComponent={renderSubComponent}
      getRowCanExpand={() => true}
      addPermissionName="generalledgeraccount.add"
    />
  );
}
