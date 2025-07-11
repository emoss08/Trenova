import { LazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const CustomerTable = lazy(() => import("./_components/customer-table"));

export function Customers() {
  return (
    <>
      <MetaTags title="Customers" description="Customers" />
      <FormSaveProvider>
        <LazyComponent>
          <CustomerTable />
        </LazyComponent>
      </FormSaveProvider>
    </>
  );
}
