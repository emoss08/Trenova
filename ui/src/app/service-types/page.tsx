import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const ServiceTypeTable = lazy(() => import("./_components/service-type-table"));

export function ServiceTypes() {
  return (
    <>
      <MetaTags title="Service Types" description="Service Types" />
      <LazyComponent>
        <ServiceTypeTable />
      </LazyComponent>
    </>
  );
}
