import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const CustomersDataTable = lazy(() => import("./_components/customer-table"));

export function Customers() {
  return (
    <>
      <MetaTags title="Customers" description="Customers" />
      <LazyComponent>
        <CustomersDataTable />
      </LazyComponent>
    </>
  );
}
