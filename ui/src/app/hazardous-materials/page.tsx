import { MetaTags } from "@/components/meta-tags";
import HazardousMaterialTable from "./_components/hazardous-material-table";

export function HazardousMaterials() {
  return (
    <>
      <MetaTags title="Hazardous Materials" description="Hazardous Materials" />
      <HazardousMaterialTable />
    </>
  );
}
