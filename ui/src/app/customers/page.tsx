import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import CustomersDataTable from "./_components/customer-table";

export function Customers() {
  return (
    <>
      <MetaTags title="Customers" description="Customers" />
      <SuspenseLoader>
        <CustomersDataTable />
      </SuspenseLoader>
    </>
  );
}
