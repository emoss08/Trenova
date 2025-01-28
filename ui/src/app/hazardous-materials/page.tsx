import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import HazardousMaterialTable from "./_components/hazardous-material-table";

export function HazardousMaterials() {
  return (
    <>
      <MetaTags title="Hazardous Materials" description="Hazardous Materials" />
      <SuspenseLoader>
        <HazardousMaterialTable />
      </SuspenseLoader>
    </>
  );
}
