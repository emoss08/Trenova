import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { lazy } from "react";
// import HazardousMaterialTable from "./_components/hazardous-material-table";

const HazmatSegregationRuleTable = lazy(
  () => import("./_components/hazmat-segregation-rule-table"),
);

export function HazmatSegregationRules() {
  return (
    <>
      <MetaTags
        title="Hazmat Segregation Rules"
        description="Hazmat Segregation Rules"
      />
      <SuspenseLoader>
        <HazmatSegregationRuleTable />
      </SuspenseLoader>
    </>
  );
}
