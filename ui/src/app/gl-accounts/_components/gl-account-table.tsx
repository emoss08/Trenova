import { DataTable } from "@/components/data-table/data-table";
import { GLAccountSchema } from "@/lib/schemas/gl-account-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./gl-account-columns";
import { CreateGLAccountModal } from "./gl-account-create-modal";

export default function GLAccountTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<GLAccountSchema>
      resource={Resource.GLAccount}
      name="GL Accounts"
      link="/gl-accounts/"
      exportModelName="gl-account"
      queryKey="gl-account-list"
      extraSearchParams={{
        includeAccountType: true,
      }}
      TableModal={CreateGLAccountModal}
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
