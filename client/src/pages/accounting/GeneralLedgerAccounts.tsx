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
