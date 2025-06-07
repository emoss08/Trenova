import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const DedicatedLaneTable = lazy(
  () => import("./_components/dedicated-lane-table"),
);

export function DedicatedLane() {
  return (
    <>
      <MetaTags title="Dedicated Lanes" description="Dedicated Lanes" />
      <LazyComponent>
        <DedicatedLaneTable />
      </LazyComponent>
    </>
  );
}
