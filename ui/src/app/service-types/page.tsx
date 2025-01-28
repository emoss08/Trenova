import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import ServiceTypesDataTable from "./_components/service-type-table";

export function ServiceTypes() {
  return (
    <>
      <MetaTags title="Service Types" description="Service Types" />
      <SuspenseLoader>
        <ServiceTypesDataTable />
      </SuspenseLoader>
    </>
  );
}
