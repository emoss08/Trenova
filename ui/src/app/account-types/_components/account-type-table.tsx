import { DataTable } from "@/components/data-table/data-table";
import { AccountTypeSchema } from "@/lib/schemas/account-type-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./account-type-columns";
import { CreateAccountTypeModal } from "./account-type-create-modal";
import { EditAccountTypeModal } from "./account-type-edit-modal";

export default function AccountTypesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<AccountTypeSchema>
      resource={Resource.AccountType}
      name="Account Type"
      link="/account-types/"
      queryKey="account-type-list"
      exportModelName="account-type"
      TableModal={CreateAccountTypeModal}
      TableEditModal={EditAccountTypeModal}
      columns={columns}
      config={{
        enableFiltering: true,
        enableSorting: true,
        enableMultiSort: true,
        maxFilters: 5,
        maxSorts: 3,
        searchDebounce: 300,
        showFilterUI: true,
        showSortUI: true,
      }}
      useEnhancedBackend={true}
    />
  );
}
