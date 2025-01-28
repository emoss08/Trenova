import { DataTable } from "@/components/data-table/data-table";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { useMemo } from "react";
import { getColumns } from "./customer-columns";

export default function CustomersDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<CustomerSchema>
      name="Customer"
      link="/customers/"
      queryKey="customer-list"
      exportModelName="customer"
      //   TableModal={CreateCustomerModal}
      //   TableEditModal={EditCustomerModal}
      columns={columns}
    />
  );
}
