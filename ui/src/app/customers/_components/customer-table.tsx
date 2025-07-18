import { DataTable } from "@/components/data-table/data-table";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./customer-columns";
import { CreateCustomerModal } from "./customer-create-modal";
import { EditCustomerModal } from "./customer-edit-modal";

export default function CustomersDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<CustomerSchema>
      name="Customer"
      resource={Resource.Customer}
      columns={columns}
      link="/customers/"
      queryKey="customers"
      exportModelName="Customer"
      TableModal={CreateCustomerModal}
      TableEditModal={EditCustomerModal}
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
      extraSearchParams={{
        includeState: true,
        includeBillingProfile: true,
        includeEmailProfile: true,
      }}
      defaultSort={[{ field: "createdAt", direction: "desc" }]}
    />
  );
}
