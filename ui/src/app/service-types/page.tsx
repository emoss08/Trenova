import { MetaTags } from "@/components/meta-tags";
import ServiceTypesDataTable from "./_components/service-type-table";

export function ServiceTypes() {
  return (
    <>
      <MetaTags title="Service Types" description="Service Types" />
      <ServiceTypesDataTable />
    </>
  );
}
