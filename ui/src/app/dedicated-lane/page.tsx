import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { queries } from "@/lib/queries";
import { lazy } from "react";
import { DedicatedLaneSuggestions } from "./_components/dedicated-lane-suggestions";

const DedicatedLaneTable = lazy(
  () => import("./_components/dedicated-lane-table"),
);

export function DedicatedLane() {
  return (
    <>
      <MetaTags title="Dedicated Lanes" description="Dedicated Lanes" />
      <QueryLazyComponent
        queryKey={queries.dedicatedLaneSuggestion.getSuggestions._def}
      >
        <DedicatedLaneSuggestions />
        <DedicatedLaneTable />
      </QueryLazyComponent>
    </>
  );
}
