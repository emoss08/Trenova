import { DataTable } from "@/components/data-table/data-table";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { useMemo } from "react";
import { getColumns } from "./customer-columns";
import { CreateCustomerModal } from "./customer-create-modal";
import { EditCustomerModal } from "./customer-edit-modal";

export default function CustomersDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<CustomerSchema>
      name="Customer"
      link="/customers/"
      queryKey="customer-list"
      extraSearchParams={{
        includeBillingProfile: true,
      }}
      exportModelName="customer"
      TableModal={CreateCustomerModal}
      TableEditModal={EditCustomerModal}
      columns={columns}
    />
  );
}
