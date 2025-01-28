import { MetaTags } from "@/components/meta-tags";
import CustomersDataTable from "./_components/customer-table";

export function Customers() {
  return (
    <>
      <MetaTags title="Customers" description="Customers" />
      <CustomersDataTable />
    </>
  );
}
