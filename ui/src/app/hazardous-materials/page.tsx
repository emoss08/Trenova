import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const HazardousMaterialTable = lazy(
  () => import("./_components/hazardous-material-table"),
);

export function HazardousMaterials() {
  return (
    <>
      <MetaTags title="Hazardous Materials" description="Hazardous Materials" />
      <LazyComponent>
        <HazardousMaterialTable />
      </LazyComponent>
    </>
  );
}
